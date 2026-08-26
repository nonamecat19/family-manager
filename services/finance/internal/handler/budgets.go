package handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

// Stored period values; the CHECK constraint in 000002_budgets.up.sql is the other half.
const (
	periodWeek  = "week"
	periodMonth = "month"
	periodYear  = "year"
)

func (h *Handler) CreateBudget(
	ctx context.Context, req *connect.Request[financev1.CreateBudgetRequest],
) (*connect.Response[financev1.CreateBudgetResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}

	name, err := requiredName(req.Msg.GetName())
	if err != nil {
		return nil, err
	}
	period, ok := periodToStored(req.Msg.GetPeriod())
	if !ok {
		return nil, invalid("period must be week, month or year")
	}
	if req.Msg.GetLimit().GetAmountMinor() <= 0 {
		return nil, invalid("limit must be above zero")
	}

	startOn, err := pgconv.Date(req.Msg.GetStartOn())
	if err != nil || !startOn.Valid {
		return nil, invalid("start_on must be YYYY-MM-DD")
	}

	categoryID, err := h.budgetCategory(ctx, familyID, req.Msg.GetCategoryId(), req.Msg.GetPeriod())
	if err != nil {
		return nil, err
	}

	currency := req.Msg.GetLimit().GetCurrencyCode()
	if currency == "" {
		currency = h.baseCurrency
	}
	if !isCurrencyCode(currency) {
		return nil, invalid("limit currency must be a three-letter ISO 4217 code")
	}

	budget, err := h.q.CreateBudget(ctx, db.CreateBudgetParams{
		FamilyID:     familyID,
		Name:         name,
		CategoryID:   categoryID,
		LimitMinor:   req.Msg.GetLimit().GetAmountMinor(),
		CurrencyCode: currency,
		Period:       period,
		StartOn:      startOn,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, connect.NewError(connect.CodeAlreadyExists,
				errors.New("a budget already covers that category for that period"))
		}
		return nil, h.internal(ctx, err, "create budget")
	}

	return connect.NewResponse(&financev1.CreateBudgetResponse{
		Budget: toProtoBudget(budget),
	}), nil
}

func (h *Handler) ListBudgets(
	ctx context.Context, req *connect.Request[financev1.ListBudgetsRequest],
) (*connect.Response[financev1.ListBudgetsResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}
	asOf, err := h.asOf(req.Msg.GetAsOf())
	if err != nil {
		return nil, err
	}

	rows, err := h.q.ListBudgets(ctx, db.ListBudgetsParams{
		FamilyID:        familyID,
		IncludeArchived: req.Msg.GetIncludeArchived(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list budgets")
	}

	out := make([]*financev1.BudgetStatus, 0, len(rows))
	for _, budget := range rows {
		status, err := h.statusOf(ctx, familyID, budget, asOf)
		if err != nil {
			return nil, err
		}
		out = append(out, status)
	}

	return connect.NewResponse(&financev1.ListBudgetsResponse{Budgets: out}), nil
}

func (h *Handler) GetBudget(
	ctx context.Context, req *connect.Request[financev1.GetBudgetRequest],
) (*connect.Response[financev1.GetBudgetResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requiredUUID(req.Msg.GetId(), "id")
	if err != nil {
		return nil, err
	}
	asOf, err := h.asOf(req.Msg.GetAsOf())
	if err != nil {
		return nil, err
	}

	budget, err := h.q.GetBudget(ctx, db.GetBudgetParams{ID: id, FamilyID: familyID})
	if err != nil {
		return nil, h.notFoundOr(ctx, err, "budget")
	}

	status, err := h.statusOf(ctx, familyID, budget, asOf)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&financev1.GetBudgetResponse{Budget: status}), nil
}

func (h *Handler) UpdateBudget(
	ctx context.Context, req *connect.Request[financev1.UpdateBudgetRequest],
) (*connect.Response[financev1.UpdateBudgetResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requiredUUID(req.Msg.GetId(), "id")
	if err != nil {
		return nil, err
	}
	name, err := requiredName(req.Msg.GetName())
	if err != nil {
		return nil, err
	}
	period, ok := periodToStored(req.Msg.GetPeriod())
	if !ok {
		return nil, invalid("period must be week, month or year")
	}
	if req.Msg.GetLimit().GetAmountMinor() <= 0 {
		return nil, invalid("limit must be above zero")
	}
	startOn, err := pgconv.Date(req.Msg.GetStartOn())
	if err != nil || !startOn.Valid {
		return nil, invalid("start_on must be YYYY-MM-DD")
	}
	categoryID, err := h.budgetCategory(ctx, familyID, req.Msg.GetCategoryId(), req.Msg.GetPeriod())
	if err != nil {
		return nil, err
	}

	budget, err := h.q.UpdateBudget(ctx, db.UpdateBudgetParams{
		ID:         id,
		FamilyID:   familyID,
		Name:       name,
		CategoryID: categoryID,
		LimitMinor: req.Msg.GetLimit().GetAmountMinor(),
		Period:     period,
		StartOn:    startOn,
		Archived:   req.Msg.GetArchived(),
		SortOrder:  req.Msg.GetSortOrder(),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, connect.NewError(connect.CodeAlreadyExists,
				errors.New("a budget already covers that category for that period"))
		}
		return nil, h.notFoundOr(ctx, err, "budget")
	}

	return connect.NewResponse(&financev1.UpdateBudgetResponse{
		Budget: toProtoBudget(budget),
	}), nil
}

func (h *Handler) DeleteBudget(
	ctx context.Context, req *connect.Request[financev1.DeleteBudgetRequest],
) (*connect.Response[financev1.DeleteBudgetResponse], error) {
	familyID, err := h.familyID(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requiredUUID(req.Msg.GetId(), "id")
	if err != nil {
		return nil, err
	}

	// Unlike accounts and categories, a budget owns no history — deleting one rewrites no
	// past report, so there is nothing to protect here.
	rows, err := h.q.DeleteBudget(ctx, db.DeleteBudgetParams{ID: id, FamilyID: familyID})
	if err != nil {
		return nil, h.internal(ctx, err, "delete budget")
	}
	if rows == 0 {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("budget not found"))
	}
	return connect.NewResponse(&financev1.DeleteBudgetResponse{}), nil
}

/* ----------------------------------------------------------------- internals */

// statusOf derives a budget's progress for the window containing asOf.
func (h *Handler) statusOf(
	ctx context.Context, familyID pgtype.UUID, budget db.Budget, asOf time.Time,
) (*financev1.BudgetStatus, error) {
	window, err := windowFor(budget.Period, budget.StartOn.Time, asOf)
	if err != nil {
		// A period outside the CHECK constraint means the row was written by something other
		// than this service; report it rather than guessing a window.
		return nil, h.internal(ctx, err, "budget window")
	}

	spent, err := h.q.SumBudgetSpend(ctx, db.SumBudgetSpendParams{
		FamilyID:   familyID,
		FromDate:   pgtype.Date{Time: window.from, Valid: true},
		ToDate:     pgtype.Date{Time: window.to, Valid: true},
		CategoryID: budget.CategoryID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "sum budget spend")
	}

	// share is computed here so every client draws the same bar; limit_minor is CHECKed above
	// zero, so there is no division by zero to guard.
	share := float64(spent) / float64(budget.LimitMinor)

	return &financev1.BudgetStatus{
		Budget: toProtoBudget(budget),
		Period: &financev1.DateRange{
			From: window.from.Format(time.DateOnly),
			To:   window.to.Format(time.DateOnly),
		},
		Spent:         money(spent, budget.CurrencyCode),
		Remaining:     money(budget.LimitMinor-spent, budget.CurrencyCode),
		Share:         share,
		Exceeded:      spent > budget.LimitMinor,
		DaysRemaining: window.daysRemaining(asOf),
	}, nil
}

// budgetCategory validates the category a budget is scoped to. An empty id means the whole
// household, which is a legitimate budget and not a missing field.
func (h *Handler) budgetCategory(
	ctx context.Context, familyID pgtype.UUID, raw string, period financev1.BudgetPeriod,
) (pgtype.UUID, error) {
	_ = period
	if raw == "" {
		return pgtype.UUID{}, nil
	}

	categoryID, err := requiredUUID(raw, "category_id")
	if err != nil {
		return pgtype.UUID{}, err
	}

	category, err := h.q.GetCategory(ctx, db.GetCategoryParams{ID: categoryID, FamilyID: familyID})
	if err != nil {
		return pgtype.UUID{}, h.notFoundOr(ctx, err, "category")
	}
	// Budgeting income makes no sense: a limit exists to cap what leaves.
	if category.Kind != typeExpense {
		return pgtype.UUID{}, invalid("only expense categories can be budgeted")
	}
	return categoryID, nil
}

// asOf resolves the requested day, defaulting to today.
func (h *Handler) asOf(raw string) (time.Time, error) {
	if raw == "" {
		return h.now(), nil
	}
	parsed, err := time.Parse(time.DateOnly, raw)
	if err != nil {
		return time.Time{}, invalid("as_of must be YYYY-MM-DD")
	}
	return parsed, nil
}

func periodToStored(p financev1.BudgetPeriod) (string, bool) {
	switch p {
	case financev1.BudgetPeriod_BUDGET_PERIOD_WEEK:
		return periodWeek, true
	case financev1.BudgetPeriod_BUDGET_PERIOD_MONTH:
		return periodMonth, true
	case financev1.BudgetPeriod_BUDGET_PERIOD_YEAR:
		return periodYear, true
	default:
		return "", false
	}
}

func periodToProto(s string) financev1.BudgetPeriod {
	switch s {
	case periodWeek:
		return financev1.BudgetPeriod_BUDGET_PERIOD_WEEK
	case periodMonth:
		return financev1.BudgetPeriod_BUDGET_PERIOD_MONTH
	case periodYear:
		return financev1.BudgetPeriod_BUDGET_PERIOD_YEAR
	default:
		return financev1.BudgetPeriod_BUDGET_PERIOD_UNSPECIFIED
	}
}

func toProtoBudget(b db.Budget) *financev1.Budget {
	return &financev1.Budget{
		Id:         pgconv.UUIDString(b.ID),
		FamilyId:   pgconv.UUIDString(b.FamilyID),
		Name:       b.Name,
		CategoryId: pgconv.UUIDString(b.CategoryID),
		Limit:      money(b.LimitMinor, b.CurrencyCode),
		Period:     periodToProto(b.Period),
		StartOn:    pgconv.DateString(b.StartOn),
		Archived:   b.Archived,
		SortOrder:  b.SortOrder,
		CreatedAt:  pgconv.Timestamp(b.CreatedAt),
		UpdatedAt:  pgconv.Timestamp(b.UpdatedAt),
	}
}

// announceBudgetCrossings publishes finance.budget.exceeded for every budget this expense
// pushed past its limit.
//
// "Pushed past" is the crossing, not the state: the spend before this transaction was within
// the limit and after it is not. Publishing on state instead would fire on every subsequent
// purchase in an overspent month, which is a notification storm rather than a signal.
//
// Best-effort by design — the transaction is already committed, and a failure to notify must
// not fail the write. Errors are logged inside publish.
func (h *Handler) announceBudgetCrossings(
	ctx context.Context, familyID pgtype.UUID, tx db.Transaction, userID string,
) {
	if tx.Type != typeExpense {
		return // only spending consumes a budget
	}

	budgets, err := h.q.ListBudgetsForCategory(ctx, db.ListBudgetsForCategoryParams{
		FamilyID:   familyID,
		CategoryID: tx.CategoryID,
	})
	if err != nil {
		h.log.WarnContext(ctx, "budget check skipped", slogError(err))
		return
	}

	for _, budget := range budgets {
		window, err := windowFor(budget.Period, budget.StartOn.Time, tx.OccurredOn.Time)
		if err != nil {
			continue
		}
		// The transaction must fall inside the window it would consume — backdating an
		// expense into a closed month must not alarm about this one.
		if tx.OccurredOn.Time.Before(window.from) || tx.OccurredOn.Time.After(window.to) {
			continue
		}

		spent, err := h.q.SumBudgetSpend(ctx, db.SumBudgetSpendParams{
			FamilyID:   familyID,
			FromDate:   pgtype.Date{Time: window.from, Valid: true},
			ToDate:     pgtype.Date{Time: window.to, Valid: true},
			CategoryID: budget.CategoryID,
		})
		if err != nil {
			h.log.WarnContext(ctx, "budget spend query failed", slogError(err))
			continue
		}

		crossed := spent > budget.LimitMinor && spent-tx.AmountMinor <= budget.LimitMinor
		if !crossed {
			continue
		}

		h.publish(ctx, events.SubjectFinanceBudgetExceeded, &financev1.BudgetExceededEvent{
			FamilyId:   pgconv.UUIDString(familyID),
			BudgetId:   pgconv.UUIDString(budget.ID),
			BudgetName: budget.Name,
			CategoryId: pgconv.UUIDString(budget.CategoryID),
			Limit:      money(budget.LimitMinor, budget.CurrencyCode),
			Spent:      money(spent, budget.CurrencyCode),
			Period: &financev1.DateRange{
				From: window.from.Format(time.DateOnly),
				To:   window.to.Format(time.DateOnly),
			},
			TriggeredByUserId: userID,
			OccurredAt:        h.timestamp(),
		})
	}
}

// isUniqueViolation reports whether err is Postgres SQLSTATE 23505.
func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23505"
}
