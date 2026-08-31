package handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

// budgetStatus derives a budget's state in the window containing asOf. Nothing here is
// stored: a persisted total drifts the moment a transaction is edited, and the drift is
// invisible until someone reconciles by hand.
func (h *Handler) budgetStatus(
	ctx context.Context, c caller, hh household, b db.Budget, asOf time.Time,
) (*financev1.BudgetStatus, error) {
	window := budgetWindow(b.Period, b.StartOn.Time, asOf)

	spent, err := h.q.SumBudgetSpend(ctx, db.SumBudgetSpendParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		FromDate: pgDate(window.from), ToDate: pgDate(window.to),
		CurrencyCode: b.CurrencyCode,
		GroupID:      b.GroupID, CategoryID: b.CategoryID, MemberID: b.MemberID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "sum budget spend")
	}
	return budgetStatusFrom(b, window, spent, h.today(hh)), nil
}

// budgetStatusFrom is the arithmetic half, kept separate from the query so the ratio and the
// day count are testable without a store.
//
// share is deliberately unclamped: the design draws 256% by clamping the bar and switching
// the colour, which it cannot do if the server has already clamped the number.
func budgetStatusFrom(b db.Budget, window dayRange, spent int64, today time.Time) *financev1.BudgetStatus {
	share := 0.0
	if b.LimitMinor > 0 {
		share = float64(spent) / float64(b.LimitMinor)
	}
	days := int32(0)
	if !today.After(window.to) {
		days = int32(window.to.Sub(startOfDay(today)).Hours()/24) + 1
	}
	return &financev1.BudgetStatus{
		Budget:        toProtoBudget(b),
		Window:        window.proto(),
		Spent:         money(spent, b.CurrencyCode),
		Remaining:     money(b.LimitMinor-spent, b.CurrencyCode),
		Share:         share,
		Exceeded:      spent > b.LimitMinor,
		DaysRemaining: days,
	}
}

func (h *Handler) ListBudgets(
	ctx context.Context, req *connect.Request[financev1.ListBudgetsRequest],
) (*connect.Response[financev1.ListBudgetsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	asOf, err := requireDay("as_of", req.Msg.GetAsOf(), h.today(hh), hh.loc)
	if err != nil {
		return nil, err
	}

	rows, err := h.q.ListBudgets(ctx, db.ListBudgetsParams{
		FamilyID: c.familyID, TargetKind: budgetTargetFilter(req.Msg.GetTarget()),
		IncludeArchived: req.Msg.GetIncludeArchived(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list budgets")
	}

	out := &financev1.ListBudgetsResponse{TotalCount: int32(len(rows))}
	for _, b := range rows {
		status, err := h.budgetStatus(ctx, c, hh, b, asOf)
		if err != nil {
			return nil, err
		}
		if !status.GetExceeded() {
			out.WithinLimitCount++
		}
		out.Budgets = append(out.Budgets, status)
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) CreateBudget(
	ctx context.Context, req *connect.Request[financev1.CreateBudgetRequest],
) (*connect.Response[financev1.CreateBudgetResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg

	limit, err := checkAmount(msg.GetLimit())
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		return nil, invalid("limit must be greater than zero")
	}
	// The budget is spent against by SumBudgetSpend, which filters transactions by the budget's
	// own currency. One in a currency the household does not use would therefore report zero
	// spent forever, never exceed and never notify — so it is refused rather than accepted and
	// left inert. UpdateBudget has no currency field, which would make it uncorrectable too.
	currency := hh.currency()
	if code := trimmed(msg.GetLimit().GetCurrencyCode()); code != "" {
		given, err := checkCurrency(code)
		if err != nil {
			return nil, err
		}
		if given != currency {
			return nil, invalid("a budget is kept in the household currency, %s", currency)
		}
	}

	params := db.CreateBudgetParams{
		FamilyID: c.familyID, LimitMinor: limit, CurrencyCode: currency,
		Period: budgetPeriodFromProto(msg.GetPeriod()), NotifyOnExceed: msg.GetNotifyOnExceed(),
	}

	// Exactly one attach point. The oneof makes "both" unrepresentable; "neither" is still a
	// request a client can send, and it has no meaning.
	switch target := msg.GetTarget().(type) {
	case *financev1.CreateBudgetRequest_GroupId:
		groupID, err := requireUUID("group_id", target.GroupId)
		if err != nil {
			return nil, err
		}
		if _, err := h.q.GetCategoryGroup(ctx, db.GetCategoryGroupParams{
			ID: groupID, FamilyID: c.familyID,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, notFound("category group")
			}
			return nil, h.internal(ctx, err, "get category group")
		}
		params.TargetKind = targetGroup
		params.GroupID = groupID
	case *financev1.CreateBudgetRequest_CategoryId:
		categoryID, err := requireUUID("category_id", target.CategoryId)
		if err != nil {
			return nil, err
		}
		if _, err := h.q.GetCategory(ctx, db.GetCategoryParams{
			ID: categoryID, FamilyID: c.familyID,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, notFound("category")
			}
			return nil, h.internal(ctx, err, "get category")
		}
		params.TargetKind = targetCategory
		params.CategoryID = categoryID
	default:
		return nil, invalid("a budget needs either group_id or category_id")
	}

	startOn, err := requireDay("start_on", msg.GetStartOn(), h.today(hh), hh.loc)
	if err != nil {
		return nil, err
	}
	params.StartOn = pgDate(startOn)
	if params.MemberID, err = optionalUUID("member_id", msg.GetMemberId()); err != nil {
		return nil, err
	}

	row, err := h.q.CreateBudget(ctx, params)
	if err != nil {
		return nil, h.internal(ctx, err, "create budget")
	}
	h.publish(ctx, subjectBudgetCreated, &financev1.BudgetCreatedEvent{
		FamilyId:   c.family,
		BudgetId:   pgconv.UUIDString(row.ID),
		TargetKind: budgetTargetToProto(row.TargetKind),
		GroupId:    pgconv.UUIDString(row.GroupID),
		CategoryId: pgconv.UUIDString(row.CategoryID),
		Limit:      money(row.LimitMinor, row.CurrencyCode),
		Period:     budgetPeriodToProto(row.Period),
		OccurredAt: h.timestamp(),
	})

	status, err := h.budgetStatus(ctx, c, hh, row, h.today(hh))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&financev1.CreateBudgetResponse{Budget: status}), nil
}

func (h *Handler) UpdateBudget(
	ctx context.Context, req *connect.Request[financev1.UpdateBudgetRequest],
) (*connect.Response[financev1.UpdateBudgetResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	id, err := requireUUID("budget_id", msg.GetBudgetId())
	if err != nil {
		return nil, err
	}

	before, err := h.q.GetBudget(ctx, db.GetBudgetParams{ID: id, FamilyID: c.familyID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("budget")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "get budget")
	}
	wasExceeded, err := h.budgetStatus(ctx, c, hh, before, h.today(hh))
	if err != nil {
		return nil, err
	}

	params := db.UpdateBudgetParams{ID: id, FamilyID: c.familyID}
	if msg.Limit != nil {
		limit, err := checkAmount(msg.GetLimit())
		if err != nil {
			return nil, err
		}
		if limit <= 0 {
			return nil, invalid("limit must be greater than zero")
		}
		params.LimitMinor = &limit
	}
	if msg.Period != nil {
		period := budgetPeriodFromProto(msg.GetPeriod())
		params.Period = &period
	}
	if msg.StartOn != nil {
		day, ok := parseDay(trimmed(msg.GetStartOn()), hh.loc)
		if !ok {
			return nil, invalid("start_on must be a date as YYYY-MM-DD")
		}
		params.StartOn = pgDate(day)
	}
	if msg.NotifyOnExceed != nil {
		notify := msg.GetNotifyOnExceed()
		params.NotifyOnExceed = &notify
	}
	if msg.Archived != nil {
		archived := msg.GetArchived()
		params.Archived = &archived
	}

	row, err := h.q.UpdateBudget(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("budget")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "update budget")
	}
	h.publish(ctx, subjectBudgetUpdated, &financev1.BudgetUpdatedEvent{
		FamilyId:   c.family,
		BudgetId:   pgconv.UUIDString(row.ID),
		Limit:      money(row.LimitMinor, row.CurrencyCode),
		Period:     budgetPeriodToProto(row.Period),
		Archived:   row.Archived,
		OccurredAt: h.timestamp(),
	})

	status, err := h.budgetStatus(ctx, c, hh, row, h.today(hh))
	if err != nil {
		return nil, err
	}
	// Raising a limit can pull a budget back under it. Without the recovery event a sent
	// overspend alert could never be retracted, and the same window would alert again on the
	// next breach.
	if wasExceeded.GetExceeded() && !status.GetExceeded() {
		h.publish(ctx, subjectBudgetRecovered, &financev1.BudgetRecoveredEvent{
			FamilyId: c.family, BudgetId: pgconv.UUIDString(row.ID),
			Window: status.GetWindow(), OccurredAt: h.timestamp(),
		})
	}
	return connect.NewResponse(&financev1.UpdateBudgetResponse{Budget: status}), nil
}

func (h *Handler) DeleteBudget(
	ctx context.Context, req *connect.Request[financev1.DeleteBudgetRequest],
) (*connect.Response[financev1.DeleteBudgetResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requireUUID("budget_id", req.Msg.GetBudgetId())
	if err != nil {
		return nil, err
	}
	rows, err := h.q.DeleteBudget(ctx, db.DeleteBudgetParams{ID: id, FamilyID: c.familyID})
	if err != nil {
		return nil, h.internal(ctx, err, "delete budget")
	}
	if rows == 0 {
		return nil, notFound("budget")
	}
	return connect.NewResponse(&financev1.DeleteBudgetResponse{}), nil
}

// affectedBudgets is what every transaction write returns: the budgets whose window contains
// the transaction, so the app repaints its bars and can raise an overspend toast without a
// refetch. Both the category's own budget and its group's are included, because spend in a
// category counts toward both.
//
// A transaction with no category (a transfer) moves no budget, and asks nothing.
func (h *Handler) affectedBudgets(
	ctx context.Context, c caller, hh household, categoryID pgtype.UUID, occurredOn time.Time,
) ([]*financev1.BudgetStatus, error) {
	if !categoryID.Valid {
		return nil, nil
	}
	rows, err := h.q.ListBudgetsForCategory(ctx, db.ListBudgetsForCategoryParams{
		FamilyID: c.familyID, CategoryID: categoryID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list budgets for category")
	}
	out := make([]*financev1.BudgetStatus, 0, len(rows))
	for _, b := range rows {
		// The window the transaction landed in, not today's: editing last month's grocery bill
		// has to report last month's bar.
		status, err := h.budgetStatus(ctx, c, hh, b, occurredOn)
		if err != nil {
			return nil, err
		}
		out = append(out, status)
	}
	return out, nil
}

// announceBudgetChanges compares a write's before and after state and publishes the
// exceeded/recovered edges. Only edges: republishing "exceeded" on every subsequent
// transaction in an already-blown budget would make the notification service the one deciding
// what is new, which is exactly the state that produces duplicate pushes.
func (h *Handler) announceBudgetChanges(
	ctx context.Context, c caller, before, after []*financev1.BudgetStatus, triggeringTxID string,
) {
	was := make(map[string]bool, len(before))
	for _, s := range before {
		was[s.GetBudget().GetId()] = s.GetExceeded()
	}
	for _, s := range after {
		id := s.GetBudget().GetId()
		switch {
		case s.GetExceeded() && !was[id]:
			if !s.GetBudget().GetNotifyOnExceed() {
				continue
			}
			h.publish(ctx, subjectBudgetExceeded, &financev1.BudgetExceededEvent{
				FamilyId:                c.family,
				BudgetId:                id,
				TargetKind:              s.GetBudget().GetTargetKind(),
				GroupId:                 s.GetBudget().GetGroupId(),
				CategoryId:              s.GetBudget().GetCategoryId(),
				Limit:                   s.GetBudget().GetLimit(),
				Spent:                   s.GetSpent(),
				Share:                   s.GetShare(),
				Window:                  s.GetWindow(),
				TriggeringTransactionId: triggeringTxID,
				MemberId:                s.GetBudget().GetMemberId(),
				OccurredAt:              h.timestamp(),
			})
		case !s.GetExceeded() && was[id]:
			h.publish(ctx, subjectBudgetRecovered, &financev1.BudgetRecoveredEvent{
				FamilyId: c.family, BudgetId: id,
				Window: s.GetWindow(), OccurredAt: h.timestamp(),
			})
		}
	}
}
