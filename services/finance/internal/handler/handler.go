// Package handler implements finance.v1.FinanceService over Connect (and gRPC, from the same
// type). Authorization lives here: the caller's family_id and user_id come from the verified
// access token, never from the request body.
//
// The rule this package exists to enforce, beyond family scoping, is the private-account
// boundary. A private account belongs to exactly one member; its balance is excluded from the
// household headline, its transactions are invisible to everyone else, and another member's
// private accounts become a count and nothing else. That predicate is written once, in
// internal/db/queries, and every read here passes the caller's own member id into it — so
// forgetting the filter is not a thing a handler can do by omission.
package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/rpc"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

// Tx is the transaction boundary. Implemented by internal/store over a pgx pool, and by the
// fake store in tests.
//
// It takes a callback rather than returning a transaction handle so that there is no way to
// begin one and forget to finish it, and so the Querier bound to the transaction is the only
// one in scope while it is open.
type Tx interface {
	InTx(ctx context.Context, fn func(q db.Querier) error) error
}

// Handler serves finance.v1.FinanceService.
type Handler struct {
	q   db.Querier
	tx  Tx
	bus EventBus
	log *slog.Logger
	now func() time.Time

	// defaultCurrency and defaultTimezone are only used by BootstrapHousehold when the caller
	// sends none. Once finance_settings exists it is the authority and these are never read.
	defaultCurrency string
	defaultTimezone string
}

// Options configures a Handler. Only Queries is required.
type Options struct {
	Queries db.Querier
	// Tx groups the writes that must not half-apply — a transfer's two legs, a bootstrap's
	// whole taxonomy. Nil means every write runs on its own, which is what a zero-valued
	// Options in a test gets.
	Tx  Tx
	Bus EventBus
	Log *slog.Logger
	// Now is injected by tests so budget windows and "today" are deterministic.
	Now             func() time.Time
	DefaultCurrency string
	DefaultTimezone string
}

func New(opts Options) *Handler {
	h := &Handler{
		q:               opts.Queries,
		tx:              opts.Tx,
		bus:             opts.Bus,
		log:             opts.Log,
		now:             opts.Now,
		defaultCurrency: opts.DefaultCurrency,
		defaultTimezone: opts.DefaultTimezone,
	}
	if h.log == nil {
		h.log = slog.Default()
	}
	if h.now == nil {
		h.now = time.Now
	}
	if h.tx == nil {
		h.tx = withoutTx{h.q}
	}
	if h.bus == nil {
		h.bus = noopBus{}
	}
	if h.defaultCurrency == "" {
		h.defaultCurrency = "UAH"
	}
	if h.defaultTimezone == "" {
		h.defaultTimezone = "UTC"
	}
	return h
}

// withoutTx runs the callback on the plain querier, for a Handler built without a Tx.
type withoutTx struct{ q db.Querier }

func (w withoutTx) InTx(_ context.Context, fn func(db.Querier) error) error { return fn(w.q) }

func (h *Handler) internal(ctx context.Context, err error, what string) error {
	return rpc.Internal(ctx, h.log, err, what)
}

/* ------------------------------------------------------------------ identity */

// caller is the verified identity behind a request. Both ids come from the token; nothing in
// this package reads a family or user id out of a request message.
//
// memberID is the same value as userID: finance's member projection is keyed by the auth
// service's user id, so "who spent" and "who is asking" are comparable without a lookup. The
// separate name is what keeps the two roles legible at the call sites that mean one and not
// the other.
type caller struct {
	family string
	user   string
	// email is informational: authorization never reads it, and it is used only to give a
	// member row a name until services/family sends a better one.
	email    string
	familyID pgtype.UUID
	userID   pgtype.UUID
}

func (c caller) memberID() pgtype.UUID { return c.userID }

func (h *Handler) caller(ctx context.Context) (caller, error) {
	claims, err := fmauth.Require(ctx)
	if err != nil {
		return caller{}, err
	}
	if claims.UserID == "" {
		return caller{}, connect.NewError(connect.CodeUnauthenticated,
			errors.New("token has no subject"))
	}
	if claims.FamilyID == "" {
		return caller{}, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("caller belongs to no family"))
	}
	familyID, err := pgconv.UUID(claims.FamilyID)
	if err != nil {
		return caller{}, connect.NewError(connect.CodeUnauthenticated,
			errors.New("token family_id is not a uuid"))
	}
	userID, err := pgconv.UUID(claims.UserID)
	if err != nil {
		return caller{}, connect.NewError(connect.CodeUnauthenticated,
			errors.New("token subject is not a uuid"))
	}
	return caller{
		family: claims.FamilyID, user: claims.UserID, email: claims.Email,
		familyID: familyID, userID: userID,
	}, nil
}

// ensureSelf gives the caller a member row if the projection has not produced one.
//
// The roster is projected from `family.member.*`, and those events carry ids but no name — and
// with no broker they do not arrive at all. Without this, a household that signed up before
// finance was listening would draw an empty member switcher and a feed with no avatars. The
// name is derived from the token's email, which is the only human-readable fact finance has
// about a person; family's own display name overwrites it the moment an event lands.
func (h *Handler) ensureSelf(ctx context.Context, c caller) error {
	if _, err := h.q.GetMember(ctx, db.GetMemberParams{
		FamilyID: c.familyID, UserID: c.userID,
	}); err == nil {
		return nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return h.internal(ctx, err, "get member")
	}

	// The first member of a household is its owner: they are the one who bootstrapped it.
	count, err := h.q.CountMembers(ctx, c.familyID)
	if err != nil {
		return h.internal(ctx, err, "count members")
	}
	role := "member"
	if count == 0 {
		role = "owner"
	}
	name := displayNameFrom(c.email)
	if _, err := h.q.UpsertMember(ctx, db.UpsertMemberParams{
		FamilyID: c.familyID, UserID: c.userID,
		DisplayName: name, Initial: initialOf(name),
		AvatarColorStep: int32(count % avatarColorSteps),
		Role:            role, Status: "active", Email: c.email,
	}); err != nil {
		return h.internal(ctx, err, "upsert member")
	}
	return nil
}

// avatarColorSteps is the width of the design's accent ramp; a member's colour is their
// position in the household modulo it, so the same person keeps the same hue everywhere.
const avatarColorSteps = 8

// displayNameFrom turns "olena@example.com" into "olena". It is a placeholder, not an identity:
// the local part is what a person recognises as themselves when nothing better is known.
func displayNameFrom(email string) string {
	at := strings.IndexByte(email, '@')
	if at <= 0 {
		return trimmed(email)
	}
	return trimmed(email[:at])
}

// initialOf is the single grapheme the avatar chip draws, uppercased once here so every
// surface shows the same letter.
func initialOf(name string) string {
	for _, r := range name {
		return strings.ToUpper(string(r))
	}
	return ""
}

/* ----------------------------------------------------------------- household */

// household is the settings row plus its resolved timezone. Every aggregate needs both: the
// base currency decides which rows are summable, and the timezone decides where a day ends.
type household struct {
	settings db.FinanceSetting
	loc      *time.Location
}

func (hh household) currency() string { return hh.settings.BaseCurrencyCode }

func (h *Handler) household(ctx context.Context, c caller) (household, error) {
	row, err := h.q.GetFinanceSettings(ctx, c.familyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return household{}, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("household finance is not set up yet"))
	}
	if err != nil {
		return household{}, h.internal(ctx, err, "load finance settings")
	}
	// A timezone the database accepted but this binary cannot load (a tzdata skew between
	// deploys) must not fail every read: UTC gives a defensible window, and the misconfigured
	// name is worth a log line.
	loc, lerr := time.LoadLocation(row.Timezone)
	if lerr != nil {
		h.log.WarnContext(ctx, "unknown household timezone",
			slog.String("timezone", row.Timezone), slog.String("error", lerr.Error()))
		loc = time.UTC
	}
	return household{settings: row, loc: loc}, nil
}

// today is the household's calendar day, not the server's.
func (h *Handler) today(hh household) time.Time { return startOfDay(h.now().In(hh.loc)) }

/* ----------------------------------------------------------- input plumbing */

// invalid is the client-facing validation error. These strings are shown verbatim by the app
// (packages/api/src/errors.ts), so they are written for a person.
func invalid(format string, args ...any) error {
	return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf(format, args...))
}

func notFound(what string) error {
	return connect.NewError(connect.CodeNotFound, errors.New(what+" not found"))
}

// requireUUID parses an id that must be present. A blank id and a malformed one are different
// mistakes and get different messages.
func requireUUID(field, s string) (pgtype.UUID, error) {
	if trimmed(s) == "" {
		return pgtype.UUID{}, invalid("%s is required", field)
	}
	u, err := pgconv.UUID(trimmed(s))
	if err != nil {
		return pgtype.UUID{}, invalid("%s is not a uuid: %v", field, err)
	}
	return u, nil
}

// optionalUUID maps an empty string to a NULL uuid. A malformed one is still an error — never
// silently NULL a value the client meant.
func optionalUUID(field, s string) (pgtype.UUID, error) {
	if trimmed(s) == "" {
		return pgtype.UUID{}, nil
	}
	u, err := pgconv.UUID(trimmed(s))
	if err != nil {
		return pgtype.UUID{}, invalid("%s is not a uuid: %v", field, err)
	}
	return u, nil
}

// requireDay parses a YYYY-MM-DD, defaulting to the household's today when empty: the add
// sheet opens on today and an omitted date means exactly that.
func requireDay(field, s string, def time.Time, loc *time.Location) (time.Time, error) {
	if trimmed(s) == "" {
		return def, nil
	}
	d, ok := parseDay(trimmed(s), loc)
	if !ok {
		return time.Time{}, invalid("%s must be a date as YYYY-MM-DD", field)
	}
	return d, nil
}

func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: startOfDay(t), Valid: true}
}

func uuidList(field string, ids []string) ([]pgtype.UUID, error) {
	out := make([]pgtype.UUID, 0, len(ids))
	for _, raw := range ids {
		if trimmed(raw) == "" {
			continue
		}
		u, err := pgconv.UUID(trimmed(raw))
		if err != nil {
			return nil, invalid("%s contains a value that is not a uuid: %v", field, err)
		}
		out = append(out, u)
	}
	return out, nil
}

/* ------------------------------------------------------------------- bounds */

// Bounds on free text a client sends. Counted in runes, not bytes: a Ukrainian note must not
// be worth half an English one.
const (
	maxNameRunes     = 120
	maxNoteRunes     = 1000
	maxMerchantRunes = 120
	maxLabelRunes    = 60
	maxIconRunes     = 40
	maxTitleRunes    = 200
	maxCurrencyRunes = 3
	maxColorStep     = 7
	// maxBatchIDs bounds every repeated-id field: a reorder or a widget refresh is a screen's
	// worth of ids, and an unbounded array is an unbounded query.
	maxBatchIDs = 500

	defaultPageSize = 50
	maxPageSize     = 200
)

// checkText reports the first violation rather than a list: a client that has exceeded one of
// these has a bug, not a form to fix.
func checkText(field, value string, max int) error {
	if len([]rune(value)) > max {
		return invalid("%s must be at most %d characters", field, max)
	}
	return nil
}

func checkColorStep(step int32) error {
	if step < 0 || step > maxColorStep {
		return invalid("color_step must be between 0 and %d", maxColorStep)
	}
	return nil
}

// checkCurrency normalises to the ISO 4217 shape the CHECK constraint enforces. A currency is
// three letters or it is a typo.
func checkCurrency(code string) (string, error) {
	c := trimmed(code)
	if len([]rune(c)) != maxCurrencyRunes {
		return "", invalid("currency_code must be a 3-letter ISO 4217 code")
	}
	// Uppercased once, at the door. Every aggregate compares the code in SQL with `=`, and
	// daySections compares it in Go with `==`, so a "uah" account in a "UAH" household would
	// otherwise show its transactions in the feed and in no total anywhere.
	return strings.ToUpper(c), nil
}

// checkAmount refuses a negative amount rather than taking its absolute value: the sign is the
// transaction type's job, and a negative expense is a client bug worth surfacing.
func checkAmount(m *financev1.Money) (int64, error) {
	if m == nil {
		return 0, invalid("amount is required")
	}
	if m.GetAmountMinor() < 0 {
		return 0, invalid("amount must not be negative")
	}
	return m.GetAmountMinor(), nil
}

// checkMoneyCurrency rejects an amount denominated in a currency other than the one the row is
// stored in. Every write here takes its currency from the account (or the template the amount
// overrides), so a client that sends USD into a UAH account would otherwise have it silently
// recorded as UAH. An empty code means "whatever that row uses" and is accepted.
func checkMoneyCurrency(m *financev1.Money, expected string) error {
	code := trimmed(m.GetCurrencyCode())
	if code == "" {
		return nil
	}
	c, err := checkCurrency(code)
	if err != nil {
		return err
	}
	if !strings.EqualFold(c, expected) {
		return invalid("amount is in %s but this account is in %s", c, expected)
	}
	return nil
}

func pageSize(requested int32) int32 {
	switch {
	case requested <= 0:
		return defaultPageSize
	case requested > maxPageSize:
		return maxPageSize
	default:
		return requested
	}
}

func checkBatch(field string, ids []string) error {
	if len(ids) > maxBatchIDs {
		return invalid("%s must contain at most %d ids", field, maxBatchIDs)
	}
	return nil
}

/* ------------------------------------------------ household & settings RPCs */

// defaultTaxonomy is the seeded set the onboarding screen promises. It lives here rather than
// in SQL because it is product copy, not schema: changing it must not need a migration.
var defaultTaxonomy = []struct {
	name       string
	icon       string
	colorStep  int32
	categories []struct{ name, icon string }
}{
	{"Дім", "house", 0, []struct{ name, icon string }{
		{"Оренда", "key"}, {"Комунальні", "lightning"}, {"Ремонт", "hammer"},
	}},
	{"Їжа", "fork-knife", 1, []struct{ name, icon string }{
		{"Продукти", "basket"}, {"Кафе", "coffee"}, {"Доставка", "moped"},
	}},
	{"Транспорт", "car", 2, []struct{ name, icon string }{
		{"Пальне", "gas-pump"}, {"Таксі", "taxi"}, {"Проїзд", "bus"},
	}},
	{"Здоров'я", "heartbeat", 3, []struct{ name, icon string }{
		{"Ліки", "pill"}, {"Лікар", "stethoscope"},
	}},
	{"Розваги", "confetti", 4, []struct{ name, icon string }{
		{"Підписки", "television"}, {"Кіно", "film-reel"}, {"Подорожі", "airplane"},
	}},
}

// defaultIncomeTaxonomy exists because every screen carries a ДОХОДИ tab: seeding only expense
// groups leaves that tab empty on a brand-new household with no way to file a salary.
var defaultIncomeTaxonomy = []struct {
	name       string
	icon       string
	colorStep  int32
	categories []struct{ name, icon string }
}{
	{"Дохід", "wallet", 5, []struct{ name, icon string }{
		{"Зарплата", "briefcase"}, {"Підробіток", "laptop"}, {"Інше", "gift"},
	}},
}

func (h *Handler) BootstrapHousehold(
	ctx context.Context, req *connect.Request[financev1.BootstrapHouseholdRequest],
) (*connect.Response[financev1.BootstrapHouseholdResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	msg := req.Msg

	// Onboarding is where a household comes into being, so it is also where its first member
	// row does: everything else on screen 02 draws a member.
	if err := h.ensureSelf(ctx, c); err != nil {
		return nil, err
	}

	currency := h.defaultCurrency
	if trimmed(msg.GetBaseCurrencyCode()) != "" {
		currency, err = checkCurrency(msg.GetBaseCurrencyCode())
		if err != nil {
			return nil, err
		}
	}
	timezone := trimmed(msg.GetTimezone())
	if timezone == "" {
		timezone = h.defaultTimezone
	}
	if _, lerr := time.LoadLocation(timezone); lerr != nil {
		return nil, invalid("timezone %q is not an IANA timezone name", timezone)
	}
	weekStart := weekdayFromProto(msg.GetWeekStartsOn())
	if weekStart == "" {
		weekStart = "monday"
	}

	out := &financev1.BootstrapHouseholdResponse{}

	// One transaction: a household that got its settings row but not its taxonomy would show
	// an empty categories screen with no way to tell that bootstrap half-ran.
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		settings, err := q.BootstrapFinanceSettings(ctx, db.BootstrapFinanceSettingsParams{
			FamilyID:         c.familyID,
			BaseCurrencyCode: currency,
			Timezone:         timezone,
			WeekStartsOn:     weekStart,
		})
		if err != nil {
			return err
		}
		out.Settings = toProtoSettings(settings)

		if !msg.GetSeedDefaultTaxonomy() {
			return nil
		}
		// Idempotence: a second bootstrap must not double the taxonomy. Existing groups are
		// the signal that seeding already happened.
		existing, err := q.ListCategoryGroups(ctx, db.ListCategoryGroupsParams{
			FamilyID: c.familyID, IncludeArchived: true,
		})
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			cats, err := q.ListCategories(ctx, db.ListCategoriesParams{
				FamilyID: c.familyID, IncludeArchived: true,
			})
			if err != nil {
				return err
			}
			for _, g := range existing {
				out.Groups = append(out.Groups, toProtoGroup(g))
			}
			for _, cat := range cats {
				out.Categories = append(out.Categories, toProtoCategory(cat))
			}
			return nil
		}

		seed := func(kind string, groups []struct {
			name       string
			icon       string
			colorStep  int32
			categories []struct{ name, icon string }
		}) error {
			for _, g := range groups {
				group, err := q.CreateCategoryGroup(ctx, db.CreateCategoryGroupParams{
					FamilyID: c.familyID, Name: g.name, Kind: kind,
					Icon: g.icon, ColorStep: g.colorStep,
				})
				if err != nil {
					return err
				}
				out.Groups = append(out.Groups, toProtoGroup(group))
				for _, cat := range g.categories {
					created, err := q.CreateCategory(ctx, db.CreateCategoryParams{
						FamilyID: c.familyID, GroupID: group.ID,
						Name: cat.name, Kind: kind, Icon: cat.icon,
					})
					if err != nil {
						return err
					}
					out.Categories = append(out.Categories, toProtoCategory(created))
				}
			}
			return nil
		}
		if err := seed(kindExpense, defaultTaxonomy); err != nil {
			return err
		}
		return seed(kindIncome, defaultIncomeTaxonomy)
	})
	if err != nil {
		return nil, h.internal(ctx, err, "bootstrap household")
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) GetFinanceSettings(
	ctx context.Context, _ *connect.Request[financev1.GetFinanceSettingsRequest],
) (*connect.Response[financev1.GetFinanceSettingsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&financev1.GetFinanceSettingsResponse{
		Settings: toProtoSettings(hh.settings),
	}), nil
}

func (h *Handler) UpdateFinanceSettings(
	ctx context.Context, req *connect.Request[financev1.UpdateFinanceSettingsRequest],
) (*connect.Response[financev1.UpdateFinanceSettingsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	msg := req.Msg

	params := db.UpdateFinanceSettingsParams{
		FamilyID:                      c.familyID,
		OverspendNotificationsEnabled: msg.OverspendNotificationsEnabled,
		PinLockEnabled:                msg.PinLockEnabled,
	}
	if msg.BaseCurrencyCode != nil {
		code, err := checkCurrency(msg.GetBaseCurrencyCode())
		if err != nil {
			return nil, err
		}
		params.BaseCurrencyCode = &code
	}
	if msg.Timezone != nil {
		tz := trimmed(msg.GetTimezone())
		if _, lerr := time.LoadLocation(tz); lerr != nil {
			return nil, invalid("timezone %q is not an IANA timezone name", tz)
		}
		params.Timezone = &tz
	}
	if msg.WeekStartsOn != nil {
		day := weekdayFromProto(msg.GetWeekStartsOn())
		if day == "" {
			return nil, invalid("week_starts_on must name a weekday")
		}
		params.WeekStartsOn = &day
	}

	row, err := h.q.UpdateFinanceSettings(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("household finance is not set up yet"))
	}
	if err != nil {
		return nil, h.internal(ctx, err, "update finance settings")
	}
	return connect.NewResponse(&financev1.UpdateFinanceSettingsResponse{
		Settings: toProtoSettings(row),
	}), nil
}

func (h *Handler) SetOverspendNotifications(
	ctx context.Context, req *connect.Request[financev1.SetOverspendNotificationsRequest],
) (*connect.Response[financev1.SetOverspendNotificationsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	row, err := h.q.SetOverspendNotifications(ctx, db.SetOverspendNotificationsParams{
		FamilyID:                      c.familyID,
		OverspendNotificationsEnabled: req.Msg.GetEnabled(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("household finance is not set up yet"))
	}
	if err != nil {
		return nil, h.internal(ctx, err, "set overspend notifications")
	}
	return connect.NewResponse(&financev1.SetOverspendNotificationsResponse{
		Enabled: row.OverspendNotificationsEnabled,
	}), nil
}

func (h *Handler) ListMembers(
	ctx context.Context, req *connect.Request[financev1.ListMembersRequest],
) (*connect.Response[financev1.ListMembersResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.ensureSelf(ctx, c); err != nil {
		return nil, err
	}
	rows, err := h.q.ListMembers(ctx, db.ListMembersParams{
		FamilyID: c.familyID, IncludePending: req.Msg.GetIncludePending(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list members")
	}
	out := make([]*financev1.Member, 0, len(rows))
	for _, m := range rows {
		out = append(out, toProtoMember(m))
	}
	return connect.NewResponse(&financev1.ListMembersResponse{Members: out}), nil
}

func (h *Handler) GetHouseholdOverview(
	ctx context.Context, req *connect.Request[financev1.GetHouseholdOverviewRequest],
) (*connect.Response[financev1.GetHouseholdOverviewResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	window, err := h.window(req.Msg.GetPeriod(), hh)
	if err != nil {
		return nil, err
	}

	balances, err := h.q.SumFamilyBalances(ctx, db.SumFamilyBalancesParams{
		FamilyID: c.familyID, CurrencyCode: hh.currency(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "sum family balances")
	}

	expenseKind := kindExpense
	total, err := h.q.SumVisibleTransactions(ctx, db.SumVisibleTransactionsParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		FromDate: pgDate(window.from), ToDate: pgDate(window.to),
		CurrencyCode: hh.currency(), Kind: &expenseKind,
		MemberIds: []pgtype.UUID{}, AccountIds: []pgtype.UUID{}, CategoryIds: []pgtype.UUID{},
	})
	if err != nil {
		return nil, h.internal(ctx, err, "sum household expense")
	}

	members, err := h.memberSpending(ctx, c, hh, window, &expenseKind, nil)
	if err != nil {
		return nil, err
	}

	// The private-account counts on the member cards come from the same query that hides the
	// accounts themselves: a count is all this response is allowed to carry.
	hidden, err := h.q.CountHiddenPrivateAccounts(ctx, db.CountHiddenPrivateAccountsParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "count hidden private accounts")
	}
	counts := map[string]int32{}
	for _, row := range hidden {
		counts[pgconv.UUIDString(row.OwnerMemberID)] = row.AccountCount
	}
	// The caller's own private accounts are visible to them, so their card's count comes from
	// the list rather than from the hidden summary.
	own, err := h.q.ListVisibleAccounts(ctx, db.ListVisibleAccountsParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list accounts")
	}
	for _, a := range own {
		if a.Visibility == visibilityPrivate {
			counts[pgconv.UUIDString(a.OwnerMemberID)]++
		}
	}
	for _, ms := range members {
		ms.PrivateAccountCount = counts[ms.GetMember().GetUserId()]
	}

	budgets, err := h.q.ListBudgets(ctx, db.ListBudgetsParams{
		FamilyID: c.familyID, TargetKind: strPtr(targetGroup),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list group budgets")
	}

	return connect.NewResponse(&financev1.GetHouseholdOverviewResponse{
		SharedBalance:                 money(balances.SharedBalanceMinor, hh.currency()),
		SavingsTotal:                  money(balances.SavingsMinor, hh.currency()),
		PeriodExpense:                 money(total.TotalMinor, hh.currency()),
		Members:                       members,
		SharedAccountCount:            balances.SharedAccountCount,
		GroupBudgetCount:              int32(len(budgets)),
		OverspendNotificationsEnabled: hh.settings.OverspendNotificationsEnabled,
	}), nil
}

func strPtr(s string) *string { return &s }

// window resolves a request's Period against the household calendar, turning a parse failure
// into an InvalidArgument the app can show.
func (h *Handler) window(p *financev1.Period, hh household) (dayRange, error) {
	r, err := resolvePeriod(p, h.now(), hh.loc, hh.settings.WeekStartsOn)
	if err != nil {
		return dayRange{}, invalid("%v", err)
	}
	return r, nil
}
