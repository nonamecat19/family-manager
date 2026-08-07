package handler

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/nnc/family-manager/libs/go/events"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
)

// Budget tests pin "today" so a window is deterministic rather than depending on the day the
// suite runs.
func newBudgetFixture(t *testing.T, today string) *fixture {
	t.Helper()
	f := newFixture(t)
	f.h.now = func() time.Time { return day(today) }
	return f
}

func (f *fixture) budget(
	t *testing.T, name string, cat *financev1.Category, limit int64, period financev1.BudgetPeriod,
	startOn string,
) *financev1.Budget {
	t.Helper()
	categoryID := ""
	if cat != nil {
		categoryID = cat.GetId()
	}
	res, err := f.h.CreateBudget(asA(), connect.NewRequest(&financev1.CreateBudgetRequest{
		Name:       name,
		CategoryId: categoryID,
		Limit:      money64(limit, "EUR"),
		Period:     period,
		StartOn:    startOn,
	}))
	if err != nil {
		t.Fatalf("CreateBudget(%s): %v", name, err)
	}
	return res.Msg.GetBudget()
}

func (f *fixture) statuses(t *testing.T, asOf string) []*financev1.BudgetStatus {
	t.Helper()
	res, err := f.h.ListBudgets(asA(), connect.NewRequest(&financev1.ListBudgetsRequest{
		AsOf: asOf,
	}))
	if err != nil {
		t.Fatalf("ListBudgets: %v", err)
	}
	return res.Msg.GetBudgets()
}

func TestBudgetProgressCountsOnlySpendingInsideTheWindow(t *testing.T) {
	f := newBudgetFixture(t, "2026-03-15")
	acc := f.account(t, asA(), "Wallet", "EUR", 1000000)
	food := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	f.budget(t, "Food", food, 50000, financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, "2026-03-01")

	f.spend(t, asA(), acc, food, 20000, "2026-03-10", "in window")
	f.spend(t, asA(), acc, food, 5000, "2026-02-28", "previous window")
	f.spend(t, asA(), acc, food, 7000, "2026-04-02", "next window")

	statuses := f.statuses(t, "2026-03-15")
	if len(statuses) != 1 {
		t.Fatalf("got %d budgets, want 1", len(statuses))
	}
	s := statuses[0]

	if got := s.GetSpent().GetAmountMinor(); got != 20000 {
		t.Errorf("spent = %d, want 20000 (only the in-window expense)", got)
	}
	if got := s.GetRemaining().GetAmountMinor(); got != 30000 {
		t.Errorf("remaining = %d, want 30000", got)
	}
	if s.GetShare() != 0.4 {
		t.Errorf("share = %v, want 0.4", s.GetShare())
	}
	if s.GetExceeded() {
		t.Error("a budget at 40%% reported itself exceeded")
	}
	if s.GetPeriod().GetFrom() != "2026-03-01" || s.GetPeriod().GetTo() != "2026-03-31" {
		t.Errorf("window = %s..%s", s.GetPeriod().GetFrom(), s.GetPeriod().GetTo())
	}
	if s.GetDaysRemaining() != 17 {
		t.Errorf("days_remaining = %d, want 17", s.GetDaysRemaining())
	}
}

func TestIncomeAndTransfersDoNotConsumeABudget(t *testing.T) {
	f := newBudgetFixture(t, "2026-03-15")
	wallet := f.account(t, asA(), "Wallet", "EUR", 1000000)
	savings := f.account(t, asA(), "Savings", "EUR", 0)
	salary := f.category(t, asA(), "Salary", financev1.TransactionType_TRANSACTION_TYPE_INCOME)
	f.budget(t, "Everything", nil, 50000, financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, "2026-03-01")

	if _, err := f.h.CreateTransaction(asA(),
		connect.NewRequest(&financev1.CreateTransactionRequest{
			AccountId: wallet.GetId(), CategoryId: salary.GetId(),
			Type:       financev1.TransactionType_TRANSACTION_TYPE_INCOME,
			Amount:     money64(300000, "EUR"),
			OccurredOn: "2026-03-05",
		})); err != nil {
		t.Fatalf("income: %v", err)
	}
	if _, err := f.h.CreateTransaction(asA(),
		connect.NewRequest(&financev1.CreateTransactionRequest{
			AccountId: wallet.GetId(), CounterAccountId: savings.GetId(),
			Type:       financev1.TransactionType_TRANSACTION_TYPE_TRANSFER,
			Amount:     money64(400000, "EUR"),
			OccurredOn: "2026-03-06",
		})); err != nil {
		t.Fatalf("transfer: %v", err)
	}

	if got := f.statuses(t, "2026-03-15")[0].GetSpent().GetAmountMinor(); got != 0 {
		t.Errorf("spent = %d, want 0: neither income nor a transfer is spending", got)
	}
}

// A budget with no category covers every expense; a per-category one counts only its own.
func TestTotalAndPerCategoryBudgetsCoexist(t *testing.T) {
	f := newBudgetFixture(t, "2026-03-15")
	acc := f.account(t, asA(), "Wallet", "EUR", 1000000)
	food := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	travel := f.category(t, asA(), "Travel", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)

	f.budget(t, "All spending", nil, 100000, financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, "2026-03-01")
	f.budget(t, "Food", food, 30000, financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, "2026-03-01")

	f.spend(t, asA(), acc, food, 20000, "2026-03-05", "groceries")
	f.spend(t, asA(), acc, travel, 15000, "2026-03-06", "train")

	byName := map[string]*financev1.BudgetStatus{}
	for _, s := range f.statuses(t, "2026-03-15") {
		byName[s.GetBudget().GetName()] = s
	}

	if got := byName["All spending"].GetSpent().GetAmountMinor(); got != 35000 {
		t.Errorf("total budget spent = %d, want 35000", got)
	}
	if got := byName["Food"].GetSpent().GetAmountMinor(); got != 20000 {
		t.Errorf("food budget spent = %d, want 20000", got)
	}
}

func TestOverspendReportsNegativeRemaining(t *testing.T) {
	f := newBudgetFixture(t, "2026-03-15")
	acc := f.account(t, asA(), "Wallet", "EUR", 1000000)
	food := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	f.budget(t, "Food", food, 10000, financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, "2026-03-01")

	f.spend(t, asA(), acc, food, 12500, "2026-03-05", "big shop")

	s := f.statuses(t, "2026-03-15")[0]
	if !s.GetExceeded() {
		t.Error("exceeded = false at 125%")
	}
	// Clamping to zero would hide the overspend, which is the number the user needs.
	if got := s.GetRemaining().GetAmountMinor(); got != -2500 {
		t.Errorf("remaining = %d, want -2500", got)
	}
	if s.GetShare() != 1.25 {
		t.Errorf("share = %v, want 1.25", s.GetShare())
	}
}

func TestCrossingTheLimitPublishesOnceNotOnEveryPurchase(t *testing.T) {
	f := newBudgetFixture(t, "2026-03-15")
	acc := f.account(t, asA(), "Wallet", "EUR", 1000000)
	food := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	f.budget(t, "Food", food, 10000, financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, "2026-03-01")

	f.spend(t, asA(), acc, food, 6000, "2026-03-05", "under")
	if countSubject(f.bus, events.SubjectFinanceBudgetExceeded) != 0 {
		t.Fatal("published while still under the limit")
	}

	f.spend(t, asA(), acc, food, 6000, "2026-03-06", "crosses")
	if got := countSubject(f.bus, events.SubjectFinanceBudgetExceeded); got != 1 {
		t.Fatalf("crossings published = %d, want 1", got)
	}

	// Already over: further spending must not re-announce, or an overspent month becomes a
	// notification storm.
	f.spend(t, asA(), acc, food, 3000, "2026-03-07", "still over")
	if got := countSubject(f.bus, events.SubjectFinanceBudgetExceeded); got != 1 {
		t.Errorf("crossings published = %d after further spending, want 1", got)
	}
}

func TestBackdatedExpenseDoesNotAlarmTheCurrentWindow(t *testing.T) {
	f := newBudgetFixture(t, "2026-03-15")
	acc := f.account(t, asA(), "Wallet", "EUR", 1000000)
	food := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	f.budget(t, "Food", food, 10000, financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, "2026-03-01")

	// January is a closed window; blowing it now is history, not news.
	f.spend(t, asA(), acc, food, 99999, "2026-01-10", "old receipt")

	if got := countSubject(f.bus, events.SubjectFinanceBudgetExceeded); got != 0 {
		t.Errorf("published %d crossings for a closed window, want 0", got)
	}
	if got := f.statuses(t, "2026-03-15")[0].GetSpent().GetAmountMinor(); got != 0 {
		t.Errorf("March spend = %d, want 0", got)
	}
}

func TestBudgetsAreScopedToTheCallersFamily(t *testing.T) {
	f := newBudgetFixture(t, "2026-03-15")
	food := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	created := f.budget(t, "Food", food, 10000,
		financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, "2026-03-01")

	res, err := f.h.ListBudgets(asB(), connect.NewRequest(&financev1.ListBudgetsRequest{}))
	if err != nil {
		t.Fatalf("ListBudgets: %v", err)
	}
	if len(res.Msg.GetBudgets()) != 0 {
		t.Error("family B can see family A's budgets")
	}

	_, err = f.h.GetBudget(asB(), connect.NewRequest(&financev1.GetBudgetRequest{
		Id: created.GetId(),
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("cross-family GetBudget = %v, want not_found", connect.CodeOf(err))
	}
}

func TestOnlyOneBudgetPerCategoryPerPeriod(t *testing.T) {
	f := newBudgetFixture(t, "2026-03-15")
	food := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	f.budget(t, "Food", food, 10000, financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, "2026-03-01")

	_, err := f.h.CreateBudget(asA(), connect.NewRequest(&financev1.CreateBudgetRequest{
		Name: "Food again", CategoryId: food.GetId(), Limit: money64(20000, "EUR"),
		Period: financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, StartOn: "2026-03-01",
	}))
	if connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("code = %v, want already_exists", connect.CodeOf(err))
	}

	// A different period is a different budget and is allowed.
	f.budget(t, "Food weekly", food, 3000,
		financev1.BudgetPeriod_BUDGET_PERIOD_WEEK, "2026-03-02")
}

func TestOnlyOneTotalBudgetPerPeriod(t *testing.T) {
	f := newBudgetFixture(t, "2026-03-15")
	f.budget(t, "All", nil, 100000, financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, "2026-03-01")

	// NULL never equals NULL in a unique index, which is why the schema needs a second
	// partial index — without it this would succeed.
	_, err := f.h.CreateBudget(asA(), connect.NewRequest(&financev1.CreateBudgetRequest{
		Name: "All again", Limit: money64(1, "EUR"),
		Period: financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, StartOn: "2026-03-01",
	}))
	if connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("code = %v, want already_exists", connect.CodeOf(err))
	}
}

func TestIncomeCategoriesCannotBeBudgeted(t *testing.T) {
	f := newBudgetFixture(t, "2026-03-15")
	salary := f.category(t, asA(), "Salary", financev1.TransactionType_TRANSACTION_TYPE_INCOME)

	_, err := f.h.CreateBudget(asA(), connect.NewRequest(&financev1.CreateBudgetRequest{
		Name: "Salary", CategoryId: salary.GetId(), Limit: money64(1000, "EUR"),
		Period: financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, StartOn: "2026-03-01",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestCreateBudgetValidatesInput(t *testing.T) {
	f := newBudgetFixture(t, "2026-03-15")

	cases := []struct {
		name string
		req  *financev1.CreateBudgetRequest
	}{
		{"blank name", &financev1.CreateBudgetRequest{
			Name: "  ", Limit: money64(100, "EUR"),
			Period: financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, StartOn: "2026-03-01"}},
		{"unspecified period", &financev1.CreateBudgetRequest{
			Name: "X", Limit: money64(100, "EUR"), StartOn: "2026-03-01"}},
		{"zero limit", &financev1.CreateBudgetRequest{
			Name: "X", Limit: money64(0, "EUR"),
			Period: financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, StartOn: "2026-03-01"}},
		{"negative limit", &financev1.CreateBudgetRequest{
			Name: "X", Limit: money64(-5, "EUR"),
			Period: financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, StartOn: "2026-03-01"}},
		{"bad start_on", &financev1.CreateBudgetRequest{
			Name: "X", Limit: money64(100, "EUR"),
			Period: financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, StartOn: "01/03/2026"}},
	}

	for _, c := range cases {
		_, err := f.h.CreateBudget(asA(), connect.NewRequest(c.req))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("%s: code = %v, want invalid_argument", c.name, connect.CodeOf(err))
		}
	}
}

func TestBudgetRequiresAFamily(t *testing.T) {
	f := newBudgetFixture(t, "2026-03-15")
	_, err := f.h.ListBudgets(context.Background(),
		connect.NewRequest(&financev1.ListBudgetsRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code = %v, want unauthenticated", connect.CodeOf(err))
	}
}

func TestAsOfSelectsAnotherWindow(t *testing.T) {
	f := newBudgetFixture(t, "2026-03-15")
	acc := f.account(t, asA(), "Wallet", "EUR", 1000000)
	food := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	f.budget(t, "Food", food, 50000, financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, "2026-01-01")

	f.spend(t, asA(), acc, food, 11000, "2026-02-10", "february")

	if got := f.statuses(t, "2026-03-15")[0].GetSpent().GetAmountMinor(); got != 0 {
		t.Errorf("March spend = %d, want 0", got)
	}
	if got := f.statuses(t, "2026-02-20")[0].GetSpent().GetAmountMinor(); got != 11000 {
		t.Errorf("February spend = %d, want 11000", got)
	}
}

func TestUpdateAndDeleteBudget(t *testing.T) {
	f := newBudgetFixture(t, "2026-03-15")
	food := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	created := f.budget(t, "Food", food, 10000,
		financev1.BudgetPeriod_BUDGET_PERIOD_MONTH, "2026-03-01")

	updated, err := f.h.UpdateBudget(asA(), connect.NewRequest(&financev1.UpdateBudgetRequest{
		Id: created.GetId(), Name: "Groceries", CategoryId: food.GetId(),
		Limit: money64(25000, "EUR"), Period: financev1.BudgetPeriod_BUDGET_PERIOD_MONTH,
		StartOn: "2026-03-01",
	}))
	if err != nil {
		t.Fatalf("UpdateBudget: %v", err)
	}
	if updated.Msg.GetBudget().GetName() != "Groceries" ||
		updated.Msg.GetBudget().GetLimit().GetAmountMinor() != 25000 {
		t.Errorf("update did not stick: %+v", updated.Msg.GetBudget())
	}

	if _, err := f.h.DeleteBudget(asA(),
		connect.NewRequest(&financev1.DeleteBudgetRequest{Id: created.GetId()})); err != nil {
		t.Fatalf("DeleteBudget: %v", err)
	}
	if len(f.statuses(t, "2026-03-15")) != 0 {
		t.Error("the budget survived deletion")
	}
}

func countSubject(r *recorder, s events.Subject) int {
	var n int
	for _, got := range r.published {
		if got == s {
			n++
		}
	}
	return n
}
