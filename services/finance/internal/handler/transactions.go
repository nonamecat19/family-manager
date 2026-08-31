package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

func (h *Handler) CreateTransaction(
	ctx context.Context, req *connect.Request[financev1.CreateTransactionRequest],
) (*connect.Response[financev1.CreateTransactionResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg

	txType := txTypeFromProto(msg.GetType())
	if txType == kindTransfer {
		return nil, invalid("use TransferBetweenAccounts for a transfer")
	}
	accountID, err := requireUUID("account_id", msg.GetAccountId())
	if err != nil {
		return nil, err
	}
	categoryID, err := optionalUUID("category_id", msg.GetCategoryId())
	if err != nil {
		return nil, err
	}
	amount, err := checkAmount(msg.GetAmount())
	if err != nil {
		return nil, err
	}
	if amount == 0 {
		return nil, invalid("amount must be greater than zero")
	}
	if err := checkText("note", msg.GetNote(), maxNoteRunes); err != nil {
		return nil, err
	}
	if err := checkText("merchant", msg.GetMerchant(), maxMerchantRunes); err != nil {
		return nil, err
	}

	// The account is read through the visibility predicate, so a transaction cannot be written
	// against an account the caller may not see — which would otherwise be a way to discover
	// that one exists.
	account, err := h.visibleAccount(ctx, c, accountID)
	if err != nil {
		return nil, err
	}
	if account.Archived {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("this account is archived; unarchive it before adding transactions"))
	}
	if err := checkMoneyCurrency(msg.GetAmount(), account.CurrencyCode); err != nil {
		return nil, err
	}
	if err := h.checkCategoryKind(ctx, c, categoryID, txType); err != nil {
		return nil, err
	}

	// member_id is who spent, and defaults to the caller: the "Хто" picker opens on them.
	// It is deliberately allowed to name someone else — one member logging a shared purchase
	// on another's behalf is the flow screen 03 draws.
	memberID := c.memberID()
	if trimmed(msg.GetMemberId()) != "" {
		if memberID, err = requireUUID("member_id", msg.GetMemberId()); err != nil {
			return nil, err
		}
	}
	templateID, err := optionalUUID("template_id", msg.GetTemplateId())
	if err != nil {
		return nil, err
	}
	occurred, err := requireDay("occurred_on", msg.GetOccurredOn(), h.today(hh), hh.loc)
	if err != nil {
		return nil, err
	}

	before, err := h.affectedBudgets(ctx, c, hh, categoryID, occurred)
	if err != nil {
		return nil, err
	}

	row, err := h.q.CreateTransaction(ctx, db.CreateTransactionParams{
		FamilyID: c.familyID, Type: txType, AccountID: accountID, CategoryID: categoryID,
		AmountMinor: amount, CurrencyCode: account.CurrencyCode,
		Note: trimmed(msg.GetNote()), Merchant: trimmed(msg.GetMerchant()),
		OccurredOn: pgDate(occurred), MemberID: memberID, CreatedByUserID: c.userID,
		TemplateID: templateID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "create transaction")
	}

	view, err := h.attachGroup(ctx, c, row)
	if err != nil {
		return nil, err
	}
	after, err := h.affectedBudgets(ctx, c, hh, categoryID, occurred)
	if err != nil {
		return nil, err
	}
	h.announceTransactionCreated(ctx, c, view)
	h.announceBudgetChanges(ctx, c, before, after, pgconv.UUIDString(row.ID))

	return connect.NewResponse(&financev1.CreateTransactionResponse{
		Transaction:     toProtoTransaction(view),
		AffectedBudgets: after,
	}), nil
}

func (h *Handler) announceTransactionCreated(ctx context.Context, c caller, v transactionView) {
	h.publish(ctx, subjectTransactionCreated, &financev1.TransactionCreatedEvent{
		FamilyId:      c.family,
		TransactionId: pgconv.UUIDString(v.row.ID),
		MemberId:      pgconv.UUIDString(v.row.MemberID),
		AccountId:     pgconv.UUIDString(v.row.AccountID),
		CategoryId:    pgconv.UUIDString(v.row.CategoryID),
		GroupId:       pgconv.UUIDString(v.groupID),
		Type:          txTypeToProto(v.row.Type),
		Amount:        money(v.row.AmountMinor, v.row.CurrencyCode),
		OccurredOn:    pgconv.DateString(v.row.OccurredOn),
		TemplateId:    pgconv.UUIDString(v.row.TemplateID),
		ActorUserId:   c.user,
		OccurredAt:    h.timestamp(),
	})
}

// attachGroup fills in the category's group for a row that came back from a write, where no
// join ran. The group id is on the wire because every feed row and every event consumer needs
// it, and re-deriving it client-side would need the whole taxonomy.
func (h *Handler) attachGroup(ctx context.Context, c caller, row db.Transaction) (transactionView, error) {
	v := transactionView{row: row}
	if !row.CategoryID.Valid {
		return v, nil
	}
	cat, err := h.q.GetCategory(ctx, db.GetCategoryParams{ID: row.CategoryID, FamilyID: c.familyID})
	if errors.Is(err, pgx.ErrNoRows) {
		return v, nil
	}
	if err != nil {
		return transactionView{}, h.internal(ctx, err, "get category")
	}
	v.groupID = cat.GroupID
	return v, nil
}

func (h *Handler) GetTransaction(
	ctx context.Context, req *connect.Request[financev1.GetTransactionRequest],
) (*connect.Response[financev1.GetTransactionResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requireUUID("transaction_id", req.Msg.GetTransactionId())
	if err != nil {
		return nil, err
	}
	row, err := h.q.GetVisibleTransaction(ctx, db.GetVisibleTransactionParams{
		ID: id, FamilyID: c.familyID, ViewerMemberID: c.memberID(),
	})
	// A transaction paid from another member's private account is invisible, and gets the same
	// answer as one that never existed.
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("transaction")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "get transaction")
	}
	return connect.NewResponse(&financev1.GetTransactionResponse{
		Transaction: toProtoTransaction(viewFromGetTx(row)),
	}), nil
}

func (h *Handler) UpdateTransaction(
	ctx context.Context, req *connect.Request[financev1.UpdateTransactionRequest],
) (*connect.Response[financev1.UpdateTransactionResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	id, err := requireUUID("transaction_id", msg.GetTransactionId())
	if err != nil {
		return nil, err
	}

	existing, err := h.q.GetVisibleTransaction(ctx, db.GetVisibleTransactionParams{
		ID: id, FamilyID: c.familyID, ViewerMemberID: c.memberID(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("transaction")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "get transaction")
	}
	if existing.Type == kindTransfer {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("a transfer cannot be edited; delete it and create a new one"))
	}

	params := db.UpdateTransactionParams{ID: id, FamilyID: c.familyID}
	// The row keeps its currency unless the edit moves it to an account that holds another one.
	currency := existing.CurrencyCode
	if msg.Type != nil {
		newType := txTypeFromProto(msg.GetType())
		if newType == kindTransfer {
			return nil, invalid("a transaction cannot be turned into a transfer")
		}
		params.Type = &newType
	}
	if msg.AccountId != nil {
		accountID, err := requireUUID("account_id", msg.GetAccountId())
		if err != nil {
			return nil, err
		}
		account, err := h.visibleAccount(ctx, c, accountID)
		if err != nil {
			return nil, err
		}
		params.AccountID = accountID
		// Moving a transaction to an account in another currency has to bring a new amount:
		// relabelling 500,00 UAH as 500,00 USD is not a conversion, and this service has no
		// rate it could defend doing one with.
		if account.CurrencyCode != currency && msg.Amount == nil {
			return nil, invalid(
				"this account is in %s: send the amount in %s as well",
				account.CurrencyCode, account.CurrencyCode)
		}
		params.CurrencyCode = &account.CurrencyCode
		currency = account.CurrencyCode
	}
	if msg.CategoryId == nil && params.Type != nil {
		// Only the type is changing, so the category being re-checked is the one already on
		// the row: turning an expense into an income leaves it under an expense category.
		if err := h.checkCategoryKind(ctx, c, existing.CategoryID, *params.Type); err != nil {
			return nil, err
		}
	}
	if msg.CategoryId != nil {
		categoryID, err := optionalUUID("category_id", msg.GetCategoryId())
		if err != nil {
			return nil, err
		}
		// The kind is the edit's own if it is changing one, and the row's otherwise: an edit
		// that moves an expense under an income category is refused either way.
		kind := existing.Type
		if params.Type != nil {
			kind = *params.Type
		}
		if err := h.checkCategoryKind(ctx, c, categoryID, kind); err != nil {
			return nil, err
		}
		params.CategoryID = categoryID
	}
	if msg.Amount != nil {
		amount, err := checkAmount(msg.GetAmount())
		if err != nil {
			return nil, err
		}
		if amount == 0 {
			return nil, invalid("amount must be greater than zero")
		}
		if err := checkMoneyCurrency(msg.GetAmount(), currency); err != nil {
			return nil, err
		}
		params.AmountMinor = &amount
	}
	if msg.Note != nil {
		note := trimmed(msg.GetNote())
		if err := checkText("note", note, maxNoteRunes); err != nil {
			return nil, err
		}
		params.Note = &note
	}
	if msg.Merchant != nil {
		merchant := trimmed(msg.GetMerchant())
		if err := checkText("merchant", merchant, maxMerchantRunes); err != nil {
			return nil, err
		}
		params.Merchant = &merchant
	}
	if msg.OccurredOn != nil {
		day, ok := parseDay(trimmed(msg.GetOccurredOn()), hh.loc)
		if !ok {
			return nil, invalid("occurred_on must be a date as YYYY-MM-DD")
		}
		params.OccurredOn = pgDate(day)
	}
	if msg.MemberId != nil {
		memberID, err := requireUUID("member_id", msg.GetMemberId())
		if err != nil {
			return nil, err
		}
		params.MemberID = memberID
	}

	// The budgets the transaction was in before the edit, and the ones it is in after: an edit
	// that moves a purchase from Їжа to Розваги changes two bars, not one.
	beforeOld, err := h.affectedBudgets(ctx, c, hh, existing.CategoryID, existing.OccurredOn.Time)
	if err != nil {
		return nil, err
	}
	newCategory := existing.CategoryID
	if params.CategoryID.Valid || msg.CategoryId != nil {
		newCategory = params.CategoryID
	}
	beforeNew, err := h.affectedBudgets(ctx, c, hh, newCategory, occurredOnOf(params, existing))
	if err != nil {
		return nil, err
	}

	row, err := h.q.UpdateTransaction(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("transaction")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "update transaction")
	}

	view, err := h.attachGroup(ctx, c, row)
	if err != nil {
		return nil, err
	}
	afterOld, err := h.affectedBudgets(ctx, c, hh, existing.CategoryID, existing.OccurredOn.Time)
	if err != nil {
		return nil, err
	}
	after, err := h.affectedBudgets(ctx, c, hh, row.CategoryID, row.OccurredOn.Time)
	if err != nil {
		return nil, err
	}

	h.publish(ctx, subjectTransactionUpdated, &financev1.TransactionUpdatedEvent{
		FamilyId:           c.family,
		TransactionId:      pgconv.UUIDString(row.ID),
		PreviousAmount:     money(existing.AmountMinor, existing.CurrencyCode),
		PreviousCategoryId: pgconv.UUIDString(existing.CategoryID),
		PreviousMemberId:   pgconv.UUIDString(existing.MemberID),
		PreviousOccurredOn: pgconv.DateString(existing.OccurredOn),
		Amount:             money(row.AmountMinor, row.CurrencyCode),
		CategoryId:         pgconv.UUIDString(row.CategoryID),
		MemberId:           pgconv.UUIDString(row.MemberID),
		OccurredOn:         pgconv.DateString(row.OccurredOn),
		ActorUserId:        c.user,
		OccurredAt:         h.timestamp(),
	})
	h.announceBudgetChanges(ctx, c, beforeOld, afterOld, pgconv.UUIDString(row.ID))
	h.announceBudgetChanges(ctx, c, beforeNew, after, pgconv.UUIDString(row.ID))

	return connect.NewResponse(&financev1.UpdateTransactionResponse{
		Transaction:     toProtoTransaction(view),
		AffectedBudgets: after,
	}), nil
}

// occurredOnOf answers "which day will this transaction sit on after the update", which is the
// day the post-edit budget window has to be evaluated in.
func occurredOnOf(params db.UpdateTransactionParams, existing db.GetVisibleTransactionRow) time.Time {
	if params.OccurredOn.Valid {
		return params.OccurredOn.Time
	}
	return existing.OccurredOn.Time
}

func (h *Handler) DeleteTransaction(
	ctx context.Context, req *connect.Request[financev1.DeleteTransactionRequest],
) (*connect.Response[financev1.DeleteTransactionResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	id, err := requireUUID("transaction_id", req.Msg.GetTransactionId())
	if err != nil {
		return nil, err
	}

	existing, err := h.q.GetVisibleTransaction(ctx, db.GetVisibleTransactionParams{
		ID: id, FamilyID: c.familyID, ViewerMemberID: c.memberID(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("transaction")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "get transaction")
	}

	before, err := h.affectedBudgets(ctx, c, hh, existing.CategoryID, existing.OccurredOn.Time)
	if err != nil {
		return nil, err
	}
	rows, err := h.q.DeleteTransaction(ctx, db.DeleteTransactionParams{ID: id, FamilyID: c.familyID})
	if err != nil {
		return nil, h.internal(ctx, err, "delete transaction")
	}
	if rows == 0 {
		return nil, notFound("transaction")
	}
	after, err := h.affectedBudgets(ctx, c, hh, existing.CategoryID, existing.OccurredOn.Time)
	if err != nil {
		return nil, err
	}

	h.publish(ctx, subjectTransactionDeleted, &financev1.TransactionDeletedEvent{
		FamilyId:      c.family,
		TransactionId: pgconv.UUIDString(existing.ID),
		AccountId:     pgconv.UUIDString(existing.AccountID),
		CategoryId:    pgconv.UUIDString(existing.CategoryID),
		Amount:        money(existing.AmountMinor, existing.CurrencyCode),
		OccurredOn:    pgconv.DateString(existing.OccurredOn),
		ActorUserId:   c.user,
		OccurredAt:    h.timestamp(),
	})
	// A delete can pull a budget back under its limit, which announceBudgetChanges turns into
	// the recovery event the notifications service needs to retract its alert.
	h.announceBudgetChanges(ctx, c, before, after, pgconv.UUIDString(existing.ID))

	return connect.NewResponse(&financev1.DeleteTransactionResponse{AffectedBudgets: after}), nil
}

// ListTransactions is the feed, grouped into day sections server-side: the per-day subtotal
// and the period header have to agree with the page boundary, and only the server knows both.
func (h *Handler) ListTransactions(
	ctx context.Context, req *connect.Request[financev1.ListTransactionsRequest],
) (*connect.Response[financev1.ListTransactionsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	window, err := h.window(msg.GetPeriod(), hh)
	if err != nil {
		return nil, err
	}

	filters, err := h.scopeFilters(msg.GetScope(), msg.GetMemberIds(), msg.GetAccountIds())
	if err != nil {
		return nil, err
	}
	categoryIDs, err := uuidList("category_ids", msg.GetCategoryIds())
	if err != nil {
		return nil, err
	}
	groupIDs, err := uuidList("group_ids", msg.GetGroupIds())
	if err != nil {
		return nil, err
	}
	if err := checkText("query", msg.GetQuery(), maxNameRunes); err != nil {
		return nil, err
	}

	params := db.ListVisibleTransactionsParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		FromDate: pgDate(window.from), ToDate: pgDate(window.to),
		Kind:      kindFilter(msg.GetKind()),
		MemberIds: filters.members, AccountIds: filters.accounts,
		CategoryIds: categoryIDs, GroupIds: groupIDs,
		PageSize: pageSize(msg.GetPageSize()),
		// Transfers belong in the feed on the ЗАГАЛЬНЕ tab — "where did the money go" includes
		// moving it — but never under an expense or income tab, where they would double the
		// month.
		IncludeTransfers: msg.GetKind() == financev1.TransactionKind_TRANSACTION_KIND_UNSPECIFIED,
	}
	if q := trimmed(msg.GetQuery()); q != "" {
		params.Query = &q
	}
	if cursor := trimmed(msg.GetCursor()); cursor != "" {
		date, id, err := decodeCursor(cursor, hh.loc)
		if err != nil {
			return nil, err
		}
		params.CursorDate = pgDate(date)
		params.CursorID = id
	}

	rows, err := h.q.ListVisibleTransactions(ctx, params)
	if err != nil {
		return nil, h.internal(ctx, err, "list transactions")
	}

	// The header total carries EVERY predicate the page carries — group and search included —
	// or the number above the list describes a wider set of transactions than the list itself.
	totalParams := db.SumVisibleTransactionsParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		FromDate: pgDate(window.from), ToDate: pgDate(window.to),
		CurrencyCode: hh.currency(), Kind: kindFilter(msg.GetKind()),
		MemberIds: filters.members, AccountIds: filters.accounts, CategoryIds: categoryIDs,
		GroupIds: groupIDs, Query: params.Query,
	}
	total, err := h.q.SumVisibleTransactions(ctx, totalParams)
	if err != nil {
		return nil, h.internal(ctx, err, "sum transactions")
	}

	out := &financev1.ListTransactionsResponse{
		Days:        daySections(rows, hh.currency(), totalParams.Kind == nil),
		PeriodTotal: money(total.TotalMinor, hh.currency()),
	}
	// A full page means there may be more. A short page cannot have more, so the cursor is
	// cleared rather than handing the app a request that returns nothing.
	if int32(len(rows)) == params.PageSize && len(rows) > 0 {
		last := rows[len(rows)-1]
		out.NextCursor = encodeCursor(pgconv.DateString(last.OccurredOn), pgconv.UUIDString(last.ID))
	}
	return connect.NewResponse(out), nil
}

// daySections groups an already-ordered page by calendar day. Each subtotal is scoped the same
// way the period total is — no transfers, household currency only, the two sides netted when
// no kind filter is set — so a day subtotal and the header mean the same thing.
//
// It sees one PAGE, not the period: a day with more transactions than fit on a page is emitted
// as one section per page, each subtotalling the rows it carries. The app merges sections by
// date as it scrolls, so what the user sees adds up; a single section is not, on its own, a
// promise that the day is complete.
func daySections(
	rows []db.ListVisibleTransactionsRow, currency string, netIncome bool,
) []*financev1.DaySection {
	var out []*financev1.DaySection
	var current *financev1.DaySection
	var dayTotal int64

	flush := func() {
		if current != nil {
			current.DayTotal = money(dayTotal, currency)
			out = append(out, current)
		}
	}
	for _, r := range rows {
		date := pgconv.DateString(r.OccurredOn)
		if current == nil || current.GetDate() != date {
			flush()
			dayTotal = 0
			current = &financev1.DaySection{
				Date:         date,
				WeekdayLabel: r.OccurredOn.Time.Weekday().String(),
			}
		}
		// The subtotal is in the household currency, so a row booked in another one is shown
		// but not added: minor units of two currencies are not the same unit, and summing them
		// would put a number on the section that means nothing.
		if r.Type != kindTransfer && r.CurrencyCode == currency {
			if netIncome && r.Type == kindIncome {
				dayTotal -= r.AmountMinor
			} else {
				dayTotal += r.AmountMinor
			}
		}
		current.Transactions = append(current.Transactions, toProtoTransaction(viewFromTxRow(r)))
	}
	flush()
	return out
}

// scopeFilter is the resolved (member, account) pair every aggregate query takes. The Scope
// switcher and the explicit filter lists are the same thing to SQL, so they are merged once
// here rather than in each RPC.
type scopeFilter struct {
	members  []pgtype.UUID
	accounts []pgtype.UUID
}

func (h *Handler) scopeFilters(scope *financev1.Scope, memberIDs, accountIDs []string) (scopeFilter, error) {
	if err := checkBatch("member_ids", memberIDs); err != nil {
		return scopeFilter{}, err
	}
	if err := checkBatch("account_ids", accountIDs); err != nil {
		return scopeFilter{}, err
	}
	members, err := uuidList("member_ids", memberIDs)
	if err != nil {
		return scopeFilter{}, err
	}
	accounts, err := uuidList("account_ids", accountIDs)
	if err != nil {
		return scopeFilter{}, err
	}

	switch scope.GetKind() {
	case financev1.ScopeKind_SCOPE_KIND_MEMBER:
		id, err := requireUUID("scope.member_id", scope.GetMemberId())
		if err != nil {
			return scopeFilter{}, err
		}
		members = append(members, id)
	case financev1.ScopeKind_SCOPE_KIND_ACCOUNT:
		id, err := requireUUID("scope.account_id", scope.GetAccountId())
		if err != nil {
			return scopeFilter{}, err
		}
		accounts = append(accounts, id)
	}
	return scopeFilter{members: members, accounts: accounts}, nil
}

// The cursor is the feed's sort key, not an offset: an OFFSET moves under a feed that is
// being written to, and a page boundary that shifts shows a row twice or not at all.
func encodeCursor(date, id string) string { return date + "|" + id }

func decodeCursor(cursor string, loc *time.Location) (time.Time, pgtype.UUID, error) {
	date, id, ok := strings.Cut(cursor, "|")
	if !ok {
		return time.Time{}, pgtype.UUID{}, invalid("cursor is malformed")
	}
	day, parsed := parseDay(date, loc)
	if !parsed {
		return time.Time{}, pgtype.UUID{}, invalid("cursor is malformed")
	}
	u, err := pgconv.UUID(id)
	if err != nil {
		return time.Time{}, pgtype.UUID{}, invalid("cursor is malformed")
	}
	return day, u, nil
}

// checkCategoryKind refuses a category from the other half of the taxonomy. Categories are
// created under an expense or an income group and the app only ever offers the matching tree,
// so a mismatch is either a stale screen or a client bug — and the row it would write is
// invisible in every breakdown while still moving a balance.
//
// An unset category is allowed: "не вказано" is a real answer on screen 03.
func (h *Handler) checkCategoryKind(
	ctx context.Context, c caller, categoryID pgtype.UUID, txType string,
) error {
	if !categoryID.Valid {
		return nil
	}
	cat, err := h.q.GetCategory(ctx, db.GetCategoryParams{ID: categoryID, FamilyID: c.familyID})
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound("category")
	}
	if err != nil {
		return h.internal(ctx, err, "get category")
	}
	if cat.Kind != txType {
		return invalid("category %q is an %s category", cat.Name, cat.Kind)
	}
	return nil
}
