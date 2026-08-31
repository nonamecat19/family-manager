package handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

// Widgets are per-user placements, not household state: two members place the same type and
// each sees their own scope, so every read here is keyed by user_id as well as family_id.

func (h *Handler) ListWidgets(
	ctx context.Context, _ *connect.Request[financev1.ListWidgetsRequest],
) (*connect.Response[financev1.ListWidgetsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.q.ListWidgets(ctx, db.ListWidgetsParams{
		FamilyID: c.familyID, UserID: c.userID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list widgets")
	}
	out := make([]*financev1.WidgetInstance, 0, len(rows))
	for _, w := range rows {
		out = append(out, toProtoWidget(w))
	}
	return connect.NewResponse(&financev1.ListWidgetsResponse{Widgets: out}), nil
}

func (h *Handler) AddWidget(
	ctx context.Context, req *connect.Request[financev1.AddWidgetRequest],
) (*connect.Response[financev1.AddWidgetResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	widgetType := widgetTypeFromProto(msg.GetType())
	if widgetType == "" {
		return nil, invalid("type must name a widget type")
	}
	if err := checkBatch("target_account_ids", msg.GetTargetAccountIds()); err != nil {
		return nil, err
	}
	accountIDs, err := uuidList("target_account_ids", msg.GetTargetAccountIds())
	if err != nil {
		return nil, err
	}
	targetRef, err := optionalUUID("target_ref", msg.GetTargetRef())
	if err != nil {
		return nil, err
	}
	scopeMember, err := optionalUUID("scope.member_id", msg.GetScope().GetMemberId())
	if err != nil {
		return nil, err
	}
	scopeAccount, err := optionalUUID("scope.account_id", msg.GetScope().GetAccountId())
	if err != nil {
		return nil, err
	}

	row, err := h.q.CreateWidget(ctx, db.CreateWidgetParams{
		FamilyID: c.familyID, UserID: c.userID, Type: widgetType,
		Size:          widgetSizeFromProto(msg.GetSize()),
		ScopeKind:     scopeKindFromProto(msg.GetScope().GetKind()),
		ScopeMemberID: scopeMember, ScopeAccountID: scopeAccount,
		TargetRef: targetRef, TargetAccountIds: accountIDs,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "add widget")
	}
	return connect.NewResponse(&financev1.AddWidgetResponse{Widget: toProtoWidget(row)}), nil
}

func (h *Handler) UpdateWidget(
	ctx context.Context, req *connect.Request[financev1.UpdateWidgetRequest],
) (*connect.Response[financev1.UpdateWidgetResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	id, err := requireUUID("widget_id", msg.GetWidgetId())
	if err != nil {
		return nil, err
	}

	params := db.UpdateWidgetParams{ID: id, FamilyID: c.familyID, UserID: c.userID}
	if msg.Size != nil {
		size := widgetSizeFromProto(msg.GetSize())
		params.Size = &size
	}
	if scope := msg.GetScope(); scope != nil {
		kind := scopeKindFromProto(scope.GetKind())
		params.ScopeKind = &kind
		if params.ScopeMemberID, err = optionalUUID("scope.member_id", scope.GetMemberId()); err != nil {
			return nil, err
		}
		if params.ScopeAccountID, err = optionalUUID("scope.account_id", scope.GetAccountId()); err != nil {
			return nil, err
		}
	}
	if msg.TargetRef != nil {
		if params.TargetRef, err = optionalUUID("target_ref", msg.GetTargetRef()); err != nil {
			return nil, err
		}
	}
	if len(msg.GetTargetAccountIds()) > 0 {
		if err := checkBatch("target_account_ids", msg.GetTargetAccountIds()); err != nil {
			return nil, err
		}
		if params.TargetAccountIds, err = uuidList("target_account_ids", msg.GetTargetAccountIds()); err != nil {
			return nil, err
		}
	}

	row, err := h.q.UpdateWidget(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("widget")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "update widget")
	}
	return connect.NewResponse(&financev1.UpdateWidgetResponse{Widget: toProtoWidget(row)}), nil
}

func (h *Handler) RemoveWidget(
	ctx context.Context, req *connect.Request[financev1.RemoveWidgetRequest],
) (*connect.Response[financev1.RemoveWidgetResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requireUUID("widget_id", req.Msg.GetWidgetId())
	if err != nil {
		return nil, err
	}
	rows, err := h.q.DeleteWidget(ctx, db.DeleteWidgetParams{
		ID: id, FamilyID: c.familyID, UserID: c.userID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "remove widget")
	}
	if rows == 0 {
		return nil, notFound("widget")
	}
	return connect.NewResponse(&financev1.RemoveWidgetResponse{}), nil
}

// GetWidgetData refreshes every placed widget in one request. One call per widget would make
// the Android update pass N round trips on a process that may have just been cold-started; an
// empty id list means "everything I placed", because a cold widget host may not know its own
// ids yet.
func (h *Handler) GetWidgetData(
	ctx context.Context, req *connect.Request[financev1.GetWidgetDataRequest],
) (*connect.Response[financev1.GetWidgetDataResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	if err := checkBatch("widget_ids", req.Msg.GetWidgetIds()); err != nil {
		return nil, err
	}

	var widgets []db.WidgetInstance
	if ids := req.Msg.GetWidgetIds(); len(ids) > 0 {
		parsed, err := uuidList("widget_ids", ids)
		if err != nil {
			return nil, err
		}
		widgets, err = h.q.ListWidgetsByIDs(ctx, db.ListWidgetsByIDsParams{
			FamilyID: c.familyID, UserID: c.userID, WidgetIds: parsed,
		})
		if err != nil {
			return nil, h.internal(ctx, err, "list widgets")
		}
	} else {
		widgets, err = h.q.ListWidgets(ctx, db.ListWidgetsParams{
			FamilyID: c.familyID, UserID: c.userID,
		})
		if err != nil {
			return nil, h.internal(ctx, err, "list widgets")
		}
	}

	// Widgets show the current month; a widget with its own period selector is not a widget.
	month, err := resolvePeriod(nil, h.now(), hh.loc, hh.settings.WeekStartsOn)
	if err != nil {
		return nil, h.internal(ctx, err, "resolve widget period")
	}

	out := &financev1.GetWidgetDataResponse{}
	for _, w := range widgets {
		payload, err := h.widgetPayload(ctx, c, hh, w, month)
		if err != nil {
			return nil, err
		}
		out.Payloads = append(out.Payloads, payload)
	}
	return connect.NewResponse(out), nil
}

func (h *Handler) widgetPayload(
	ctx context.Context, c caller, hh household, w db.WidgetInstance, month dayRange,
) (*financev1.WidgetPayload, error) {
	payload := &financev1.WidgetPayload{
		WidgetId:    pgconv.UUIDString(w.ID),
		Type:        widgetTypeToProto(w.Type),
		RefreshedAt: h.timestamp(),
	}
	// A widget scoped to one member filters like the member chip does; a family-scoped widget
	// filters by nothing, which is the same request with an empty list.
	members := []pgtype.UUID{}
	if w.ScopeKind == "member" && w.ScopeMemberID.Valid {
		members = append(members, w.ScopeMemberID)
	}
	accounts := []pgtype.UUID{}
	if w.ScopeKind == "account" && w.ScopeAccountID.Valid {
		accounts = append(accounts, w.ScopeAccountID)
	}
	expense := kindExpense

	switch w.Type {
	case "quick_add":
		templates, err := h.q.ListTemplates(ctx, db.ListTemplatesParams{
			FamilyID: c.familyID, OwnerUserID: c.userID,
		})
		if err != nil {
			return nil, h.internal(ctx, err, "list templates")
		}
		data := &financev1.QuickAddWidgetData{}
		for i, t := range templates {
			// The 4×2 cell draws four slots; sending more is bytes the widget throws away.
			if i == quickAddSlots {
				break
			}
			data.Templates = append(data.Templates, toProtoTemplate(t))
		}
		payload.Data = &financev1.WidgetPayload_QuickAdd{QuickAdd: data}

	case "month":
		total, err := h.q.SumVisibleTransactions(ctx, db.SumVisibleTransactionsParams{
			FamilyID: c.familyID, ViewerMemberID: c.memberID(),
			FromDate: pgDate(month.from), ToDate: pgDate(month.to),
			CurrencyCode: hh.currency(), Kind: &expense,
			MemberIds: members, AccountIds: accounts, CategoryIds: []pgtype.UUID{},
		})
		if err != nil {
			return nil, h.internal(ctx, err, "sum transactions")
		}
		statuses, within, err := h.groupBudgetStatuses(ctx, c, hh)
		if err != nil {
			return nil, err
		}
		payload.Data = &financev1.WidgetPayload_Month{Month: &financev1.MonthWidgetData{
			Label:              bucketLabel(financev1.PeriodGranularity_PERIOD_GRANULARITY_MONTH, month),
			PeriodTotal:        money(total.TotalMinor, hh.currency()),
			BudgetCount:        int32(len(statuses)),
			BudgetsWithinLimit: within,
		}}

	case "category":
		data := &financev1.CategoryWidgetData{}
		if w.TargetRef.Valid {
			cat, err := h.q.GetCategory(ctx, db.GetCategoryParams{
				ID: w.TargetRef, FamilyID: c.familyID,
			})
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, h.internal(ctx, err, "get category")
			}
			if err == nil {
				sums, err := h.q.SumByCategory(ctx, db.SumByCategoryParams{
					FamilyID: c.familyID, ViewerMemberID: c.memberID(),
					FromDate: pgDate(month.from), ToDate: pgDate(month.to),
					CurrencyCode: hh.currency(), Kind: &expense,
					GroupID: cat.GroupID, MemberIds: members,
				})
				if err != nil {
					return nil, h.internal(ctx, err, "sum by category")
				}
				var amount int64
				for _, r := range sums {
					if pgconv.UUIDString(r.CategoryID) == pgconv.UUIDString(cat.ID) {
						amount = r.TotalMinor
					}
				}
				data.CategoryId = pgconv.UUIDString(cat.ID)
				data.Name = cat.Name
				data.Icon = cat.Icon
				data.Amount = money(amount, hh.currency())

				budgets, err := h.q.ListBudgetsForCategory(ctx, db.ListBudgetsForCategoryParams{
					FamilyID: c.familyID, CategoryID: cat.ID,
				})
				if err != nil {
					return nil, h.internal(ctx, err, "list budgets for category")
				}
				for _, b := range budgets {
					if b.TargetKind != targetCategory {
						continue
					}
					status, err := h.budgetStatus(ctx, c, hh, b, h.today(hh))
					if err != nil {
						return nil, err
					}
					data.Budget = status
					break
				}
			}
		}
		payload.Data = &financev1.WidgetPayload_Category{Category: data}

	case "budgets_and_family":
		statuses, _, err := h.groupBudgetStatuses(ctx, c, hh)
		if err != nil {
			return nil, err
		}
		spending, err := h.memberSpending(ctx, c, hh, month, &expense, accounts)
		if err != nil {
			return nil, err
		}
		total, err := h.q.SumVisibleTransactions(ctx, db.SumVisibleTransactionsParams{
			FamilyID: c.familyID, ViewerMemberID: c.memberID(),
			FromDate: pgDate(month.from), ToDate: pgDate(month.to),
			CurrencyCode: hh.currency(), Kind: &expense,
			MemberIds: members, AccountIds: accounts, CategoryIds: []pgtype.UUID{},
		})
		if err != nil {
			return nil, h.internal(ctx, err, "sum transactions")
		}
		payload.Data = &financev1.WidgetPayload_BudgetsAndFamily{
			BudgetsAndFamily: &financev1.BudgetsAndFamilyWidgetData{
				Budgets: statuses, Members: spending,
				PeriodTotal: money(total.TotalMinor, hh.currency()),
			},
		}

	case "recent_transactions":
		rows, err := h.q.ListVisibleTransactions(ctx, db.ListVisibleTransactionsParams{
			FamilyID: c.familyID, ViewerMemberID: c.memberID(),
			FromDate: pgDate(month.from.AddDate(0, -1, 0)), ToDate: pgDate(month.to),
			MemberIds: members, AccountIds: accounts,
			CategoryIds: []pgtype.UUID{}, GroupIds: []pgtype.UUID{},
			IncludeTransfers: true, PageSize: recentTransactionRows,
		})
		if err != nil {
			return nil, h.internal(ctx, err, "list transactions")
		}
		data := &financev1.RecentTransactionsWidgetData{}
		for _, r := range rows {
			data.Transactions = append(data.Transactions, toProtoTransaction(viewFromTxRow(r)))
		}
		payload.Data = &financev1.WidgetPayload_RecentTransactions{RecentTransactions: data}

	case "accounts":
		// The same visibility predicate as the accounts screen: a widget on a locked home
		// screen is not a reason to skip it.
		rows, err := h.q.ListVisibleAccounts(ctx, db.ListVisibleAccountsParams{
			FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		})
		if err != nil {
			return nil, h.internal(ctx, err, "list accounts")
		}
		wanted := map[string]bool{}
		for _, id := range w.TargetAccountIds {
			wanted[pgconv.UUIDString(id)] = true
		}
		data := &financev1.AccountsWidgetData{}
		for _, r := range rows {
			if len(wanted) > 0 && !wanted[pgconv.UUIDString(r.ID)] {
				continue
			}
			data.Accounts = append(data.Accounts, toProtoAccount(viewFromList(r)))
		}
		balances, err := h.q.SumFamilyBalances(ctx, db.SumFamilyBalancesParams{
			FamilyID: c.familyID, CurrencyCode: hh.currency(),
		})
		if err != nil {
			return nil, h.internal(ctx, err, "sum family balances")
		}
		data.SharedBalance = money(balances.SharedBalanceMinor, hh.currency())
		payload.Data = &financev1.WidgetPayload_Accounts{Accounts: data}
	}
	return payload, nil
}

const (
	quickAddSlots         = 4
	recentTransactionRows = 5
)

// groupBudgetStatuses is the "4 з 5" pair the month and budgets widgets both draw: every group
// budget's status, and how many of them are currently inside their limit.
func (h *Handler) groupBudgetStatuses(
	ctx context.Context, c caller, hh household,
) ([]*financev1.BudgetStatus, int32, error) {
	rows, err := h.q.ListBudgets(ctx, db.ListBudgetsParams{
		FamilyID: c.familyID, TargetKind: strPtr(targetGroup),
	})
	if err != nil {
		return nil, 0, h.internal(ctx, err, "list budgets")
	}
	out := make([]*financev1.BudgetStatus, 0, len(rows))
	var within int32
	for _, b := range rows {
		status, err := h.budgetStatus(ctx, c, hh, b, h.today(hh))
		if err != nil {
			return nil, 0, err
		}
		if !status.GetExceeded() {
			within++
		}
		out = append(out, status)
	}
	return out, within, nil
}
