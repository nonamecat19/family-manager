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

type Tx interface {
	InTx(ctx context.Context, fn func(q db.Querier) error) error
}

type Handler struct {
	q   db.Querier
	tx  Tx
	bus EventBus
	log *slog.Logger
	now func() time.Time

	defaultCurrency string
	defaultTimezone string
}

type Options struct {
	Queries         db.Querier
	Tx              Tx
	Bus             EventBus
	Log             *slog.Logger
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

type withoutTx struct{ q db.Querier }

func (w withoutTx) InTx(_ context.Context, fn func(db.Querier) error) error { return fn(w.q) }

func (h *Handler) internal(ctx context.Context, err error, what string) error {
	return rpc.Internal(ctx, h.log, err, what)
}

type caller struct {
	family   string
	user     string
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

func (h *Handler) ensureSelf(ctx context.Context, c caller) error {
	if _, err := h.q.GetMember(ctx, db.GetMemberParams{
		FamilyID: c.familyID, UserID: c.userID,
	}); err == nil {
		return nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return h.internal(ctx, err, "get member")
	}

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

const avatarColorSteps = 8

func displayNameFrom(email string) string {
	at := strings.IndexByte(email, '@')
	if at <= 0 {
		return trimmed(email)
	}
	return trimmed(email[:at])
}

func initialOf(name string) string {
	for _, r := range name {
		return strings.ToUpper(string(r))
	}
	return ""
}

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
	loc, lerr := time.LoadLocation(row.Timezone)
	if lerr != nil {
		h.log.WarnContext(ctx, "unknown household timezone",
			slog.String("timezone", row.Timezone), slog.String("error", lerr.Error()))
		loc = time.UTC
	}
	return household{settings: row, loc: loc}, nil
}

func (h *Handler) today(hh household) time.Time { return startOfDay(h.now().In(hh.loc)) }

func invalid(format string, args ...any) error {
	return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf(format, args...))
}

func notFound(what string) error {
	return connect.NewError(connect.CodeNotFound, errors.New(what+" not found"))
}

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

const (
	maxNameRunes     = 120
	maxNoteRunes     = 1000
	maxMerchantRunes = 120
	maxLabelRunes    = 60
	maxIconRunes     = 40
	maxTitleRunes    = 200
	maxCurrencyRunes = 3
	maxColorStep     = 7
	maxBatchIDs      = 500

	defaultPageSize = 50
	maxPageSize     = 200
)

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

func checkCurrency(code string) (string, error) {
	c := trimmed(code)
	if len([]rune(c)) != maxCurrencyRunes {
		return "", invalid("currency_code must be a 3-letter ISO 4217 code")
	}
	return strings.ToUpper(c), nil
}

func checkAmount(m *financev1.Money) (int64, error) {
	if m == nil {
		return 0, invalid("amount is required")
	}
	if m.GetAmountMinor() < 0 {
		return 0, invalid("amount must not be negative")
	}
	return m.GetAmountMinor(), nil
}

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

func (h *Handler) window(p *financev1.Period, hh household) (dayRange, error) {
	r, err := resolvePeriod(p, h.now(), hh.loc, hh.settings.WeekStartsOn)
	if err != nil {
		return dayRange{}, invalid("%v", err)
	}
	return r, nil
}
