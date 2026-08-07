// Package handler implements finance.v1.FinanceService over Connect (and gRPC, from the same
// type).
//
// Two rules run through every method:
//
//   - family_id comes from the verified access token, never from the request. A client cannot
//     name someone else's family, so every query is scoped by construction.
//   - money is (amount_minor, currency_code) and stays integral end to end.
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
	"github.com/nnc/family-manager/libs/go/events"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

// Handler serves finance.v1.FinanceService.
type Handler struct {
	q   db.Querier
	bus EventBus
	log *slog.Logger

	baseCurrency    string
	defaultPageSize int32
	maxPageSize     int32
	now             func() time.Time
}

// Options configures a Handler. Only Queries is required.
type Options struct {
	Queries         db.Querier
	Bus             EventBus
	Log             *slog.Logger
	BaseCurrency    string
	DefaultPageSize int32
	MaxPageSize     int32
	Now             func() time.Time
}

func New(opts Options) *Handler {
	h := &Handler{
		q:               opts.Queries,
		bus:             opts.Bus,
		log:             opts.Log,
		baseCurrency:    opts.BaseCurrency,
		defaultPageSize: opts.DefaultPageSize,
		maxPageSize:     opts.MaxPageSize,
		now:             opts.Now,
	}
	if h.log == nil {
		h.log = slog.Default()
	}
	if h.bus == nil {
		h.bus = noopBus{}
	}
	if h.baseCurrency == "" {
		h.baseCurrency = "EUR"
	}
	if h.defaultPageSize == 0 {
		h.defaultPageSize = 50
	}
	if h.maxPageSize == 0 {
		h.maxPageSize = 200
	}
	if h.now == nil {
		h.now = time.Now
	}
	return h
}

/* ------------------------------------------------------------------ accounts */

func (h *Handler) CreateAccount(
	ctx context.Context, req *connect.Request[financev1.CreateAccountRequest],
) (*connect.Response[financev1.CreateAccountResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}

	name := trimmed(req.Msg.GetName())
	if name == "" {
		return nil, invalid("name is required")
	}

	opening := req.Msg.GetOpeningBalance()
	currency, err := currencyOf(req.Msg.GetCurrencyCode(), opening)
	if err != nil {
		return nil, err
	}

	acc, err := h.q.CreateAccount(ctx, db.CreateAccountParams{
		FamilyID:            familyID,
		Name:                name,
		Type:                accountTypeToStored(req.Msg.GetType()),
		CurrencyCode:        currency,
		OpeningBalanceMinor: opening.GetAmountMinor(),
		Color:               req.Msg.GetColor(),
		Icon:                req.Msg.GetIcon(),
	})
	if err != nil {
		return nil, internal(err, "create account")
	}

	// A brand-new account's balance is exactly its opening balance.
	return connect.NewResponse(&financev1.CreateAccountResponse{
		Account: toProtoAccount(acc, acc.OpeningBalanceMinor),
	}), nil
}

func (h *Handler) ListAccounts(
	ctx context.Context, req *connect.Request[financev1.ListAccountsRequest],
) (*connect.Response[financev1.ListAccountsResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := h.q.ListAccountsWithBalance(ctx, db.ListAccountsWithBalanceParams{
		FamilyID:        familyID,
		IncludeArchived: req.Msg.GetIncludeArchived(),
	})
	if err != nil {
		return nil, internal(err, "list accounts")
	}

	accounts := make([]*financev1.Account, 0, len(rows))
	var total int64
	for _, r := range rows {
		accounts = append(accounts, accountFromBalanceRow(r))
		// Only same-currency accounts are summed. Converting would need a rate this service
		// does not have yet, and a wrong total is worse than an honest partial one.
		if r.CurrencyCode == h.baseCurrency {
			total += r.BalanceMinor
		}
	}

	return connect.NewResponse(&financev1.ListAccountsResponse{
		Accounts: accounts,
		Total:    money(total, h.baseCurrency),
	}), nil
}

func (h *Handler) GetAccount(
	ctx context.Context, req *connect.Request[financev1.GetAccountRequest],
) (*connect.Response[financev1.GetAccountResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requiredUUID(req.Msg.GetId(), "id")
	if err != nil {
		return nil, err
	}

	row, err := h.q.GetAccountWithBalance(ctx, db.GetAccountWithBalanceParams{
		ID: id, FamilyID: familyID,
	})
	if err != nil {
		return nil, notFoundOr(err, "account")
	}

	return connect.NewResponse(&financev1.GetAccountResponse{
		Account: accountFromBalanceRow(db.ListAccountsWithBalanceRow(row)),
	}), nil
}

func (h *Handler) UpdateAccount(
	ctx context.Context, req *connect.Request[financev1.UpdateAccountRequest],
) (*connect.Response[financev1.UpdateAccountResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requiredUUID(req.Msg.GetId(), "id")
	if err != nil {
		return nil, err
	}
	name := trimmed(req.Msg.GetName())
	if name == "" {
		return nil, invalid("name is required")
	}

	if _, err := h.q.UpdateAccount(ctx, db.UpdateAccountParams{
		ID:        id,
		FamilyID:  familyID,
		Name:      name,
		Type:      accountTypeToStored(req.Msg.GetType()),
		Color:     req.Msg.GetColor(),
		Icon:      req.Msg.GetIcon(),
		Archived:  req.Msg.GetArchived(),
		SortOrder: req.Msg.GetSortOrder(),
	}); err != nil {
		return nil, notFoundOr(err, "account")
	}

	// Re-read with the balance so the response carries the same shape as ListAccounts —
	// UPDATE ... RETURNING cannot compute it.
	row, err := h.q.GetAccountWithBalance(ctx, db.GetAccountWithBalanceParams{
		ID: id, FamilyID: familyID,
	})
	if err != nil {
		return nil, internal(err, "read back account")
	}

	return connect.NewResponse(&financev1.UpdateAccountResponse{
		Account: accountFromBalanceRow(db.ListAccountsWithBalanceRow(row)),
	}), nil
}

func (h *Handler) DeleteAccount(
	ctx context.Context, req *connect.Request[financev1.DeleteAccountRequest],
) (*connect.Response[financev1.DeleteAccountResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requiredUUID(req.Msg.GetId(), "id")
	if err != nil {
		return nil, err
	}

	// Deleting an account with history would silently rewrite past reports; archiving is the
	// operation the UI actually wants.
	used, err := h.q.CountAccountTransactions(ctx, id)
	if err != nil {
		return nil, internal(err, "count account transactions")
	}
	if used > 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("account has %d transactions; archive it instead", used))
	}

	rows, err := h.q.DeleteAccount(ctx, db.DeleteAccountParams{ID: id, FamilyID: familyID})
	if err != nil {
		return nil, internal(err, "delete account")
	}
	if rows == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("account not found"))
	}
	return connect.NewResponse(&financev1.DeleteAccountResponse{}), nil
}

/* ---------------------------------------------------------------- categories */

func (h *Handler) CreateCategory(
	ctx context.Context, req *connect.Request[financev1.CreateCategoryRequest],
) (*connect.Response[financev1.CreateCategoryResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}
	name := trimmed(req.Msg.GetName())
	if name == "" {
		return nil, invalid("name is required")
	}

	kind, ok := txTypeToStored(req.Msg.GetKind())
	if !ok || kind == typeTransfer {
		return nil, invalid("kind must be income or expense")
	}

	parentID, err := optionalUUID(req.Msg.GetParentId(), "parent_id")
	if err != nil {
		return nil, err
	}

	cat, err := h.q.CreateCategory(ctx, db.CreateCategoryParams{
		FamilyID: familyID,
		Name:     name,
		Kind:     kind,
		Color:    req.Msg.GetColor(),
		Icon:     req.Msg.GetIcon(),
		ParentID: parentID,
	})
	if err != nil {
		return nil, internal(err, "create category")
	}
	return connect.NewResponse(&financev1.CreateCategoryResponse{Category: toProtoCategory(cat)}), nil
}

func (h *Handler) ListCategories(
	ctx context.Context, req *connect.Request[financev1.ListCategoriesRequest],
) (*connect.Response[financev1.ListCategoriesResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}

	// An unspecified kind means "both", which the query expresses as an empty filter.
	kind, _ := txTypeToStored(req.Msg.GetKind())
	if kind == typeTransfer {
		return nil, invalid("transfer is not a category kind")
	}

	rows, err := h.q.ListCategories(ctx, db.ListCategoriesParams{
		FamilyID:        familyID,
		Kind:            kind,
		IncludeArchived: req.Msg.GetIncludeArchived(),
	})
	if err != nil {
		return nil, internal(err, "list categories")
	}

	out := make([]*financev1.Category, 0, len(rows))
	for _, c := range rows {
		out = append(out, toProtoCategory(c))
	}
	return connect.NewResponse(&financev1.ListCategoriesResponse{Categories: out}), nil
}

func (h *Handler) UpdateCategory(
	ctx context.Context, req *connect.Request[financev1.UpdateCategoryRequest],
) (*connect.Response[financev1.UpdateCategoryResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requiredUUID(req.Msg.GetId(), "id")
	if err != nil {
		return nil, err
	}
	name := trimmed(req.Msg.GetName())
	if name == "" {
		return nil, invalid("name is required")
	}
	parentID, err := optionalUUID(req.Msg.GetParentId(), "parent_id")
	if err != nil {
		return nil, err
	}
	// A category that is its own parent would make the tree walk never terminate.
	if parentID.Valid && pgconv.UUIDString(parentID) == req.Msg.GetId() {
		return nil, invalid("a category cannot be its own parent")
	}

	cat, err := h.q.UpdateCategory(ctx, db.UpdateCategoryParams{
		ID:        id,
		FamilyID:  familyID,
		Name:      name,
		Color:     req.Msg.GetColor(),
		Icon:      req.Msg.GetIcon(),
		ParentID:  parentID,
		Archived:  req.Msg.GetArchived(),
		SortOrder: req.Msg.GetSortOrder(),
	})
	if err != nil {
		return nil, notFoundOr(err, "category")
	}
	return connect.NewResponse(&financev1.UpdateCategoryResponse{Category: toProtoCategory(cat)}), nil
}

func (h *Handler) DeleteCategory(
	ctx context.Context, req *connect.Request[financev1.DeleteCategoryRequest],
) (*connect.Response[financev1.DeleteCategoryResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requiredUUID(req.Msg.GetId(), "id")
	if err != nil {
		return nil, err
	}

	used, err := h.q.CountCategoryTransactions(ctx, id)
	if err != nil {
		return nil, internal(err, "count category transactions")
	}
	if used > 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("category has %d transactions; archive it instead", used))
	}

	rows, err := h.q.DeleteCategory(ctx, db.DeleteCategoryParams{ID: id, FamilyID: familyID})
	if err != nil {
		return nil, internal(err, "delete category")
	}
	if rows == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("category not found"))
	}
	return connect.NewResponse(&financev1.DeleteCategoryResponse{}), nil
}

/* -------------------------------------------------------------- transactions */

func (h *Handler) CreateTransaction(
	ctx context.Context, req *connect.Request[financev1.CreateTransactionRequest],
) (*connect.Response[financev1.CreateTransactionResponse], error) {
	// familyID re-validates the claim, so a token carrying an unusable family_id fails the
	// same way on writes as it does on reads.
	claims, err := fmauth.RequireFamily(ctx)
	if err != nil {
		return nil, err
	}
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}

	input, err := h.validateTransaction(ctx, familyID, transactionInput{
		accountID:        req.Msg.GetAccountId(),
		counterAccountID: req.Msg.GetCounterAccountId(),
		categoryID:       req.Msg.GetCategoryId(),
		txType:           req.Msg.GetType(),
		amount:           req.Msg.GetAmount(),
		occurredOn:       req.Msg.GetOccurredOn(),
	})
	if err != nil {
		return nil, err
	}

	createdBy, err := requiredUUID(claims.UserID, "user id")
	if err != nil {
		return nil, err
	}

	tx, err := h.q.CreateTransaction(ctx, db.CreateTransactionParams{
		FamilyID:         familyID,
		AccountID:        input.account,
		CounterAccountID: input.counterAccount,
		CategoryID:       input.category,
		Type:             input.storedType,
		AmountMinor:      input.amountMinor,
		CurrencyCode:     input.currency,
		Note:             trimmed(req.Msg.GetNote()),
		OccurredOn:       input.date,
		CreatedByUserID:  createdBy,
	})
	if err != nil {
		return nil, internal(err, "create transaction")
	}

	h.publish(ctx, events.SubjectFinanceTransactionCreated, &financev1.TransactionCreatedEvent{
		FamilyId:        claims.FamilyID,
		TransactionId:   pgconv.UUIDString(tx.ID),
		AccountId:       pgconv.UUIDString(tx.AccountID),
		CategoryId:      pgconv.UUIDString(tx.CategoryID),
		Type:            txTypeToProto(tx.Type),
		Amount:          money(tx.AmountMinor, tx.CurrencyCode),
		CreatedByUserId: claims.UserID,
		OccurredAt:      h.timestamp(),
	})

	// After the write, not before: a budget is measured against what is recorded.
	h.announceBudgetCrossings(ctx, familyID, tx, claims.UserID)

	return connect.NewResponse(&financev1.CreateTransactionResponse{
		Transaction: toProtoTransaction(tx),
	}), nil
}

func (h *Handler) GetTransaction(
	ctx context.Context, req *connect.Request[financev1.GetTransactionRequest],
) (*connect.Response[financev1.GetTransactionResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requiredUUID(req.Msg.GetId(), "id")
	if err != nil {
		return nil, err
	}

	tx, err := h.q.GetTransaction(ctx, db.GetTransactionParams{ID: id, FamilyID: familyID})
	if err != nil {
		return nil, notFoundOr(err, "transaction")
	}
	return connect.NewResponse(&financev1.GetTransactionResponse{
		Transaction: toProtoTransaction(tx),
	}), nil
}

func (h *Handler) ListTransactions(
	ctx context.Context, req *connect.Request[financev1.ListTransactionsRequest],
) (*connect.Response[financev1.ListTransactionsResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}

	from, to, err := dateRange(req.Msg.GetRange())
	if err != nil {
		return nil, err
	}
	accountIDs, err := uuidList(req.Msg.GetAccountIds(), "account_ids")
	if err != nil {
		return nil, err
	}
	categoryIDs, err := uuidList(req.Msg.GetCategoryIds(), "category_ids")
	if err != nil {
		return nil, err
	}
	// An unspecified type means "any", which the query expresses as an empty filter.
	storedType, _ := txTypeToStored(req.Msg.GetType())

	params := db.ListTransactionsParams{
		FamilyID:    familyID,
		FromDate:    from,
		ToDate:      to,
		AccountIds:  accountIDs,
		CategoryIds: categoryIDs,
		Type:        storedType,
		Search:      trimmed(req.Msg.GetSearch()),
		PageSize:    h.pageSize(req.Msg.GetPageSize()) + 1, // one extra row answers "is there more?"
	}

	if token := req.Msg.GetPageToken(); token != "" {
		date, id, err := decodeCursor(token)
		if err != nil {
			return nil, invalid("page_token is not a cursor issued by this service")
		}
		cursorDate, err := pgconv.Date(date)
		if err != nil {
			return nil, invalid("page_token is not a cursor issued by this service")
		}
		cursorID, err := pgconv.UUID(id)
		if err != nil {
			return nil, invalid("page_token is not a cursor issued by this service")
		}
		params.UseCursor, params.CursorDate, params.CursorID = true, cursorDate, cursorID
	}

	rows, err := h.q.ListTransactions(ctx, params)
	if err != nil {
		return nil, internal(err, "list transactions")
	}

	limit := int(params.PageSize - 1)
	var next string
	if len(rows) > limit {
		last := rows[limit-1]
		next = encodeCursor(pgconv.DateString(last.OccurredOn), pgconv.UUIDString(last.ID))
		rows = rows[:limit]
	}

	out := make([]*financev1.Transaction, 0, len(rows))
	for _, t := range rows {
		out = append(out, toProtoTransaction(t))
	}

	return connect.NewResponse(&financev1.ListTransactionsResponse{
		Transactions:  out,
		NextPageToken: next,
	}), nil
}

func (h *Handler) UpdateTransaction(
	ctx context.Context, req *connect.Request[financev1.UpdateTransactionRequest],
) (*connect.Response[financev1.UpdateTransactionResponse], error) {
	// familyID re-validates the claim, so a token carrying an unusable family_id fails the
	// same way on writes as it does on reads.
	claims, err := fmauth.RequireFamily(ctx)
	if err != nil {
		return nil, err
	}
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := requiredUUID(req.Msg.GetId(), "id")
	if err != nil {
		return nil, err
	}

	input, err := h.validateTransaction(ctx, familyID, transactionInput{
		accountID:        req.Msg.GetAccountId(),
		counterAccountID: req.Msg.GetCounterAccountId(),
		categoryID:       req.Msg.GetCategoryId(),
		txType:           req.Msg.GetType(),
		amount:           req.Msg.GetAmount(),
		occurredOn:       req.Msg.GetOccurredOn(),
	})
	if err != nil {
		return nil, err
	}

	tx, err := h.q.UpdateTransaction(ctx, db.UpdateTransactionParams{
		ID:               id,
		FamilyID:         familyID,
		AccountID:        input.account,
		CounterAccountID: input.counterAccount,
		CategoryID:       input.category,
		Type:             input.storedType,
		AmountMinor:      input.amountMinor,
		CurrencyCode:     input.currency,
		Note:             trimmed(req.Msg.GetNote()),
		OccurredOn:       input.date,
	})
	if err != nil {
		return nil, notFoundOr(err, "transaction")
	}

	h.publish(ctx, events.SubjectFinanceTransactionUpdated, &financev1.TransactionUpdatedEvent{
		FamilyId:      claims.FamilyID,
		TransactionId: pgconv.UUIDString(tx.ID),
		OccurredAt:    h.timestamp(),
	})

	// Editing an amount upward can cross a limit just as adding a transaction can.
	h.announceBudgetCrossings(ctx, familyID, tx, claims.UserID)

	return connect.NewResponse(&financev1.UpdateTransactionResponse{
		Transaction: toProtoTransaction(tx),
	}), nil
}

func (h *Handler) DeleteTransaction(
	ctx context.Context, req *connect.Request[financev1.DeleteTransactionRequest],
) (*connect.Response[financev1.DeleteTransactionResponse], error) {
	claims, err := fmauth.RequireFamily(ctx)
	if err != nil {
		return nil, err
	}
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requiredUUID(req.Msg.GetId(), "id")
	if err != nil {
		return nil, err
	}

	rows, err := h.q.DeleteTransaction(ctx, db.DeleteTransactionParams{
		ID: id, FamilyID: familyID,
	})
	if err != nil {
		return nil, internal(err, "delete transaction")
	}
	if rows == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("transaction not found"))
	}

	h.publish(ctx, events.SubjectFinanceTransactionDeleted, &financev1.TransactionDeletedEvent{
		FamilyId:      claims.FamilyID,
		TransactionId: req.Msg.GetId(),
		OccurredAt:    h.timestamp(),
	})

	return connect.NewResponse(&financev1.DeleteTransactionResponse{}), nil
}

/* ------------------------------------------------------------------- reports */

func (h *Handler) GetSummary(
	ctx context.Context, req *connect.Request[financev1.GetSummaryRequest],
) (*connect.Response[financev1.GetSummaryResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}
	from, to, err := dateRange(req.Msg.GetRange())
	if err != nil {
		return nil, err
	}
	accountIDs, err := uuidList(req.Msg.GetAccountIds(), "account_ids")
	if err != nil {
		return nil, err
	}

	row, err := h.q.GetSummary(ctx, db.GetSummaryParams{
		FamilyID:   familyID,
		FromDate:   from,
		ToDate:     to,
		AccountIds: accountIDs,
	})
	if err != nil {
		return nil, internal(err, "get summary")
	}

	// The balance is "now", not "in the period": it is the answer to "how much do we have?".
	accounts, err := h.q.ListAccountsWithBalance(ctx, db.ListAccountsWithBalanceParams{
		FamilyID: familyID, IncludeArchived: false,
	})
	if err != nil {
		return nil, internal(err, "list accounts")
	}
	var balance int64
	for _, a := range accounts {
		if a.CurrencyCode == h.baseCurrency {
			balance += a.BalanceMinor
		}
	}

	return connect.NewResponse(&financev1.GetSummaryResponse{
		Income:  money(row.IncomeMinor, h.baseCurrency),
		Expense: money(row.ExpenseMinor, h.baseCurrency),
		Net:     money(row.IncomeMinor-row.ExpenseMinor, h.baseCurrency),
		Balance: money(balance, h.baseCurrency),
	}), nil
}

func (h *Handler) GetCategoryBreakdown(
	ctx context.Context, req *connect.Request[financev1.GetCategoryBreakdownRequest],
) (*connect.Response[financev1.GetCategoryBreakdownResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}
	from, to, err := dateRange(req.Msg.GetRange())
	if err != nil {
		return nil, err
	}
	accountIDs, err := uuidList(req.Msg.GetAccountIds(), "account_ids")
	if err != nil {
		return nil, err
	}

	// A breakdown of "everything" is meaningless: income and expense do not share a pie.
	storedType, ok := txTypeToStored(req.Msg.GetType())
	if !ok || storedType == typeTransfer {
		return nil, invalid("type must be income or expense")
	}

	rows, err := h.q.GetCategoryBreakdown(ctx, db.GetCategoryBreakdownParams{
		FamilyID:   familyID,
		FromDate:   from,
		ToDate:     to,
		Type:       storedType,
		AccountIds: accountIDs,
	})
	if err != nil {
		return nil, internal(err, "get category breakdown")
	}

	var total int64
	for _, r := range rows {
		total += r.TotalMinor
	}

	slices := make([]*financev1.CategorySlice, 0, len(rows))
	for _, r := range rows {
		// Shares are computed here so every client draws the same pie, and a zero total
		// yields zero shares instead of a division by zero.
		var share float64
		if total != 0 {
			share = float64(r.TotalMinor) / float64(total)
		}
		slices = append(slices, &financev1.CategorySlice{
			CategoryId:       pgconv.UUIDString(r.CategoryID),
			CategoryName:     r.CategoryName,
			Color:            r.Color,
			Total:            money(r.TotalMinor, h.baseCurrency),
			Share:            share,
			TransactionCount: int32(r.TransactionCount),
		})
	}

	return connect.NewResponse(&financev1.GetCategoryBreakdownResponse{
		Slices: slices,
		Total:  money(total, h.baseCurrency),
	}), nil
}

/* ----------------------------------------------------------------- internals */

// transactionInput is the validated, storage-ready form of a create/update request.
type transactionInput struct {
	accountID        string
	counterAccountID string
	categoryID       string
	txType           financev1.TransactionType
	amount           *financev1.Money
	occurredOn       string

	account        pgtype.UUID
	counterAccount pgtype.UUID
	category       pgtype.UUID
	storedType     string
	amountMinor    int64
	currency       string
	date           pgtype.Date
}

// validateTransaction enforces the rules the database CHECK cannot express on its own:
// that referenced accounts and categories belong to the caller's family, and that a transfer's
// two accounts are distinct and share a currency.
func (h *Handler) validateTransaction(
	ctx context.Context, familyID pgtype.UUID, in transactionInput,
) (transactionInput, error) {
	storedType, ok := txTypeToStored(in.txType)
	if !ok {
		return in, invalid("type is required")
	}
	in.storedType = storedType

	if in.amount.GetAmountMinor() <= 0 {
		return in, invalid("amount must be positive; the type carries the direction")
	}
	in.amountMinor = in.amount.GetAmountMinor()

	date, err := pgconv.Date(in.occurredOn)
	if err != nil || !date.Valid {
		return in, invalid("occurred_on must be YYYY-MM-DD")
	}
	in.date = date

	account, err := requiredUUID(in.accountID, "account_id")
	if err != nil {
		return in, err
	}
	in.account = account

	acc, err := h.q.GetAccount(ctx, db.GetAccountParams{ID: account, FamilyID: familyID})
	if err != nil {
		return in, notFoundOr(err, "account")
	}
	in.currency = acc.CurrencyCode
	if code := in.amount.GetCurrencyCode(); code != "" && code != acc.CurrencyCode {
		return in, invalid(fmt.Sprintf(
			"amount is %s but the account is %s", code, acc.CurrencyCode))
	}

	if storedType == typeTransfer {
		counter, err := requiredUUID(in.counterAccountID, "counter_account_id")
		if err != nil {
			return in, invalid("counter_account_id is required for a transfer")
		}
		if in.counterAccountID == in.accountID {
			return in, invalid("a transfer needs two different accounts")
		}
		counterAcc, err := h.q.GetAccount(ctx, db.GetAccountParams{ID: counter, FamilyID: familyID})
		if err != nil {
			return in, notFoundOr(err, "counter account")
		}
		// A cross-currency transfer needs a rate; until this service holds rates, refuse it
		// rather than move the wrong number.
		if counterAcc.CurrencyCode != acc.CurrencyCode {
			return in, invalid("cross-currency transfers are not supported yet")
		}
		in.counterAccount = counter
		in.category = pgtype.UUID{} // enforced by the transactions_shape CHECK too
		return in, nil
	}

	category, err := requiredUUID(in.categoryID, "category_id")
	if err != nil {
		return in, err
	}
	cat, err := h.q.GetCategory(ctx, db.GetCategoryParams{ID: category, FamilyID: familyID})
	if err != nil {
		return in, notFoundOr(err, "category")
	}
	// An expense filed under an income category makes every report wrong.
	if cat.Kind != storedType {
		return in, invalid(fmt.Sprintf("category is %s but the transaction is %s", cat.Kind, storedType))
	}
	in.category = category
	in.counterAccount = pgtype.UUID{}
	return in, nil
}

// familyID is the read-path guard: the caller must be in a family, and that family scopes
// every query.
func (h *Handler) familyID(ctx context.Context) (pgtype.UUID, error) {
	claims, err := fmauth.RequireFamily(ctx)
	if err != nil {
		return pgtype.UUID{}, err
	}
	id, convErr := pgconv.UUID(claims.FamilyID)
	if convErr != nil || !id.Valid {
		return pgtype.UUID{}, connect.NewError(connect.CodeUnauthenticated,
			errors.New("token carries an unusable family_id"))
	}
	return id, nil
}

func (h *Handler) pageSize(requested int32) int32 {
	switch {
	case requested <= 0:
		return h.defaultPageSize
	case requested > h.maxPageSize:
		return h.maxPageSize
	default:
		return requested
	}
}

func dateRange(r *financev1.DateRange) (from, to pgtype.Date, err error) {
	from, ferr := pgconv.Date(r.GetFrom())
	to, terr := pgconv.Date(r.GetTo())
	if ferr != nil || terr != nil || !from.Valid || !to.Valid {
		return from, to, invalid("range.from and range.to must be YYYY-MM-DD")
	}
	if to.Time.Before(from.Time) {
		return from, to, invalid("range.to is before range.from")
	}
	return from, to, nil
}

func uuidList(ids []string, field string) ([]pgtype.UUID, error) {
	// A nil slice would be NULL in Postgres, and `cardinality(NULL) = 0` is NULL, not true —
	// which would silently filter everything out. An empty non-nil slice means "no filter".
	out := make([]pgtype.UUID, 0, len(ids))
	for _, raw := range ids {
		id, err := pgconv.UUID(raw)
		if err != nil || !id.Valid {
			return nil, invalid(fmt.Sprintf("%s contains an invalid id", field))
		}
		out = append(out, id)
	}
	return out, nil
}

func requiredUUID(raw, field string) (pgtype.UUID, error) {
	id, err := pgconv.UUID(raw)
	if err != nil || !id.Valid {
		return pgtype.UUID{}, invalid(field + " is required")
	}
	return id, nil
}

func optionalUUID(raw, field string) (pgtype.UUID, error) {
	if raw == "" {
		return pgtype.UUID{}, nil
	}
	id, err := pgconv.UUID(raw)
	if err != nil {
		return pgtype.UUID{}, invalid(field + " is not a valid id")
	}
	return id, nil
}

// currencyOf resolves the account's currency from the explicit field or the opening balance,
// and refuses anything that is not a three-letter ISO 4217 code — the same rule the column's
// CHECK constraint enforces, reported here as a client error instead of a driver error.
func currencyOf(explicit string, opening *financev1.Money) (string, error) {
	code := strings.ToUpper(trimmed(explicit))
	if code == "" {
		code = strings.ToUpper(trimmed(opening.GetCurrencyCode()))
	}
	if !isCurrencyCode(code) {
		return "", invalid("currency_code must be a three-letter ISO 4217 code")
	}
	if other := strings.ToUpper(trimmed(opening.GetCurrencyCode())); other != "" && other != code {
		return "", invalid("opening_balance currency does not match currency_code")
	}
	return code, nil
}

func isCurrencyCode(s string) bool {
	if len(s) != 3 {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

func invalid(msg string) error {
	return connect.NewError(connect.CodeInvalidArgument, errors.New(msg))
}

func internal(err error, what string) error {
	return connect.NewError(connect.CodeInternal, fmt.Errorf("%s: %w", what, err))
}

// notFoundOr maps "no rows" to NotFound and everything else to Internal. Scoping every query
// by family_id means a row belonging to another family is indistinguishable from a missing
// one — which is the correct answer to give.
func notFoundOr(err error, what string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("%s not found", what))
	}
	return internal(err, "get "+what)
}
