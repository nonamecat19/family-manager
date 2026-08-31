package handler

import (
	"context"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

// Subjects this service publishes.
//
// CONVENTION DEBT, deliberate and reported: every other subject in the repo is a constant in
// libs/go/events/events.go, and these belong there too. libs/go is outside this task's file
// assignment (one writer per file), so they are declared here — as constants rather than
// string literals at the call sites, which is the half of the rule this file can keep — and
// the move is called out in the task's return value.
const (
	subjectTransactionCreated events.Subject = "finance.transaction.created"
	subjectTransactionUpdated events.Subject = "finance.transaction.updated"
	subjectTransactionDeleted events.Subject = "finance.transaction.deleted"
	subjectTransferCreated    events.Subject = "finance.transfer.created"
	subjectAccountCreated     events.Subject = "finance.account.created"
	subjectAccountUpdated     events.Subject = "finance.account.updated"
	subjectBudgetCreated      events.Subject = "finance.budget.created"
	subjectBudgetUpdated      events.Subject = "finance.budget.updated"
	subjectBudgetExceeded     events.Subject = "finance.budget.exceeded"
	subjectBudgetRecovered    events.Subject = "finance.budget.recovered"
	subjectTemplateUsed       events.Subject = "finance.template.used"
	subjectRecurringPosted    events.Subject = "finance.recurring.posted"
)

// Stored enum values. The CHECK constraints in 000001_init.up.sql are the other half of each
// of these sets; a value that is not here cannot reach the database.
const (
	kindExpense  = "expense"
	kindIncome   = "income"
	kindTransfer = "transfer"

	visibilityShared  = "shared"
	visibilityPrivate = "private"

	accountKindSavings = "savings"

	targetGroup    = "group"
	targetCategory = "category"

	budgetPeriodWeek  = "week"
	budgetPeriodMonth = "month"
	budgetPeriodYear  = "year"
)

// EventBus is the slice of libs/go/events this service uses. Narrow on purpose: tests pass a
// recorder instead of standing up NATS.
type EventBus interface {
	Publish(ctx context.Context, subject events.Subject, msg proto.Message) error
}

// noopBus lets the service run (and tests pass) with no broker attached.
type noopBus struct{}

func (noopBus) Publish(context.Context, events.Subject, proto.Message) error { return nil }

// publish is fire-and-forget by design: a transaction that was written must not be reported
// as failed because the broker hiccuped. The failure is logged, not returned.
func (h *Handler) publish(ctx context.Context, subject events.Subject, msg proto.Message) {
	if err := h.bus.Publish(ctx, subject, msg); err != nil {
		h.log.WarnContext(ctx, "publish failed", slog.String("subject", string(subject)),
			slog.String("error", err.Error()))
	}
}

func (h *Handler) timestamp() *timestamppb.Timestamp { return timestamppb.New(h.now()) }

func trimmed(s string) string { return strings.TrimSpace(s) }

/* --------------------------------------------------------------------- money */

// money is the one place a stored (minor, code) pair becomes the wire type. Floating point
// never enters: the minor unit is an int64 from the column to the app's number.
func money(minor int64, code string) *financev1.Money {
	return &financev1.Money{AmountMinor: minor, CurrencyCode: code}
}

func moneyMinor(m *financev1.Money) int64 { return m.GetAmountMinor() }

/* --------------------------------------------------------------------- enums */

func accountKindToProto(s string) financev1.AccountKind {
	switch s {
	case "cash":
		return financev1.AccountKind_ACCOUNT_KIND_CASH
	case "card":
		return financev1.AccountKind_ACCOUNT_KIND_CARD
	case "bank":
		return financev1.AccountKind_ACCOUNT_KIND_BANK
	case accountKindSavings:
		return financev1.AccountKind_ACCOUNT_KIND_SAVINGS
	case "crypto":
		return financev1.AccountKind_ACCOUNT_KIND_CRYPTO
	case "debt":
		return financev1.AccountKind_ACCOUNT_KIND_DEBT
	default:
		return financev1.AccountKind_ACCOUNT_KIND_UNSPECIFIED
	}
}

// accountKindFromProto defaults an unspecified kind to cash rather than rejecting it: the
// picker starts on a kind, and "no kind" from an older client is a cash account, not an error.
func accountKindFromProto(k financev1.AccountKind) string {
	switch k {
	case financev1.AccountKind_ACCOUNT_KIND_CARD:
		return "card"
	case financev1.AccountKind_ACCOUNT_KIND_BANK:
		return "bank"
	case financev1.AccountKind_ACCOUNT_KIND_SAVINGS:
		return accountKindSavings
	case financev1.AccountKind_ACCOUNT_KIND_CRYPTO:
		return "crypto"
	case financev1.AccountKind_ACCOUNT_KIND_DEBT:
		return "debt"
	default:
		return "cash"
	}
}

func visibilityToProto(s string) financev1.AccountVisibility {
	if s == visibilityPrivate {
		return financev1.AccountVisibility_ACCOUNT_VISIBILITY_PRIVATE
	}
	return financev1.AccountVisibility_ACCOUNT_VISIBILITY_SHARED
}

// visibilityFromProto defaults to shared. An account whose visibility a client forgot to set
// must not silently become private: private is the state with a security consequence, so it
// has to be asked for.
func visibilityFromProto(v financev1.AccountVisibility) string {
	if v == financev1.AccountVisibility_ACCOUNT_VISIBILITY_PRIVATE {
		return visibilityPrivate
	}
	return visibilityShared
}

func txTypeToProto(s string) financev1.TransactionType {
	switch s {
	case kindExpense:
		return financev1.TransactionType_TRANSACTION_TYPE_EXPENSE
	case kindIncome:
		return financev1.TransactionType_TRANSACTION_TYPE_INCOME
	case kindTransfer:
		return financev1.TransactionType_TRANSACTION_TYPE_TRANSFER
	default:
		return financev1.TransactionType_TRANSACTION_TYPE_UNSPECIFIED
	}
}

// txTypeFromProto defaults to expense, which is what the add sheet opens on.
func txTypeFromProto(t financev1.TransactionType) string {
	switch t {
	case financev1.TransactionType_TRANSACTION_TYPE_INCOME:
		return kindIncome
	case financev1.TransactionType_TRANSACTION_TYPE_TRANSFER:
		return kindTransfer
	default:
		return kindExpense
	}
}

func txKindToProto(s string) financev1.TransactionKind {
	switch s {
	case kindIncome:
		return financev1.TransactionKind_TRANSACTION_KIND_INCOME
	case kindExpense:
		return financev1.TransactionKind_TRANSACTION_KIND_EXPENSE
	default:
		return financev1.TransactionKind_TRANSACTION_KIND_UNSPECIFIED
	}
}

// taxonomyKindFromProto is the expense/income discriminator on a group or category. Unlike a
// filter, a stored kind must be one or the other, so UNSPECIFIED becomes expense.
func taxonomyKindFromProto(k financev1.TransactionKind) string {
	if k == financev1.TransactionKind_TRANSACTION_KIND_INCOME {
		return kindIncome
	}
	return kindExpense
}

// kindFilter is the nullable filter form: UNSPECIFIED means "both", so it maps to a NULL
// sentinel rather than to a value. This is the ЗАГАЛЬНЕ tab.
func kindFilter(k financev1.TransactionKind) *string {
	switch k {
	case financev1.TransactionKind_TRANSACTION_KIND_EXPENSE:
		v := kindExpense
		return &v
	case financev1.TransactionKind_TRANSACTION_KIND_INCOME:
		v := kindIncome
		return &v
	default:
		return nil
	}
}

func budgetPeriodToProto(s string) financev1.BudgetPeriod {
	switch s {
	case budgetPeriodWeek:
		return financev1.BudgetPeriod_BUDGET_PERIOD_WEEK
	case budgetPeriodYear:
		return financev1.BudgetPeriod_BUDGET_PERIOD_YEAR
	default:
		return financev1.BudgetPeriod_BUDGET_PERIOD_MONTH
	}
}

// budgetPeriodFromProto defaults to month: every budget the design draws is monthly.
func budgetPeriodFromProto(p financev1.BudgetPeriod) string {
	switch p {
	case financev1.BudgetPeriod_BUDGET_PERIOD_WEEK:
		return budgetPeriodWeek
	case financev1.BudgetPeriod_BUDGET_PERIOD_YEAR:
		return budgetPeriodYear
	default:
		return budgetPeriodMonth
	}
}

func budgetTargetToProto(s string) financev1.BudgetTargetKind {
	if s == targetCategory {
		return financev1.BudgetTargetKind_BUDGET_TARGET_KIND_CATEGORY
	}
	return financev1.BudgetTargetKind_BUDGET_TARGET_KIND_GROUP
}

// budgetTargetFilter maps the list filter to the nullable column sentinel; UNSPECIFIED is
// every budget, which is what the household counter needs.
func budgetTargetFilter(f financev1.BudgetTargetFilter) *string {
	switch f {
	case financev1.BudgetTargetFilter_BUDGET_TARGET_FILTER_GROUP:
		v := targetGroup
		return &v
	case financev1.BudgetTargetFilter_BUDGET_TARGET_FILTER_CATEGORY:
		v := targetCategory
		return &v
	default:
		return nil
	}
}

func memberRoleToProto(s string) financev1.MemberRole {
	if s == "owner" {
		return financev1.MemberRole_MEMBER_ROLE_OWNER
	}
	return financev1.MemberRole_MEMBER_ROLE_MEMBER
}

func memberStatusToProto(s string) financev1.MemberStatus {
	if s == "pending" {
		return financev1.MemberStatus_MEMBER_STATUS_PENDING
	}
	return financev1.MemberStatus_MEMBER_STATUS_ACTIVE
}

var weekdayNames = []string{
	"", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday",
}

func weekdayToProto(s string) financev1.Weekday {
	for i, name := range weekdayNames {
		if i > 0 && name == s {
			return financev1.Weekday(i)
		}
	}
	return financev1.Weekday_WEEKDAY_UNSPECIFIED
}

// weekdayFromProto returns "" for UNSPECIFIED so a caller can tell "leave it alone" from a
// chosen day; the recurring-payment column stores "" for "no weekday anchor".
func weekdayFromProto(w financev1.Weekday) string {
	i := int(w)
	if i <= 0 || i >= len(weekdayNames) {
		return ""
	}
	return weekdayNames[i]
}

var recurrenceUnitNames = []string{"", "day", "week", "month", "year"}

func recurrenceUnitToProto(s string) financev1.RecurrenceUnit {
	for i, name := range recurrenceUnitNames {
		if i > 0 && name == s {
			return financev1.RecurrenceUnit(i)
		}
	}
	return financev1.RecurrenceUnit_RECURRENCE_UNIT_UNSPECIFIED
}

func recurrenceUnitFromProto(u financev1.RecurrenceUnit) string {
	i := int(u)
	if i <= 0 || i >= len(recurrenceUnitNames) {
		return ""
	}
	return recurrenceUnitNames[i]
}

func reminderKindToProto(s string) financev1.ReminderKind {
	switch s {
	case "budget_exceeded":
		return financev1.ReminderKind_REMINDER_KIND_BUDGET_EXCEEDED
	case "recurring_due":
		return financev1.ReminderKind_REMINDER_KIND_RECURRING_DUE
	default:
		return financev1.ReminderKind_REMINDER_KIND_CUSTOM
	}
}

func reminderKindFromProto(k financev1.ReminderKind) string {
	switch k {
	case financev1.ReminderKind_REMINDER_KIND_BUDGET_EXCEEDED:
		return "budget_exceeded"
	case financev1.ReminderKind_REMINDER_KIND_RECURRING_DUE:
		return "recurring_due"
	default:
		return "custom"
	}
}

var widgetTypeNames = []string{
	"", "quick_add", "month", "category", "budgets_and_family", "recent_transactions", "accounts",
}

func widgetTypeToProto(s string) financev1.WidgetType {
	for i, name := range widgetTypeNames {
		if i > 0 && name == s {
			return financev1.WidgetType(i)
		}
	}
	return financev1.WidgetType_WIDGET_TYPE_UNSPECIFIED
}

func widgetTypeFromProto(t financev1.WidgetType) string {
	i := int(t)
	if i <= 0 || i >= len(widgetTypeNames) {
		return ""
	}
	return widgetTypeNames[i]
}

var widgetSizeNames = []string{"", "4x2", "2x2", "2x1", "4x3", "4x1"}

func widgetSizeToProto(s string) financev1.WidgetSize {
	for i, name := range widgetSizeNames {
		if i > 0 && name == s {
			return financev1.WidgetSize(i)
		}
	}
	return financev1.WidgetSize_WIDGET_SIZE_UNSPECIFIED
}

// widgetSizeFromProto defaults to the 4×2 cell, the size the gallery opens on.
func widgetSizeFromProto(s financev1.WidgetSize) string {
	i := int(s)
	if i <= 0 || i >= len(widgetSizeNames) {
		return "4x2"
	}
	return widgetSizeNames[i]
}

var scopeKindNames = []string{"", "family", "member", "account"}

func scopeKindToProto(s string) financev1.ScopeKind {
	for i, name := range scopeKindNames {
		if i > 0 && name == s {
			return financev1.ScopeKind(i)
		}
	}
	return financev1.ScopeKind_SCOPE_KIND_FAMILY
}

func scopeKindFromProto(k financev1.ScopeKind) string {
	i := int(k)
	if i <= 0 || i >= len(scopeKindNames) {
		return "family"
	}
	return scopeKindNames[i]
}

/* --------------------------------------------------------------- to-proto */

func toProtoSettings(s db.FinanceSetting) *financev1.HouseholdFinanceSettings {
	return &financev1.HouseholdFinanceSettings{
		FamilyId:                      pgconv.UUIDString(s.FamilyID),
		BaseCurrencyCode:              s.BaseCurrencyCode,
		Timezone:                      s.Timezone,
		WeekStartsOn:                  weekdayToProto(s.WeekStartsOn),
		OverspendNotificationsEnabled: s.OverspendNotificationsEnabled,
		PinLockEnabled:                s.PinLockEnabled,
		CreatedAt:                     pgconv.Timestamp(s.CreatedAt),
		UpdatedAt:                     pgconv.Timestamp(s.UpdatedAt),
	}
}

func toProtoMember(m db.FinanceMember) *financev1.Member {
	return &financev1.Member{
		UserId:          pgconv.UUIDString(m.UserID),
		FamilyId:        pgconv.UUIDString(m.FamilyID),
		DisplayName:     m.DisplayName,
		Initial:         m.Initial,
		AvatarColorStep: m.AvatarColorStep,
		Role:            memberRoleToProto(m.Role),
		Status:          memberStatusToProto(m.Status),
		Email:           m.Email,
		JoinedAt:        pgconv.Timestamp(m.JoinedAt),
	}
}

// accountView is the shape every account read produces: the row plus its derived balance.
// ListVisibleAccounts and GetVisibleAccount generate two structurally identical row types, so
// this is where they converge instead of duplicating the converter.
type accountView struct {
	row     db.Account
	balance int64
}

func viewFromList(r db.ListVisibleAccountsRow) accountView {
	return accountView{row: db.Account{
		ID: r.ID, FamilyID: r.FamilyID, Name: r.Name, Kind: r.Kind, Visibility: r.Visibility,
		OwnerMemberID: r.OwnerMemberID, CurrencyCode: r.CurrencyCode,
		OpeningBalanceMinor: r.OpeningBalanceMinor, Icon: r.Icon, ColorStep: r.ColorStep,
		ExcludedFromFamilyTotal: r.ExcludedFromFamilyTotal, Archived: r.Archived,
		SortOrder: r.SortOrder, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, balance: r.BalanceMinor}
}

func viewFromGet(r db.GetVisibleAccountRow) accountView {
	return accountView{row: db.Account{
		ID: r.ID, FamilyID: r.FamilyID, Name: r.Name, Kind: r.Kind, Visibility: r.Visibility,
		OwnerMemberID: r.OwnerMemberID, CurrencyCode: r.CurrencyCode,
		OpeningBalanceMinor: r.OpeningBalanceMinor, Icon: r.Icon, ColorStep: r.ColorStep,
		ExcludedFromFamilyTotal: r.ExcludedFromFamilyTotal, Archived: r.Archived,
		SortOrder: r.SortOrder, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, balance: r.BalanceMinor}
}

// viewFromWrite wraps a row that came back from an INSERT/UPDATE, where no balance was
// computed. The opening balance is the honest answer for a just-created account and the app
// refetches the list after an edit.
func viewFromWrite(a db.Account) accountView {
	return accountView{row: a, balance: a.OpeningBalanceMinor}
}

func toProtoAccount(v accountView) *financev1.Account {
	a := v.row
	return &financev1.Account{
		Id:                      pgconv.UUIDString(a.ID),
		FamilyId:                pgconv.UUIDString(a.FamilyID),
		Name:                    a.Name,
		Kind:                    accountKindToProto(a.Kind),
		Visibility:              visibilityToProto(a.Visibility),
		OwnerMemberId:           pgconv.UUIDString(a.OwnerMemberID),
		CurrencyCode:            a.CurrencyCode,
		OpeningBalance:          money(a.OpeningBalanceMinor, a.CurrencyCode),
		Balance:                 money(v.balance, a.CurrencyCode),
		Icon:                    a.Icon,
		ColorStep:               a.ColorStep,
		ExcludedFromFamilyTotal: a.ExcludedFromFamilyTotal,
		Archived:                a.Archived,
		SortOrder:               a.SortOrder,
		CreatedAt:               pgconv.Timestamp(a.CreatedAt),
		UpdatedAt:               pgconv.Timestamp(a.UpdatedAt),
	}
}

func toProtoGroup(g db.CategoryGroup) *financev1.CategoryGroup {
	return &financev1.CategoryGroup{
		Id:        pgconv.UUIDString(g.ID),
		FamilyId:  pgconv.UUIDString(g.FamilyID),
		Name:      g.Name,
		Kind:      txKindToProto(g.Kind),
		Icon:      g.Icon,
		ColorStep: g.ColorStep,
		SortOrder: g.SortOrder,
		Archived:  g.Archived,
		CreatedAt: pgconv.Timestamp(g.CreatedAt),
		UpdatedAt: pgconv.Timestamp(g.UpdatedAt),
	}
}

func toProtoCategory(c db.Category) *financev1.Category {
	return &financev1.Category{
		Id:        pgconv.UUIDString(c.ID),
		FamilyId:  pgconv.UUIDString(c.FamilyID),
		GroupId:   pgconv.UUIDString(c.GroupID),
		Name:      c.Name,
		Kind:      txKindToProto(c.Kind),
		Icon:      c.Icon,
		SortOrder: c.SortOrder,
		Archived:  c.Archived,
		CreatedAt: pgconv.Timestamp(c.CreatedAt),
		UpdatedAt: pgconv.Timestamp(c.UpdatedAt),
	}
}

func toProtoBudget(b db.Budget) *financev1.Budget {
	return &financev1.Budget{
		Id:             pgconv.UUIDString(b.ID),
		FamilyId:       pgconv.UUIDString(b.FamilyID),
		TargetKind:     budgetTargetToProto(b.TargetKind),
		GroupId:        pgconv.UUIDString(b.GroupID),
		CategoryId:     pgconv.UUIDString(b.CategoryID),
		Limit:          money(b.LimitMinor, b.CurrencyCode),
		Period:         budgetPeriodToProto(b.Period),
		StartOn:        pgconv.DateString(b.StartOn),
		MemberId:       pgconv.UUIDString(b.MemberID),
		NotifyOnExceed: b.NotifyOnExceed,
		Archived:       b.Archived,
		SortOrder:      b.SortOrder,
		CreatedAt:      pgconv.Timestamp(b.CreatedAt),
		UpdatedAt:      pgconv.Timestamp(b.UpdatedAt),
	}
}

// transactionView is a transaction plus the group its category belongs to. The feed reads it
// from a join; a write path fills GroupID separately, because the row an INSERT returns has
// no join.
type transactionView struct {
	row     db.Transaction
	groupID pgtype.UUID
}

func viewFromTxRow(r db.ListVisibleTransactionsRow) transactionView {
	return transactionView{row: db.Transaction{
		ID: r.ID, FamilyID: r.FamilyID, Type: r.Type, AccountID: r.AccountID,
		CounterAccountID: r.CounterAccountID, CategoryID: r.CategoryID,
		AmountMinor: r.AmountMinor, CurrencyCode: r.CurrencyCode,
		ReceivedAmountMinor: r.ReceivedAmountMinor, ReceivedCurrencyCode: r.ReceivedCurrencyCode,
		Note: r.Note, Merchant: r.Merchant, OccurredOn: r.OccurredOn, MemberID: r.MemberID,
		CreatedByUserID: r.CreatedByUserID, TemplateID: r.TemplateID, RecurringID: r.RecurringID,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, groupID: r.GroupID}
}

func viewFromGetTx(r db.GetVisibleTransactionRow) transactionView {
	return transactionView{row: db.Transaction{
		ID: r.ID, FamilyID: r.FamilyID, Type: r.Type, AccountID: r.AccountID,
		CounterAccountID: r.CounterAccountID, CategoryID: r.CategoryID,
		AmountMinor: r.AmountMinor, CurrencyCode: r.CurrencyCode,
		ReceivedAmountMinor: r.ReceivedAmountMinor, ReceivedCurrencyCode: r.ReceivedCurrencyCode,
		Note: r.Note, Merchant: r.Merchant, OccurredOn: r.OccurredOn, MemberID: r.MemberID,
		CreatedByUserID: r.CreatedByUserID, TemplateID: r.TemplateID, RecurringID: r.RecurringID,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, groupID: r.GroupID}
}

func toProtoTransaction(v transactionView) *financev1.Transaction {
	t := v.row
	out := &financev1.Transaction{
		Id:               pgconv.UUIDString(t.ID),
		FamilyId:         pgconv.UUIDString(t.FamilyID),
		Type:             txTypeToProto(t.Type),
		AccountId:        pgconv.UUIDString(t.AccountID),
		CounterAccountId: pgconv.UUIDString(t.CounterAccountID),
		CategoryId:       pgconv.UUIDString(t.CategoryID),
		GroupId:          pgconv.UUIDString(v.groupID),
		Amount:           money(t.AmountMinor, t.CurrencyCode),
		Note:             t.Note,
		Merchant:         t.Merchant,
		OccurredOn:       pgconv.DateString(t.OccurredOn),
		MemberId:         pgconv.UUIDString(t.MemberID),
		CreatedByUserId:  pgconv.UUIDString(t.CreatedByUserID),
		TemplateId:       pgconv.UUIDString(t.TemplateID),
		RecurringId:      pgconv.UUIDString(t.RecurringID),
		CreatedAt:        pgconv.Timestamp(t.CreatedAt),
		UpdatedAt:        pgconv.Timestamp(t.UpdatedAt),
	}
	// received_amount exists only on a cross-currency transfer; a zeroed Money on every other
	// row would make the app draw a second amount of 0.
	if t.ReceivedAmountMinor != nil {
		out.ReceivedAmount = money(*t.ReceivedAmountMinor, t.ReceivedCurrencyCode)
	}
	return out
}

func toProtoTemplate(t db.QuickTemplate) *financev1.QuickTemplate {
	return &financev1.QuickTemplate{
		Id:          pgconv.UUIDString(t.ID),
		FamilyId:    pgconv.UUIDString(t.FamilyID),
		OwnerUserId: pgconv.UUIDString(t.OwnerUserID),
		Label:       t.Label,
		Icon:        t.Icon,
		Amount:      money(t.AmountMinor, t.CurrencyCode),
		Type:        txTypeToProto(t.Type),
		CategoryId:  pgconv.UUIDString(t.CategoryID),
		AccountId:   pgconv.UUIDString(t.AccountID),
		MemberId:    pgconv.UUIDString(t.MemberID),
		SortOrder:   t.SortOrder,
		UsageCount:  t.UsageCount,
		LastUsedAt:  pgconv.Timestamp(t.LastUsedAt),
		CreatedAt:   pgconv.Timestamp(t.CreatedAt),
		UpdatedAt:   pgconv.Timestamp(t.UpdatedAt),
	}
}

func toProtoRecurring(r db.RecurringPayment) *financev1.RecurringPayment {
	return &financev1.RecurringPayment{
		Id:         pgconv.UUIDString(r.ID),
		FamilyId:   pgconv.UUIDString(r.FamilyID),
		Name:       r.Name,
		Amount:     money(r.AmountMinor, r.CurrencyCode),
		Type:       txTypeToProto(r.Type),
		CategoryId: pgconv.UUIDString(r.CategoryID),
		AccountId:  pgconv.UUIDString(r.AccountID),
		MemberId:   pgconv.UUIDString(r.MemberID),
		Cadence: &financev1.Cadence{
			Interval:   r.IntervalCount,
			Unit:       recurrenceUnitToProto(r.IntervalUnit),
			DayOfMonth: r.DayOfMonth,
			DayOfWeek:  weekdayToProto(r.DayOfWeek),
		},
		NextDueOn:    pgconv.DateString(r.NextDueOn),
		EndOn:        pgconv.DateString(r.EndOn),
		AutoPost:     r.AutoPost,
		Active:       r.Active,
		LastPostedOn: pgconv.DateString(r.LastPostedOn),
		CreatedAt:    pgconv.Timestamp(r.CreatedAt),
		UpdatedAt:    pgconv.Timestamp(r.UpdatedAt),
	}
}

func toProtoReminder(r db.Reminder) *financev1.Reminder {
	return &financev1.Reminder{
		Id:       pgconv.UUIDString(r.ID),
		FamilyId: pgconv.UUIDString(r.FamilyID),
		UserId:   pgconv.UUIDString(r.UserID),
		Kind:     reminderKindToProto(r.Kind),
		Title:    r.Title,
		DueAt:    pgconv.Timestamp(r.DueAt),
		Repeat: &financev1.Cadence{
			Interval: r.RepeatInterval,
			Unit:     recurrenceUnitToProto(r.RepeatUnit),
		},
		Enabled:   r.Enabled,
		CreatedAt: pgconv.Timestamp(r.CreatedAt),
		UpdatedAt: pgconv.Timestamp(r.UpdatedAt),
	}
}

func toProtoWidget(w db.WidgetInstance) *financev1.WidgetInstance {
	ids := make([]string, 0, len(w.TargetAccountIds))
	for _, id := range w.TargetAccountIds {
		ids = append(ids, pgconv.UUIDString(id))
	}
	return &financev1.WidgetInstance{
		Id:       pgconv.UUIDString(w.ID),
		FamilyId: pgconv.UUIDString(w.FamilyID),
		UserId:   pgconv.UUIDString(w.UserID),
		Type:     widgetTypeToProto(w.Type),
		Size:     widgetSizeToProto(w.Size),
		Scope: &financev1.Scope{
			Kind:      scopeKindToProto(w.ScopeKind),
			MemberId:  pgconv.UUIDString(w.ScopeMemberID),
			AccountId: pgconv.UUIDString(w.ScopeAccountID),
		},
		TargetRef:        pgconv.UUIDString(w.TargetRef),
		TargetAccountIds: ids,
		SortOrder:        w.SortOrder,
		CreatedAt:        pgconv.Timestamp(w.CreatedAt),
		UpdatedAt:        pgconv.Timestamp(w.UpdatedAt),
	}
}
