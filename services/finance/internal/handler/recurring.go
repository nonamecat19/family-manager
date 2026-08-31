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

// advanceDue is the whole recurrence engine: step the due date forward by one interval, then
// re-apply the day anchor. Deliberately not rrule — a rent payment does not need it, and the
// home-screen widget would have to reimplement whatever rrule subset the server chose.
func advanceDue(from time.Time, interval int32, unit string, dayOfMonth int32) time.Time {
	if interval <= 0 {
		interval = 1
	}
	next := from
	switch unit {
	case "day":
		next = from.AddDate(0, 0, int(interval))
	case "week":
		next = from.AddDate(0, 0, 7*int(interval))
	case "year":
		next = addMonthsClamped(from, 12*int(interval))
	default: // month
		next = addMonthsClamped(from, int(interval))
	}
	// A payment anchored on the 31st must land on the 31st of the months that have one and on
	// the last day of the months that do not — never skip a month, and never drift earlier.
	if dayOfMonth > 0 && (unit == "month" || unit == "year") {
		first := time.Date(next.Year(), next.Month(), 1, 0, 0, 0, 0, next.Location())
		last := first.AddDate(0, 1, -1).Day()
		day := int(dayOfMonth)
		if day > last {
			day = last
		}
		next = time.Date(next.Year(), next.Month(), day, 0, 0, 0, 0, next.Location())
	}
	return next
}

// visibleRecurring is to a schedule what visibleAccount is to an account: the one read every
// path takes before touching a recurring payment, so "which schedules exist" is answered by
// the same predicate as "which schedules I may list".
func (h *Handler) visibleRecurring(ctx context.Context, c caller, id pgtype.UUID) (db.RecurringPayment, error) {
	row, err := h.q.GetVisibleRecurringPayment(ctx, db.GetVisibleRecurringPaymentParams{
		ID: id, FamilyID: c.familyID, ViewerMemberID: c.memberID(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.RecurringPayment{}, notFound("recurring payment")
	}
	if err != nil {
		return db.RecurringPayment{}, h.internal(ctx, err, "get recurring payment")
	}
	return row, nil
}

func (h *Handler) ListRecurringPayments(
	ctx context.Context, req *connect.Request[financev1.ListRecurringPaymentsRequest],
) (*connect.Response[financev1.ListRecurringPaymentsResponse], error) {
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

	rows, err := h.q.ListVisibleRecurringPayments(ctx, db.ListVisibleRecurringPaymentsParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		IncludeInactive: req.Msg.GetIncludeInactive(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list recurring payments")
	}
	out := make([]*financev1.RecurringPaymentStatus, 0, len(rows))
	for _, r := range rows {
		out = append(out, &financev1.RecurringPaymentStatus{
			Payment:   toProtoRecurring(r),
			NextDueOn: pgconv.DateString(r.NextDueOn),
			Overdue:   r.Active && r.NextDueOn.Valid && r.NextDueOn.Time.Before(asOf),
		})
	}
	return connect.NewResponse(&financev1.ListRecurringPaymentsResponse{Payments: out}), nil
}

func (h *Handler) CreateRecurringPayment(
	ctx context.Context, req *connect.Request[financev1.CreateRecurringPaymentRequest],
) (*connect.Response[financev1.CreateRecurringPaymentResponse], error) {
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
	if err := checkMoneyCurrency(msg.GetAmount(), account.CurrencyCode); err != nil {
		return nil, err
	}
	categoryID, err := optionalUUID("category_id", msg.GetCategoryId())
	if err != nil {
		return nil, err
	}
	// Same rule as a template: the schedule posts this pair onto a transaction every occurrence.
	if err := h.checkCategoryKind(ctx, c, categoryID, txTypeFromProto(msg.GetType())); err != nil {
		return nil, err
	}
	memberID := c.memberID()
	if trimmed(msg.GetMemberId()) != "" {
		if memberID, err = requireUUID("member_id", msg.GetMemberId()); err != nil {
			return nil, err
		}
	}
	nextDue, err := requireDay("next_due_on", msg.GetNextDueOn(), h.today(hh), hh.loc)
	if err != nil {
		return nil, err
	}
	cadence := msg.GetCadence()
	unit := recurrenceUnitFromProto(cadence.GetUnit())
	if unit == "" {
		unit = "month"
	}
	if cadence.GetDayOfMonth() < 0 || cadence.GetDayOfMonth() > 31 {
		return nil, invalid("cadence.day_of_month must be between 0 and 31")
	}

	params := db.CreateRecurringPaymentParams{
		FamilyID: c.familyID, Name: name, AmountMinor: amount,
		CurrencyCode: account.CurrencyCode, Type: txTypeFromProto(msg.GetType()),
		CategoryID: categoryID, AccountID: accountID, MemberID: memberID,
		IntervalCount: maxInt32(cadence.GetInterval(), 1), IntervalUnit: unit,
		DayOfMonth: cadence.GetDayOfMonth(), DayOfWeek: weekdayFromProto(cadence.GetDayOfWeek()),
		NextDueOn: pgDate(nextDue), AutoPost: msg.GetAutoPost(),
	}
	if end := trimmed(msg.GetEndOn()); end != "" {
		day, ok := parseDay(end, hh.loc)
		if !ok {
			return nil, invalid("end_on must be a date as YYYY-MM-DD")
		}
		params.EndOn = pgDate(day)
	}

	row, err := h.q.CreateRecurringPayment(ctx, params)
	if err != nil {
		return nil, h.internal(ctx, err, "create recurring payment")
	}
	return connect.NewResponse(&financev1.CreateRecurringPaymentResponse{
		Payment: toProtoRecurring(row),
	}), nil
}

func maxInt32(v, min int32) int32 {
	if v < min {
		return min
	}
	return v
}

func (h *Handler) UpdateRecurringPayment(
	ctx context.Context, req *connect.Request[financev1.UpdateRecurringPaymentRequest],
) (*connect.Response[financev1.UpdateRecurringPaymentResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	id, err := requireUUID("recurring_id", msg.GetRecurringId())
	if err != nil {
		return nil, err
	}

	// The read is the guard: without it a no-op patch against a schedule on another member's
	// private account would answer with the whole row, turning a write RPC into the read the
	// boundary exists to refuse — and would let one member retarget or reprice that schedule.
	payment, err := h.visibleRecurring(ctx, c, id)
	if err != nil {
		return nil, err
	}

	params := db.UpdateRecurringPaymentParams{ID: id, FamilyID: c.familyID}
	if msg.Name != nil {
		name := trimmed(msg.GetName())
		if name == "" {
			return nil, invalid("name is required")
		}
		if err := checkText("name", name, maxNameRunes); err != nil {
			return nil, err
		}
		params.Name = &name
	}
	// The account is resolved before the amount: moving the schedule to an account in another
	// currency is what decides which currency the new amount has to be in.
	currency := payment.CurrencyCode
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
		// Same rule as a template: the schedule follows its account's currency, and the move
		// has to carry a new amount rather than relabelling the old one.
		if account.CurrencyCode != currency {
			if msg.Amount == nil {
				return nil, invalid(
					"this account is in %s: send the amount in %s as well",
					account.CurrencyCode, account.CurrencyCode)
			}
			currency = account.CurrencyCode
			params.CurrencyCode = &currency
		}
	}
	if msg.Amount != nil {
		amount, err := checkAmount(msg.GetAmount())
		if err != nil {
			return nil, err
		}
		if err := checkMoneyCurrency(msg.GetAmount(), currency); err != nil {
			return nil, err
		}
		if amount <= 0 {
			return nil, invalid("amount must be greater than zero")
		}
		params.AmountMinor = &amount
	}
	if msg.CategoryId != nil {
		if params.CategoryID, err = optionalUUID("category_id", msg.GetCategoryId()); err != nil {
			return nil, err
		}
		if err := h.checkCategoryKind(ctx, c, params.CategoryID, payment.Type); err != nil {
			return nil, err
		}
	}
	if msg.MemberId != nil {
		if params.MemberID, err = requireUUID("member_id", msg.GetMemberId()); err != nil {
			return nil, err
		}
	}
	if cadence := msg.GetCadence(); cadence != nil {
		interval := maxInt32(cadence.GetInterval(), 1)
		params.IntervalCount = &interval
		if unit := recurrenceUnitFromProto(cadence.GetUnit()); unit != "" {
			params.IntervalUnit = &unit
		}
		if cadence.GetDayOfMonth() < 0 || cadence.GetDayOfMonth() > 31 {
			return nil, invalid("cadence.day_of_month must be between 0 and 31")
		}
		day := cadence.GetDayOfMonth()
		params.DayOfMonth = &day
		weekday := weekdayFromProto(cadence.GetDayOfWeek())
		params.DayOfWeek = &weekday
	}
	if msg.NextDueOn != nil {
		day, ok := parseDay(trimmed(msg.GetNextDueOn()), hh.loc)
		if !ok {
			return nil, invalid("next_due_on must be a date as YYYY-MM-DD")
		}
		params.NextDueOn = pgDate(day)
	}
	if msg.EndOn != nil {
		day, ok := parseDay(trimmed(msg.GetEndOn()), hh.loc)
		if !ok {
			return nil, invalid("end_on must be a date as YYYY-MM-DD")
		}
		params.EndOn = pgDate(day)
	}
	if msg.AutoPost != nil {
		autoPost := msg.GetAutoPost()
		params.AutoPost = &autoPost
	}
	if msg.Active != nil {
		active := msg.GetActive()
		params.Active = &active
	}

	row, err := h.q.UpdateRecurringPayment(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("recurring payment")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "update recurring payment")
	}
	return connect.NewResponse(&financev1.UpdateRecurringPaymentResponse{
		Payment: toProtoRecurring(row),
	}), nil
}

func (h *Handler) DeleteRecurringPayment(
	ctx context.Context, req *connect.Request[financev1.DeleteRecurringPaymentRequest],
) (*connect.Response[financev1.DeleteRecurringPaymentResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requireUUID("recurring_id", req.Msg.GetRecurringId())
	if err != nil {
		return nil, err
	}
	// A delete that is refused with NotFound for an invisible id and accepted for a visible
	// one answers the same way as a delete of an id that never existed.
	if _, err := h.visibleRecurring(ctx, c, id); err != nil {
		return nil, err
	}
	rows, err := h.q.DeleteRecurringPayment(ctx, db.DeleteRecurringPaymentParams{
		ID: id, FamilyID: c.familyID,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "delete recurring payment")
	}
	if rows == 0 {
		return nil, notFound("recurring payment")
	}
	return connect.NewResponse(&financev1.DeleteRecurringPaymentResponse{}), nil
}

// PostRecurringOccurrence confirms one due date. Posting the same occurrence twice is a no-op
// rather than a duplicate row: a widget that retried on a flaky connection must not invent a
// second rent payment.
func (h *Handler) PostRecurringOccurrence(
	ctx context.Context, req *connect.Request[financev1.PostRecurringOccurrenceRequest],
) (*connect.Response[financev1.PostRecurringOccurrenceResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	id, err := requireUUID("recurring_id", msg.GetRecurringId())
	if err != nil {
		return nil, err
	}

	payment, err := h.visibleRecurring(ctx, c, id)
	if err != nil {
		return nil, err
	}
	dueOn, err := requireDay("due_on", msg.GetDueOn(), payment.NextDueOn.Time, hh.loc)
	if err != nil {
		return nil, err
	}

	amount := payment.AmountMinor
	if msg.GetAmountOverride() != nil {
		if amount, err = checkAmount(msg.GetAmountOverride()); err != nil {
			return nil, err
		}
		if err := checkMoneyCurrency(msg.GetAmountOverride(), payment.CurrencyCode); err != nil {
			return nil, err
		}
	}

	already, err := h.q.CountTransactionsForRecurringOccurrence(ctx,
		db.CountTransactionsForRecurringOccurrenceParams{
			FamilyID: c.familyID, RecurringID: payment.ID, OccurredOn: pgDate(dueOn),
		})
	if err != nil {
		return nil, h.internal(ctx, err, "count recurring occurrence")
	}
	if already > 0 {
		return connect.NewResponse(&financev1.PostRecurringOccurrenceResponse{
			NextDueOn: pgconv.DateString(payment.NextDueOn),
		}), nil
	}

	next := advanceDue(dueOn, payment.IntervalCount, payment.IntervalUnit, payment.DayOfMonth)

	var row db.Transaction
	var updated db.RecurringPayment
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		created, err := q.CreateTransaction(ctx, db.CreateTransactionParams{
			FamilyID: c.familyID, Type: payment.Type, AccountID: payment.AccountID,
			CategoryID: payment.CategoryID, AmountMinor: amount,
			CurrencyCode: payment.CurrencyCode, Note: payment.Name,
			OccurredOn: pgDate(dueOn), MemberID: payment.MemberID,
			CreatedByUserID: c.userID, RecurringID: payment.ID,
		})
		if err != nil {
			return err
		}
		row = created
		updated, err = q.AdvanceRecurringPayment(ctx, db.AdvanceRecurringPaymentParams{
			ID: payment.ID, FamilyID: c.familyID,
			NextDueOn: pgDate(next), LastPostedOn: pgDate(dueOn),
		})
		return err
	})
	if err != nil {
		return nil, h.internal(ctx, err, "post recurring occurrence")
	}

	view, err := h.attachGroup(ctx, c, row)
	if err != nil {
		return nil, err
	}
	h.announceTransactionCreated(ctx, c, view)
	h.publish(ctx, subjectRecurringPosted, &financev1.RecurringPostedEvent{
		FamilyId:      c.family,
		RecurringId:   pgconv.UUIDString(payment.ID),
		TransactionId: pgconv.UUIDString(row.ID),
		DueOn:         dueOn.Format(dateLayout),
		NextDueOn:     pgconv.DateString(updated.NextDueOn),
		OccurredAt:    h.timestamp(),
	})

	return connect.NewResponse(&financev1.PostRecurringOccurrenceResponse{
		Transaction: toProtoTransaction(view),
		NextDueOn:   pgconv.DateString(updated.NextDueOn),
	}), nil
}

// SkipRecurringOccurrence moves the due date on without writing a row: a subscription that was
// cancelled this month is not a transaction, and confirm-or-skip is the only safe default for
// a payment that may not have happened.
func (h *Handler) SkipRecurringOccurrence(
	ctx context.Context, req *connect.Request[financev1.SkipRecurringOccurrenceRequest],
) (*connect.Response[financev1.SkipRecurringOccurrenceResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	id, err := requireUUID("recurring_id", req.Msg.GetRecurringId())
	if err != nil {
		return nil, err
	}
	payment, err := h.visibleRecurring(ctx, c, id)
	if err != nil {
		return nil, err
	}
	dueOn, err := requireDay("due_on", req.Msg.GetDueOn(), payment.NextDueOn.Time, hh.loc)
	if err != nil {
		return nil, err
	}

	next := advanceDue(dueOn, payment.IntervalCount, payment.IntervalUnit, payment.DayOfMonth)
	updated, err := h.q.AdvanceRecurringPayment(ctx, db.AdvanceRecurringPaymentParams{
		ID: payment.ID, FamilyID: c.familyID,
		NextDueOn: pgDate(next), LastPostedOn: pgtype.Date{},
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound("recurring payment")
	}
	if err != nil {
		return nil, h.internal(ctx, err, "skip recurring occurrence")
	}
	return connect.NewResponse(&financev1.SkipRecurringOccurrenceResponse{
		NextDueOn: pgconv.DateString(updated.NextDueOn),
	}), nil
}
