package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

const (
	subjectSubscriptionPosted  events.Subject = "finance.subscription.posted"
	subjectSubscriptionSkipped events.Subject = "finance.subscription.skipped"

	subscriptionActive    = "active"
	subscriptionCancelled = "cancelled"

	maxSubscriptionMonths    = 120
	subscriptionDueBatchSize = 200
)

func subscriptionStatusToProto(s string) financev1.SubscriptionStatus {
	switch s {
	case subscriptionActive:
		return financev1.SubscriptionStatus_SUBSCRIPTION_STATUS_ACTIVE
	case subscriptionCancelled:
		return financev1.SubscriptionStatus_SUBSCRIPTION_STATUS_CANCELLED
	}
	return financev1.SubscriptionStatus_SUBSCRIPTION_STATUS_UNSPECIFIED
}

func toProtoSubscription(s db.Subscription, groupID pgtype.UUID, paid, payments int64) *financev1.Subscription {
	return &financev1.Subscription{
		Id:         pgconv.UUIDString(s.ID),
		FamilyId:   pgconv.UUIDString(s.FamilyID),
		Name:       s.Name,
		Amount:     money(s.AmountMinor, s.CurrencyCode),
		Type:       txTypeToProto(s.Type),
		CategoryId: pgconv.UUIDString(s.CategoryID),
		AccountId:  pgconv.UUIDString(s.AccountID),
		MemberId:   pgconv.UUIDString(s.MemberID),
		Cadence: &financev1.Cadence{
			Interval:   s.IntervalCount,
			Unit:       recurrenceUnitToProto(s.IntervalUnit),
			DayOfMonth: s.DayOfMonth,
			DayOfWeek:  weekdayToProto(s.DayOfWeek),
		},
		NextDueOn: pgconv.DateString(s.NextDueOn),
		EndOn:     pgconv.DateString(s.EndOn),
		AutoPost:  s.AutoPost,
		Active:    s.Active,
		Status:    subscriptionStatusToProto(s.Status),
		CreatedAt: pgconv.Timestamp(s.CreatedAt),
		UpdatedAt: pgconv.Timestamp(s.UpdatedAt),
	}
}

func monthlyApproximate(amountMinor int64, intervalCount int32, intervalUnit string) int64 {
	periodsPerYear := float64(0)
	switch intervalUnit {
	case "day":
		periodsPerYear = 365.25 / float64(intervalCount)
	case "week":
		periodsPerYear = 52.14 / float64(intervalCount)
	case "month":
		periodsPerYear = 12.0 / float64(intervalCount)
	case "year":
		periodsPerYear = 1.0 / float64(intervalCount)
	}
	if periodsPerYear <= 0 {
		return amountMinor
	}
	return int64(float64(amountMinor) * periodsPerYear / 12)
}

func toProtoSubscriptionStatusView(s db.Subscription, groupID pgtype.UUID, paid, payments int64, asOf time.Time) *financev1.SubscriptionStatusView {
	monthlyApprox := monthlyApproximate(s.AmountMinor, s.IntervalCount, s.IntervalUnit)
	return &financev1.SubscriptionStatusView{
		Subscription:       toProtoSubscription(s, groupID, paid, payments),
		NextDueOn:          pgconv.DateString(s.NextDueOn),
		Overdue:            s.Active && s.Status == subscriptionActive && s.NextDueOn.Valid && s.NextDueOn.Time.Before(asOf),
		MonthlyApproximate: money(monthlyApprox, s.CurrencyCode),
	}
}

func (h *Handler) subscriptionView(ctx context.Context, c caller, id pgtype.UUID) (*financev1.SubscriptionStatusView, error) {
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	asOf := h.today(hh)
	rows, err := h.q.ListVisibleSubscriptions(ctx, db.ListVisibleSubscriptionsParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(), IncludeInactive: true,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list subscriptions")
	}
	for _, r := range rows {
		if r.Subscription.ID == id {
			return toProtoSubscriptionStatusView(r.Subscription, r.GroupID, r.PaidMinor, r.Payments, asOf), nil
		}
	}
	return nil, notFound("subscription")
}

func (h *Handler) visibleSubscription(ctx context.Context, c caller, id pgtype.UUID) (db.Subscription, error) {
	row, err := h.q.GetVisibleSubscription(ctx, db.GetVisibleSubscriptionParams{
		ID: id, FamilyID: c.familyID, ViewerMemberID: c.memberID(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Subscription{}, notFound("subscription")
	}
	if err != nil {
		return db.Subscription{}, h.internal(ctx, err, "get subscription")
	}
	return row, nil
}

func (h *Handler) checkSubscriptionMember(ctx context.Context, c caller, memberID pgtype.UUID) error {
	member, err := h.q.GetMember(ctx, db.GetMemberParams{FamilyID: c.familyID, UserID: memberID})
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && member.Status != "active") {
		return invalid("member_id must belong to this family")
	}
	if err != nil {
		return h.internal(ctx, err, "get subscription member")
	}
	return nil
}

func (h *Handler) ListSubscriptions(
	ctx context.Context, req *connect.Request[financev1.ListSubscriptionsRequest],
) (*connect.Response[financev1.ListSubscriptionsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	asOf := h.today(hh)

	rows, err := h.q.ListVisibleSubscriptions(ctx, db.ListVisibleSubscriptionsParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		IncludeInactive: req.Msg.GetIncludeInactive(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list subscriptions")
	}
	out := make([]*financev1.SubscriptionStatusView, 0, len(rows))
	for _, r := range rows {
		out = append(out, toProtoSubscriptionStatusView(r.Subscription, r.GroupID, r.PaidMinor, r.Payments, asOf))
	}
	return connect.NewResponse(&financev1.ListSubscriptionsResponse{Subscriptions: out}), nil
}

func (h *Handler) CreateSubscription(
	ctx context.Context, req *connect.Request[financev1.CreateSubscriptionRequest],
) (*connect.Response[financev1.CreateSubscriptionResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg

	name := trimmed(msg.GetName())
	if name == "" {
		return nil, invalid("name is required")
	}
	if err := checkText("name", name, maxNameRunes); err != nil {
		return nil, err
	}
	amount, err := checkAmount(msg.GetAmount())
	if err != nil {
		return nil, err
	}
	if amount == 0 {
		return nil, invalid("amount must be greater than zero")
	}
	accountID, err := requireUUID("account_id", msg.GetAccountId())
	if err != nil {
		return nil, err
	}
	account, err := h.visibleAccount(ctx, c, accountID)
	if err != nil {
		return nil, err
	}
	if account.Archived {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("this account is archived; choose another account"))
	}
	if err := checkMoneyCurrency(msg.GetAmount(), account.CurrencyCode); err != nil {
		return nil, err
	}
	memberID := c.memberID()
	if trimmed(msg.GetMemberId()) != "" {
		if memberID, err = requireUUID("member_id", msg.GetMemberId()); err != nil {
			return nil, err
		}
		if err := h.checkSubscriptionMember(ctx, c, memberID); err != nil {
			return nil, err
		}
	}
	nextDue, err := requireDay("next_due_on", msg.GetNextDueOn(), h.today(hh), hh.loc)
	if err != nil {
		return nil, err
	}
	cadence := msg.GetCadence()
	interval := maxInt32(cadence.GetInterval(), 1)
	unit := recurrenceUnitFromProto(cadence.GetUnit())
	if unit == "" {
		unit = "month"
	}
	dayOfMonth := cadence.GetDayOfMonth()
	if dayOfMonth < 1 || dayOfMonth > 31 {
		return nil, invalid("cadence.day_of_month must be between 1 and 31")
	}
	dayOfWeek := weekdayFromProto(cadence.GetDayOfWeek())

	var endOn pgtype.Date
	if end := trimmed(msg.GetEndOn()); end != "" {
		day, ok := parseDay(end, hh.loc)
		if !ok {
			return nil, invalid("end_on must be a date as YYYY-MM-DD")
		}
		if day.Before(nextDue) {
			return nil, invalid("end_on must not be before next_due_on")
		}
		endOn = pgDate(day)
	}

	var created db.Subscription
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		categoryName := name
		if account.Visibility != visibilityShared {
			categoryName = "Особиста підписка"
		}
		category, err := createOwnedCategory(ctx, q, c.familyID, roleSubscriptions, categoryName, "credit-card")
		if err != nil {
			return err
		}
		created, err = q.CreateSubscription(ctx, db.CreateSubscriptionParams{
			FamilyID: c.familyID, Name: name, AmountMinor: amount,
			CurrencyCode: account.CurrencyCode, Type: kindExpense,
			CategoryID: category.ID, AccountID: accountID, MemberID: memberID,
			CreatedByUserID: c.userID, IntervalCount: interval, IntervalUnit: unit,
			DayOfMonth: dayOfMonth, DayOfWeek: dayOfWeek,
			NextDueOn: pgDate(nextDue), EndOn: endOn, AutoPost: msg.GetAutoPost(),
		})
		return err
	})
	if err != nil {
		return nil, h.internal(ctx, err, "create subscription")
	}

	h.catchUpSubscription(ctx, created.ID)

	view, err := h.subscriptionView(ctx, c, created.ID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&financev1.CreateSubscriptionResponse{Subscription: view.Subscription}), nil
}

func (h *Handler) UpdateSubscription(
	ctx context.Context, req *connect.Request[financev1.UpdateSubscriptionRequest],
) (*connect.Response[financev1.UpdateSubscriptionResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	id, err := requireUUID("subscription_id", msg.GetSubscriptionId())
	if err != nil {
		return nil, err
	}
	sub, err := h.visibleSubscription(ctx, c, id)
	if err != nil {
		return nil, err
	}

	params := db.UpdateSubscriptionParams{ID: sub.ID, FamilyID: c.familyID}
	if msg.Name != nil {
		name := trimmed(msg.GetName())
		if name == "" {
			return nil, invalid("name must not be empty")
		}
		if err := checkText("name", name, maxNameRunes); err != nil {
			return nil, err
		}
		params.Name = &name
	}
	if msg.Amount != nil {
		amount, err := checkAmount(msg.GetAmount())
		if err != nil {
			return nil, err
		}
		if amount == 0 {
			return nil, invalid("amount must be greater than zero")
		}
		if err := checkMoneyCurrency(msg.GetAmount(), sub.CurrencyCode); err != nil {
			return nil, err
		}
		params.AmountMinor = &amount
	}
	if msg.CategoryId != nil {
		categoryID, err := requireUUID("category_id", msg.GetCategoryId())
		if err != nil {
			return nil, err
		}
		if err := h.checkCategoryKind(ctx, c, categoryID, sub.Type); err != nil {
			return nil, err
		}
		params.CategoryID = categoryID
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
		if account.CurrencyCode != sub.CurrencyCode {
			return nil, invalid("account is in %s but this subscription is in %s",
				account.CurrencyCode, sub.CurrencyCode)
		}
		params.AccountID = accountID
	}
	if msg.MemberId != nil {
		memberID, err := requireUUID("member_id", msg.GetMemberId())
		if err != nil {
			return nil, err
		}
		if err := h.checkSubscriptionMember(ctx, c, memberID); err != nil {
			return nil, err
		}
		params.MemberID = memberID
	}
	if cadence := msg.GetCadence(); cadence != nil {
		interval := maxInt32(cadence.GetInterval(), 1)
		params.IntervalCount = &interval
		if unit := recurrenceUnitFromProto(cadence.GetUnit()); unit != "" {
			params.IntervalUnit = &unit
		}
		if cadence.GetDayOfMonth() < 1 || cadence.GetDayOfMonth() > 31 {
			return nil, invalid("cadence.day_of_month must be between 1 and 31")
		}
		day := cadence.GetDayOfMonth()
		params.DayOfMonth = &day
		weekday := weekdayFromProto(cadence.GetDayOfWeek())
		params.DayOfWeek = &weekday
	}
	if msg.DayOfMonth != nil {
		day := msg.GetDayOfMonth()
		if day < 1 || day > 31 {
			return nil, invalid("day_of_month must be between 1 and 31")
		}
		params.DayOfMonth = &day
	}
	if msg.NextDueOn != nil {
		day, ok := parseDay(trimmed(msg.GetNextDueOn()), hh.loc)
		if !ok {
			return nil, invalid("next_due_on must be a date as YYYY-MM-DD")
		}
		params.NextDueOn = pgDate(day)
	}
	if msg.EndOn != nil {
		if trimmed(msg.GetEndOn()) == "" {
			params.EndOn = pgtype.Date{}
		} else {
			day, ok := parseDay(trimmed(msg.GetEndOn()), hh.loc)
			if !ok {
				return nil, invalid("end_on must be a date as YYYY-MM-DD")
			}
			params.EndOn = pgDate(day)
		}
	}
	if msg.AutoPost != nil {
		autoPost := msg.GetAutoPost()
		params.AutoPost = &autoPost
	}
	if msg.Active != nil {
		active := msg.GetActive()
		params.Active = &active
	}

	row, err := h.q.UpdateSubscription(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("subscription")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "update subscription")
	}
	return connect.NewResponse(&financev1.UpdateSubscriptionResponse{
		Subscription: toProtoSubscription(row, pgtype.UUID{}, 0, 0),
	}), nil
}

func (h *Handler) CancelSubscription(
	ctx context.Context, req *connect.Request[financev1.CancelSubscriptionRequest],
) (*connect.Response[financev1.CancelSubscriptionResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requireUUID("subscription_id", req.Msg.GetSubscriptionId())
	if err != nil {
		return nil, err
	}
	sub, err := h.visibleSubscription(ctx, c, id)
	if err != nil {
		return nil, err
	}
	if sub.Status != subscriptionActive {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("this subscription is no longer active"))
	}
	status := subscriptionCancelled
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		if _, err := q.UpdateSubscription(ctx, db.UpdateSubscriptionParams{
			ID: sub.ID, FamilyID: c.familyID, Status: &status,
		}); err != nil {
			return err
		}
		return q.SetCategoryArchived(ctx, db.SetCategoryArchivedParams{
			ID: sub.CategoryID, FamilyID: c.familyID, Archived: true,
		})
	})
	if err != nil {
		return nil, h.internal(ctx, err, "cancel subscription")
	}

	view, err := h.subscriptionView(ctx, c, sub.ID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&financev1.CancelSubscriptionResponse{Subscription: view.Subscription}), nil
}

func (h *Handler) DeleteSubscription(
	ctx context.Context, req *connect.Request[financev1.DeleteSubscriptionRequest],
) (*connect.Response[financev1.DeleteSubscriptionResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requireUUID("subscription_id", req.Msg.GetSubscriptionId())
	if err != nil {
		return nil, err
	}
	sub, err := h.visibleSubscription(ctx, c, id)
	if err != nil {
		return nil, err
	}

	err = h.tx.InTx(ctx, func(q db.Querier) error {
		count, err := q.CountCategoryTransactions(ctx, db.CountCategoryTransactionsParams{
			CategoryID: sub.CategoryID, FamilyID: c.familyID,
		})
		if err != nil {
			return err
		}
		if count > 0 {
			return connect.NewError(connect.CodeFailedPrecondition,
				errors.New("this subscription already has payments; cancel it instead"))
		}
		if _, err := q.DeleteSubscription(ctx, db.DeleteSubscriptionParams{ID: sub.ID, FamilyID: c.familyID}); err != nil {
			return err
		}
		_, err = q.DeleteCategory(ctx, db.DeleteCategoryParams{ID: sub.CategoryID, FamilyID: c.familyID})
		return err
	})
	var cerr *connect.Error
	if errors.As(err, &cerr) {
		return nil, cerr
	}
	if err != nil {
		return nil, h.internal(ctx, err, "delete subscription")
	}
	return connect.NewResponse(&financev1.DeleteSubscriptionResponse{}), nil
}

func (h *Handler) PostSubscriptionOccurrence(
	ctx context.Context, req *connect.Request[financev1.PostSubscriptionOccurrenceRequest],
) (*connect.Response[financev1.PostSubscriptionOccurrenceResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	idStr := req.Msg.GetSubscriptionId()
	id, err := requireUUID("subscription_id", idStr)
	if err != nil {
		return nil, err
	}

	sub, err := h.visibleSubscription(ctx, c, id)
	if err != nil {
		return nil, err
	}
	if sub.Status != subscriptionActive || !sub.Active {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("this subscription is no longer active"))
	}
	dueOn, err := requireDay("due_on", msg.GetDueOn(), sub.NextDueOn.Time, hh.loc)
	if err != nil {
		return nil, err
	}
	if dueOn.Format(dateLayout) != pgconv.DateString(sub.NextDueOn) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("this due date has changed"))
	}

	amount := sub.AmountMinor
	if msg.GetAmountOverride() != nil {
		if amount, err = checkAmount(msg.GetAmountOverride()); err != nil {
			return nil, err
		}
		if err := checkMoneyCurrency(msg.GetAmountOverride(), sub.CurrencyCode); err != nil {
			return nil, err
		}
	}

	next := advanceDue(dueOn, sub.IntervalCount, sub.IntervalUnit, sub.DayOfMonth)

	var row db.Transaction
	var updated db.Subscription
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		locked, err := q.LockDueSubscription(ctx, sub.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return errSubscriptionMoved
		}
		if err != nil {
			return err
		}
		if !locked.NextDueOn.Time.Equal(sub.NextDueOn.Time) || !locked.Active || locked.Status != subscriptionActive {
			return errSubscriptionMoved
		}
		claimed, err := q.ClaimSubscriptionOccurrence(ctx, db.ClaimSubscriptionOccurrenceParams{
			SubscriptionID: sub.ID, DueOn: pgDate(dueOn),
		})
		if err != nil {
			return err
		}
		if claimed == 0 {
			updated, err = q.AdvanceSubscription(ctx, db.AdvanceSubscriptionParams{
				ID: sub.ID, NextDueOn: pgDate(next), LastPostedOn: sub.LastPostedOn,
				ExpectedDueOn: sub.NextDueOn,
			})
			return err
		}
		created, err := q.CreateTransaction(ctx, db.CreateTransactionParams{
			FamilyID: c.familyID, Type: sub.Type, AccountID: sub.AccountID,
			CategoryID: sub.CategoryID, AmountMinor: amount,
			CurrencyCode: sub.CurrencyCode, Note: sub.Name,
			OccurredOn: pgDate(dueOn), MemberID: sub.MemberID,
			CreatedByUserID: c.userID,
		})
		if err != nil {
			return err
		}
		row = created
		updated, err = q.AdvanceSubscription(ctx, db.AdvanceSubscriptionParams{
			ID: sub.ID, NextDueOn: pgDate(next), LastPostedOn: pgDate(dueOn),
			ExpectedDueOn: sub.NextDueOn,
		})
		return err
	})
	if errors.Is(err, errSubscriptionMoved) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("this due date has changed"))
	}
	if err != nil {
		return nil, h.internal(ctx, err, "post subscription occurrence")
	}
	if !row.ID.Valid {
		return connect.NewResponse(&financev1.PostSubscriptionOccurrenceResponse{
			NextDueOn: pgconv.DateString(updated.NextDueOn),
		}), nil
	}

	view, err := h.attachGroup(ctx, c, row)
	if err != nil {
		return nil, err
	}
	h.announceTransactionCreated(ctx, c, view)
	h.publish(ctx, subjectSubscriptionPosted, &financev1.SubscriptionPostedEvent{
		FamilyId:       c.family,
		SubscriptionId: pgconv.UUIDString(sub.ID),
		TransactionId:  pgconv.UUIDString(row.ID),
		DueOn:          dueOn.Format(dateLayout),
		NextDueOn:      pgconv.DateString(updated.NextDueOn),
		OccurredAt:     h.timestamp(),
	})

	return connect.NewResponse(&financev1.PostSubscriptionOccurrenceResponse{
		Transaction: toProtoTransaction(view),
		NextDueOn:   pgconv.DateString(updated.NextDueOn),
	}), nil
}

func (h *Handler) SkipSubscriptionOccurrence(
	ctx context.Context, req *connect.Request[financev1.SkipSubscriptionOccurrenceRequest],
) (*connect.Response[financev1.SkipSubscriptionOccurrenceResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	idStr := req.Msg.GetSubscriptionId()
	id, err := requireUUID("subscription_id", idStr)
	if err != nil {
		return nil, err
	}
	sub, err := h.visibleSubscription(ctx, c, id)
	if err != nil {
		return nil, err
	}
	if sub.Status != subscriptionActive || !sub.Active {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("this subscription is no longer active"))
	}
	dueOn, err := requireDay("due_on", req.Msg.GetDueOn(), sub.NextDueOn.Time, hh.loc)
	if err != nil {
		return nil, err
	}
	if dueOn.Format(dateLayout) != pgconv.DateString(sub.NextDueOn) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("this due date has changed"))
	}

	next := advanceDue(dueOn, sub.IntervalCount, sub.IntervalUnit, sub.DayOfMonth)
	updated, err := h.q.AdvanceSubscription(ctx, db.AdvanceSubscriptionParams{
		ID: sub.ID, NextDueOn: pgDate(next), LastPostedOn: pgtype.Date{},
		ExpectedDueOn: sub.NextDueOn,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("subscription")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "skip subscription occurrence")
	}

	h.publish(ctx, subjectSubscriptionSkipped, &financev1.SubscriptionSkippedEvent{
		FamilyId:       c.family,
		SubscriptionId: pgconv.UUIDString(sub.ID),
		DueOn:          dueOn.Format(dateLayout),
		NextDueOn:      pgconv.DateString(updated.NextDueOn),
		OccurredAt:     h.timestamp(),
	})

	return connect.NewResponse(&financev1.SkipSubscriptionOccurrenceResponse{
		NextDueOn: pgconv.DateString(updated.NextDueOn),
	}), nil
}

type postedSubscription struct {
	subscription db.Subscription
	transaction  db.Transaction
	hh           household
	before       budgetImpact
}

func (h *Handler) PostDueSubscriptions(ctx context.Context) (int, error) {
	latest := calendarDay(h.now().UTC()).AddDate(0, 0, 1)
	due, err := h.q.ListDueSubscriptions(ctx, db.ListDueSubscriptionsParams{
		LatestDueOn: pgDate(latest), MaxRows: subscriptionDueBatchSize,
	})
	if err != nil {
		return 0, err
	}
	posted := 0
	for _, d := range due {
		posted += h.catchUpSubscription(ctx, d.ID)
	}
	return posted, nil
}

func (h *Handler) catchUpSubscription(ctx context.Context, id pgtype.UUID) int {
	posted := 0
	for range maxSubscriptionMonths + 1 {
		outcome, err := h.postNextSubscription(ctx, id)
		if err != nil {
			h.log.ErrorContext(ctx, "post subscription",
				slog.String("subscription_id", pgconv.UUIDString(id)), slog.String("error", err.Error()))
			break
		}
		if outcome == postStop {
			break
		}
		if outcome == postDone {
			posted++
		}
	}
	return posted
}

var errSubscriptionMoved = errors.New("subscription changed since it was read")

func (h *Handler) postNextSubscription(ctx context.Context, id pgtype.UUID) (postOutcome, error) {
	pre, err := h.q.GetSubscriptionByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return postStop, nil
	}
	if err != nil {
		return postStop, err
	}
	if pre.Status != subscriptionActive || !pre.Active || !pre.AutoPost {
		return postStop, nil
	}
	if pre.EndOn.Valid && pre.NextDueOn.Time.After(pre.EndOn.Time) {
		return postStop, nil
	}
	settings, err := h.q.GetFinanceSettings(ctx, pre.FamilyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return postStop, nil
	}
	if err != nil {
		return postStop, err
	}
	loc, lerr := time.LoadLocation(settings.Timezone)
	if lerr != nil {
		loc = time.UTC
	}
	hh := household{settings: settings, loc: loc}
	due := calendarDay(pre.NextDueOn.Time)
	if due.After(calendarDay(h.now().In(loc))) {
		return postStop, nil
	}

	c := caller{
		family: pgconv.UUIDString(pre.FamilyID), user: pgconv.UUIDString(pre.CreatedByUserID),
		familyID: pre.FamilyID, userID: pre.CreatedByUserID,
	}
	before, err := h.affectedBudgets(ctx, c, hh, pre.CategoryID, due)
	if err != nil {
		return postStop, err
	}

	var out *postedSubscription
	var alreadyClaimed bool
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		row, err := q.LockDueSubscription(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if !row.AutoPost || !row.NextDueOn.Time.Equal(pre.NextDueOn.Time) || row.AmountMinor != pre.AmountMinor {
			return errSubscriptionMoved
		}
		claimed, err := q.ClaimSubscriptionOccurrence(ctx, db.ClaimSubscriptionOccurrenceParams{
			SubscriptionID: row.ID, DueOn: pgDate(due),
		})
		if err != nil {
			return err
		}
		if claimed == 0 {
			alreadyClaimed = true
			_, err = q.AdvanceSubscription(ctx, db.AdvanceSubscriptionParams{
				ID:           row.ID,
				NextDueOn:    pgDate(advanceDue(due, row.IntervalCount, row.IntervalUnit, row.DayOfMonth)),
				LastPostedOn: row.LastPostedOn, ExpectedDueOn: row.NextDueOn,
			})
			return err
		}
		_, err = q.SumSubscriptionPayments(ctx, db.SumSubscriptionPaymentsParams{
			FamilyID: row.FamilyID, CategoryID: row.CategoryID, CurrencyCode: row.CurrencyCode,
		})
		if err != nil {
			return err
		}

		amount := row.AmountMinor
		result := &postedSubscription{hh: hh, before: before}

		if amount > 0 {
			result.transaction, err = q.CreateTransaction(ctx, db.CreateTransactionParams{
				FamilyID: row.FamilyID, Type: row.Type, AccountID: row.AccountID,
				CategoryID: row.CategoryID, AmountMinor: amount, CurrencyCode: row.CurrencyCode,
				Note: row.Name, OccurredOn: pgDate(due), MemberID: row.MemberID,
				CreatedByUserID: row.CreatedByUserID,
			})
			if err != nil {
				return err
			}
		}

		next := advanceDue(due, row.IntervalCount, row.IntervalUnit, row.DayOfMonth)
		result.subscription, err = q.AdvanceSubscription(ctx, db.AdvanceSubscriptionParams{
			ID: row.ID, NextDueOn: pgDate(next), LastPostedOn: pgDate(due),
			ExpectedDueOn: row.NextDueOn,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return errSubscriptionMoved
		}
		if err != nil {
			return err
		}
		out = result
		return nil
	})
	if errors.Is(err, errSubscriptionMoved) {
		return postRetry, nil
	}
	if err == nil && alreadyClaimed {
		return postRetry, nil
	}
	if err != nil || out == nil {
		return postStop, err
	}
	h.announceSubscriptionPosting(ctx, c, out)
	return postDone, nil
}

func (h *Handler) announceSubscriptionPosting(ctx context.Context, c caller, p *postedSubscription) {
	sub := p.subscription
	if p.transaction.ID.Valid {
		if view, err := h.attachGroup(ctx, c, p.transaction); err == nil {
			h.announceTransactionCreated(ctx, c, view)
		}
		after, err := h.affectedBudgets(ctx, c, p.hh, sub.CategoryID, p.transaction.OccurredOn.Time)
		if err == nil {
			private := true
			if account, aerr := h.visibleAccount(ctx, c, sub.AccountID); aerr == nil {
				private = account.Visibility != visibilityShared
			}
			h.announceBudgetChanges(ctx, c, p.before, after, pgconv.UUIDString(p.transaction.ID), private)
		}
	}
}
