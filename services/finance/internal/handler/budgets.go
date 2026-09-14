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
		status, err := h.budgetStatus(ctx, c, hh, b, occurredOn)
		if err != nil {
			return nil, err
		}
		out = append(out, status)
	}
	return out, nil
}

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
