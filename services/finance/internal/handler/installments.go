package handler

import (
	"context"
	"errors"
	"fmt"
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
	subjectInstallmentPaidOff events.Subject = "finance.installment.paid_off"

	installmentActive    = "active"
	installmentPaidOff   = "paid_off"
	installmentCancelled = "cancelled"

	maxInstallmentMonths = 120
	dueBatchSize         = 200
)

func installmentStatusToProto(s string) financev1.InstallmentStatus {
	switch s {
	case installmentActive:
		return financev1.InstallmentStatus_INSTALLMENT_STATUS_ACTIVE
	case installmentPaidOff:
		return financev1.InstallmentStatus_INSTALLMENT_STATUS_PAID_OFF
	case installmentCancelled:
		return financev1.InstallmentStatus_INSTALLMENT_STATUS_CANCELLED
	}
	return financev1.InstallmentStatus_INSTALLMENT_STATUS_UNSPECIFIED
}

func toProtoInstallment(i db.Installment, groupID pgtype.UUID, paid, payments int64) *financev1.Installment {
	remaining := i.TotalMinor - paid
	if remaining < 0 {
		remaining = 0
	}
	return &financev1.Installment{
		Id:           pgconv.UUIDString(i.ID),
		FamilyId:     pgconv.UUIDString(i.FamilyID),
		Name:         i.Name,
		Total:        money(i.TotalMinor, i.CurrencyCode),
		Monthly:      money(i.MonthlyMinor, i.CurrencyCode),
		Months:       i.Months,
		Paid:         money(paid, i.CurrencyCode),
		Remaining:    money(remaining, i.CurrencyCode),
		PaymentsMade: int32(payments),
		PurchasedOn:  pgconv.DateString(i.PurchasedOn),
		DayOfMonth:   i.DayOfMonth,
		NextDueOn:    pgconv.DateString(i.NextDueOn),
		AccountId:    pgconv.UUIDString(i.AccountID),
		CategoryId:   pgconv.UUIDString(i.CategoryID),
		GroupId:      pgconv.UUIDString(groupID),
		MemberId:     pgconv.UUIDString(i.MemberID),
		Status:       installmentStatusToProto(i.Status),
		CreatedAt:    pgconv.Timestamp(i.CreatedAt),
		UpdatedAt:    pgconv.Timestamp(i.UpdatedAt),
	}
}

func ceilDiv(a, b int64) int64 { return (a + b - 1) / b }

func (h *Handler) installmentView(ctx context.Context, c caller, id pgtype.UUID) (*financev1.Installment, error) {
	rows, err := h.q.ListVisibleInstallments(ctx, db.ListVisibleInstallmentsParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(), IncludeClosed: true,
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list installments")
	}
	for _, r := range rows {
		if r.Installment.ID == id {
			return toProtoInstallment(r.Installment, r.GroupID, r.PaidMinor, r.Payments), nil
		}
	}
	return nil, notFound("installment")
}

func (h *Handler) visibleInstallment(ctx context.Context, c caller, raw string) (db.Installment, error) {
	id, err := requireUUID("installment_id", raw)
	if err != nil {
		return db.Installment{}, err
	}
	row, err := h.q.GetVisibleInstallment(ctx, db.GetVisibleInstallmentParams{
		ID: id, FamilyID: c.familyID, ViewerMemberID: c.memberID(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Installment{}, notFound("installment")
	}
	if err != nil {
		return db.Installment{}, h.internal(ctx, err, "get installment")
	}
	return row, nil
}

func (h *Handler) ListInstallments(
	ctx context.Context, req *connect.Request[financev1.ListInstallmentsRequest],
) (*connect.Response[financev1.ListInstallmentsResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.q.ListVisibleInstallments(ctx, db.ListVisibleInstallmentsParams{
		FamilyID: c.familyID, ViewerMemberID: c.memberID(),
		IncludeClosed: req.Msg.GetIncludeClosed(),
	})
	if err != nil {
		return nil, h.internal(ctx, err, "list installments")
	}
	out := make([]*financev1.Installment, 0, len(rows))
	for _, r := range rows {
		out = append(out, toProtoInstallment(r.Installment, r.GroupID, r.PaidMinor, r.Payments))
	}
	return connect.NewResponse(&financev1.ListInstallmentsResponse{Installments: out}), nil
}

func (h *Handler) CreateInstallment(
	ctx context.Context, req *connect.Request[financev1.CreateInstallmentRequest],
) (*connect.Response[financev1.CreateInstallmentResponse], error) {
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
	total, err := checkAmount(msg.GetTotal())
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return nil, invalid("total must be greater than zero")
	}
	months := msg.GetMonths()
	if months < 1 || months > maxInstallmentMonths {
		return nil, invalid("months must be between 1 and %d", maxInstallmentMonths)
	}
	monthly := ceilDiv(total, int64(months))
	if msg.GetMonthly() != nil && msg.GetMonthly().GetAmountMinor() != 0 {
		if monthly, err = checkAmount(msg.GetMonthly()); err != nil {
			return nil, err
		}
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
	if err := checkMoneyCurrency(msg.GetTotal(), account.CurrencyCode); err != nil {
		return nil, err
	}
	if msg.GetMonthly() != nil {
		if err := checkMoneyCurrency(msg.GetMonthly(), account.CurrencyCode); err != nil {
			return nil, err
		}
	}
	memberID := c.memberID()
	if trimmed(msg.GetMemberId()) != "" {
		if memberID, err = requireUUID("member_id", msg.GetMemberId()); err != nil {
			return nil, err
		}
	}
	purchasedOn, err := requireDay("purchased_on", msg.GetPurchasedOn(), h.today(hh), hh.loc)
	if err != nil {
		return nil, err
	}
	firstDue, err := requireDay("first_due_on", msg.GetFirstDueOn(), addMonthsClamped(purchasedOn, 1), hh.loc)
	if err != nil {
		return nil, err
	}

	var created db.Installment
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		category, err := createOwnedCategory(ctx, q, c.familyID, roleInstallments, name, "credit-card")
		if err != nil {
			return err
		}
		created, err = q.CreateInstallment(ctx, db.CreateInstallmentParams{
			FamilyID: c.familyID, Name: name, TotalMinor: total, MonthlyMinor: monthly,
			Months: months, CurrencyCode: account.CurrencyCode, AccountID: accountID,
			CategoryID: category.ID, MemberID: memberID, CreatedByUserID: c.userID,
			PurchasedOn: pgDate(purchasedOn), DayOfMonth: int32(firstDue.Day()),
			NextDueOn: pgDate(firstDue),
		})
		return err
	})
	if err != nil {
		return nil, h.internal(ctx, err, "create installment")
	}

	h.catchUpInstallment(ctx, created.ID)

	view, err := h.installmentView(ctx, c, created.ID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&financev1.CreateInstallmentResponse{Installment: view}), nil
}

func (h *Handler) UpdateInstallment(
	ctx context.Context, req *connect.Request[financev1.UpdateInstallmentRequest],
) (*connect.Response[financev1.UpdateInstallmentResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	hh, err := h.household(ctx, c)
	if err != nil {
		return nil, err
	}
	msg := req.Msg
	inst, err := h.visibleInstallment(ctx, c, msg.GetInstallmentId())
	if err != nil {
		return nil, err
	}

	params := db.UpdateInstallmentParams{ID: inst.ID, FamilyID: c.familyID}
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
	if msg.GetMonthly() != nil {
		monthly, err := checkAmount(msg.GetMonthly())
		if err != nil {
			return nil, err
		}
		if monthly == 0 {
			return nil, invalid("monthly must be greater than zero")
		}
		if err := checkMoneyCurrency(msg.GetMonthly(), inst.CurrencyCode); err != nil {
			return nil, err
		}
		params.MonthlyMinor = &monthly
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
		if account.CurrencyCode != inst.CurrencyCode {
			return nil, invalid("account is in %s but this installment is in %s",
				account.CurrencyCode, inst.CurrencyCode)
		}
		params.AccountID = accountID
	}
	if msg.MemberId != nil {
		memberID, err := requireUUID("member_id", msg.GetMemberId())
		if err != nil {
			return nil, err
		}
		params.MemberID = memberID
	}
	if msg.NextDueOn != nil {
		next, ok := parseDay(trimmed(msg.GetNextDueOn()), hh.loc)
		if !ok {
			return nil, invalid("next_due_on must be a date as YYYY-MM-DD")
		}
		current := calendarDay(inst.NextDueOn.Time)
		monthStart := time.Date(current.Year(), current.Month(), 1, 0, 0, 0, 0, time.UTC)
		if calendarDay(next).Before(monthStart) {
			sums, err := h.q.SumInstallmentPayments(ctx, db.SumInstallmentPaymentsParams{
				FamilyID: inst.FamilyID, CategoryID: inst.CategoryID, CurrencyCode: inst.CurrencyCode,
			})
			if err != nil {
				return nil, h.internal(ctx, err, "sum installment payments")
			}
			if sums.Payments > 0 {
				return nil, invalid("next_due_on must not move back into a month that is already paid")
			}
		}
		params.NextDueOn = pgDate(next)
		day := int32(next.Day())
		params.DayOfMonth = &day
	}

	err = h.tx.InTx(ctx, func(q db.Querier) error {
		if _, err := q.UpdateInstallment(ctx, params); err != nil {
			return err
		}
		if params.Name != nil {
			if _, err := q.UpdateCategory(ctx, db.UpdateCategoryParams{
				ID: inst.CategoryID, FamilyID: c.familyID, Name: params.Name,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if pgErrorCode(err) == pgUniqueViolation {
		return nil, connect.NewError(connect.CodeAlreadyExists,
			errors.New("a category with this name already exists"))
	}
	if err != nil {
		return nil, h.internal(ctx, err, "update installment")
	}

	view, err := h.installmentView(ctx, c, inst.ID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&financev1.UpdateInstallmentResponse{Installment: view}), nil
}

func (h *Handler) CancelInstallment(
	ctx context.Context, req *connect.Request[financev1.CancelInstallmentRequest],
) (*connect.Response[financev1.CancelInstallmentResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	inst, err := h.visibleInstallment(ctx, c, req.Msg.GetInstallmentId())
	if err != nil {
		return nil, err
	}
	if inst.Status != installmentActive {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errors.New("this installment is no longer active"))
	}
	status := installmentCancelled
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		if _, err := q.UpdateInstallment(ctx, db.UpdateInstallmentParams{
			ID: inst.ID, FamilyID: c.familyID, Status: &status,
		}); err != nil {
			return err
		}
		return q.SetCategoryArchived(ctx, db.SetCategoryArchivedParams{
			ID: inst.CategoryID, FamilyID: c.familyID, Archived: true,
		})
	})
	if err != nil {
		return nil, h.internal(ctx, err, "cancel installment")
	}

	view, err := h.installmentView(ctx, c, inst.ID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&financev1.CancelInstallmentResponse{Installment: view}), nil
}

func (h *Handler) DeleteInstallment(
	ctx context.Context, req *connect.Request[financev1.DeleteInstallmentRequest],
) (*connect.Response[financev1.DeleteInstallmentResponse], error) {
	c, err := h.caller(ctx)
	if err != nil {
		return nil, err
	}
	inst, err := h.visibleInstallment(ctx, c, req.Msg.GetInstallmentId())
	if err != nil {
		return nil, err
	}

	err = h.tx.InTx(ctx, func(q db.Querier) error {
		count, err := q.CountCategoryTransactions(ctx, db.CountCategoryTransactionsParams{
			CategoryID: inst.CategoryID, FamilyID: c.familyID,
		})
		if err != nil {
			return err
		}
		if count > 0 {
			return connect.NewError(connect.CodeFailedPrecondition,
				errors.New("this installment already has payments; cancel it instead"))
		}
		if _, err := q.DeleteInstallment(ctx, db.DeleteInstallmentParams{ID: inst.ID, FamilyID: c.familyID}); err != nil {
			return err
		}
		_, err = q.DeleteCategory(ctx, db.DeleteCategoryParams{ID: inst.CategoryID, FamilyID: c.familyID})
		return err
	})
	var cerr *connect.Error
	if errors.As(err, &cerr) {
		return nil, cerr
	}
	if err != nil {
		return nil, h.internal(ctx, err, "delete installment")
	}
	return connect.NewResponse(&financev1.DeleteInstallmentResponse{}), nil
}

type postedInstallment struct {
	installment db.Installment
	transaction db.Transaction
	paidOff     bool
	hh          household
	before      budgetImpact
}

func (h *Handler) PostDueInstallments(ctx context.Context) (int, error) {
	latest := calendarDay(h.now().UTC()).AddDate(0, 0, 1)
	due, err := h.q.ListDueInstallments(ctx, db.ListDueInstallmentsParams{
		LatestDueOn: pgDate(latest), MaxRows: dueBatchSize,
	})
	if err != nil {
		return 0, err
	}
	posted := 0
	for _, d := range due {
		posted += h.catchUpInstallment(ctx, d.ID)
	}
	return posted, nil
}

type postOutcome int

const (
	postStop postOutcome = iota
	postDone
	postRetry
)

func (h *Handler) catchUpInstallment(ctx context.Context, id pgtype.UUID) int {
	posted := 0
	for range maxInstallmentMonths + 1 {
		outcome, err := h.postNextInstallment(ctx, id)
		if err != nil {
			h.log.ErrorContext(ctx, "post installment",
				slog.String("installment_id", pgconv.UUIDString(id)), slog.String("error", err.Error()))
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

var errInstallmentMoved = errors.New("installment changed since it was read")

func calendarDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func (h *Handler) postNextInstallment(ctx context.Context, id pgtype.UUID) (postOutcome, error) {
	pre, err := h.q.GetInstallmentByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return postStop, nil
	}
	if err != nil {
		return postStop, err
	}
	if pre.Status != installmentActive {
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

	var out *postedInstallment
	err = h.tx.InTx(ctx, func(q db.Querier) error {
		row, err := q.LockDueInstallment(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if !row.NextDueOn.Time.Equal(pre.NextDueOn.Time) || row.MonthlyMinor != pre.MonthlyMinor {
			return errInstallmentMoved
		}
		sums, err := q.SumInstallmentPayments(ctx, db.SumInstallmentPaymentsParams{
			FamilyID: row.FamilyID, CategoryID: row.CategoryID, CurrencyCode: row.CurrencyCode,
		})
		if err != nil {
			return err
		}

		remaining := row.TotalMinor - sums.PaidMinor
		amount := min(row.MonthlyMinor, remaining)
		status := installmentActive
		if amount >= remaining {
			status = installmentPaidOff
		}
		result := &postedInstallment{paidOff: status == installmentPaidOff, hh: hh, before: before}

		if amount > 0 {
			result.transaction, err = q.CreateTransaction(ctx, db.CreateTransactionParams{
				FamilyID: row.FamilyID, Type: kindExpense, AccountID: row.AccountID,
				CategoryID: row.CategoryID, AmountMinor: amount, CurrencyCode: row.CurrencyCode,
				Note:       fmt.Sprintf("%s %d/%d", row.Name, sums.Payments+1, row.Months),
				OccurredOn: pgDate(due), MemberID: row.MemberID, CreatedByUserID: row.CreatedByUserID,
			})
			if err != nil {
				return err
			}
		}

		next := advanceDue(due, 1, "month", row.DayOfMonth)
		result.installment, err = q.AdvanceInstallment(ctx, db.AdvanceInstallmentParams{
			ID: row.ID, NextDueOn: pgDate(next), Status: status, ExpectedDueOn: row.NextDueOn,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return errInstallmentMoved
		}
		if err != nil {
			return err
		}
		if result.paidOff {
			if err := q.SetCategoryArchived(ctx, db.SetCategoryArchivedParams{
				ID: row.CategoryID, FamilyID: row.FamilyID, Archived: true,
			}); err != nil {
				return err
			}
		}
		out = result
		return nil
	})
	if errors.Is(err, errInstallmentMoved) {
		return postRetry, nil
	}
	if err != nil || out == nil {
		return postStop, err
	}
	h.announceInstallmentPosting(ctx, c, out)
	return postDone, nil
}

func (h *Handler) announceInstallmentPosting(ctx context.Context, c caller, p *postedInstallment) {
	inst := p.installment
	if p.transaction.ID.Valid {
		if view, err := h.attachGroup(ctx, c, p.transaction); err == nil {
			h.announceTransactionCreated(ctx, c, view)
		}
		after, err := h.affectedBudgets(ctx, c, p.hh, inst.CategoryID, p.transaction.OccurredOn.Time)
		if err == nil {
			private := true
			if account, aerr := h.visibleAccount(ctx, c, inst.AccountID); aerr == nil {
				private = account.Visibility != visibilityShared
			}
			h.announceBudgetChanges(ctx, c, p.before, after, pgconv.UUIDString(p.transaction.ID), private)
		}
	}
	if p.paidOff {
		h.publish(ctx, subjectInstallmentPaidOff, &financev1.InstallmentPaidOffEvent{
			FamilyId:      c.family,
			InstallmentId: pgconv.UUIDString(inst.ID),
			Name:          inst.Name,
			Total:         money(inst.TotalMinor, inst.CurrencyCode),
			OccurredAt:    h.timestamp(),
		})
	}
}
