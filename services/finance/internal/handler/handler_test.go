package handler

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/events"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
)

const (
	familyA = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	familyB = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	userA   = "11111111-1111-4111-8111-111111111111"
	userB   = "22222222-2222-4222-8222-222222222222"
)

type fixture struct {
	h     *Handler
	store *fakeStore
	bus   *recorder
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	store := newFakeStore()
	bus := &recorder{}
	h := New(Options{
		Queries:      store,
		Bus:          bus,
		BaseCurrency: "EUR",
		Now:          func() time.Time { return time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC) },
	})
	return &fixture{h: h, store: store, bus: bus}
}

// ctxFor builds a context carrying verified claims, as the auth interceptor does in production.
func ctxFor(userID, familyID string) context.Context {
	return fmauth.WithClaims(context.Background(), &fmauth.Claims{
		UserID: userID, FamilyID: familyID, Email: userID + "@example.test",
	})
}

func asA() context.Context { return ctxFor(userA, familyA) }
func asB() context.Context { return ctxFor(userB, familyB) }

func money64(minor int64, code string) *financev1.Money {
	return &financev1.Money{AmountMinor: minor, CurrencyCode: code}
}

func march() *financev1.DateRange {
	return &financev1.DateRange{From: "2026-03-01", To: "2026-03-31"}
}

/* ------------------------------------------------------------------- helpers */

func (f *fixture) account(t *testing.T, ctx context.Context, name, currency string, opening int64) *financev1.Account {
	t.Helper()
	res, err := f.h.CreateAccount(ctx, connect.NewRequest(&financev1.CreateAccountRequest{
		Name:           name,
		Type:           financev1.AccountType_ACCOUNT_TYPE_CASH,
		CurrencyCode:   currency,
		OpeningBalance: money64(opening, currency),
	}))
	if err != nil {
		t.Fatalf("CreateAccount(%s): %v", name, err)
	}
	return res.Msg.GetAccount()
}

func (f *fixture) category(
	t *testing.T, ctx context.Context, name string, kind financev1.TransactionType,
) *financev1.Category {
	t.Helper()
	res, err := f.h.CreateCategory(ctx, connect.NewRequest(&financev1.CreateCategoryRequest{
		Name: name, Kind: kind, Color: "#123456",
	}))
	if err != nil {
		t.Fatalf("CreateCategory(%s): %v", name, err)
	}
	return res.Msg.GetCategory()
}

func (f *fixture) spend(
	t *testing.T, ctx context.Context, acc *financev1.Account, cat *financev1.Category,
	minor int64, day, note string,
) *financev1.Transaction {
	t.Helper()
	res, err := f.h.CreateTransaction(ctx, connect.NewRequest(&financev1.CreateTransactionRequest{
		AccountId:  acc.GetId(),
		CategoryId: cat.GetId(),
		Type:       financev1.TransactionType_TRANSACTION_TYPE_EXPENSE,
		Amount:     money64(minor, acc.GetCurrencyCode()),
		Note:       note,
		OccurredOn: day,
	}))
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}
	return res.Msg.GetTransaction()
}

/* ------------------------------------------------------------------ accounts */

func TestCreateAccountRequiresAFamily(t *testing.T) {
	f := newFixture(t)

	// Authenticated but family-less: the ledger is a household ledger.
	ctx := fmauth.WithClaims(context.Background(), &fmauth.Claims{UserID: userA})
	_, err := f.h.CreateAccount(ctx, connect.NewRequest(&financev1.CreateAccountRequest{
		Name: "Wallet", CurrencyCode: "EUR",
	}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want failed_precondition", connect.CodeOf(err))
	}

	_, err = f.h.CreateAccount(context.Background(),
		connect.NewRequest(&financev1.CreateAccountRequest{Name: "Wallet", CurrencyCode: "EUR"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("code = %v, want unauthenticated", connect.CodeOf(err))
	}
}

func TestCreateAccountValidatesCurrency(t *testing.T) {
	f := newFixture(t)
	for _, bad := range []string{"", "EU", "EURO", "eu1"} {
		_, err := f.h.CreateAccount(asA(), connect.NewRequest(&financev1.CreateAccountRequest{
			Name: "Wallet", CurrencyCode: bad,
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("currency %q: code = %v, want invalid_argument", bad, connect.CodeOf(err))
		}
	}
}

func TestCreateAccountRejectsMismatchedOpeningCurrency(t *testing.T) {
	f := newFixture(t)
	_, err := f.h.CreateAccount(asA(), connect.NewRequest(&financev1.CreateAccountRequest{
		Name: "Wallet", CurrencyCode: "EUR", OpeningBalance: money64(1000, "USD"),
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestNewAccountBalanceIsItsOpeningBalance(t *testing.T) {
	f := newFixture(t)
	acc := f.account(t, asA(), "Wallet", "EUR", 5000)
	if got := acc.GetBalance().GetAmountMinor(); got != 5000 {
		t.Errorf("balance = %d, want 5000", got)
	}
}

func TestAccountsAreScopedToTheCallersFamily(t *testing.T) {
	f := newFixture(t)
	accA := f.account(t, asA(), "A wallet", "EUR", 100)
	f.account(t, asB(), "B wallet", "EUR", 200)

	list, err := f.h.ListAccounts(asA(), connect.NewRequest(&financev1.ListAccountsRequest{}))
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(list.Msg.GetAccounts()) != 1 || list.Msg.GetAccounts()[0].GetName() != "A wallet" {
		t.Fatalf("family A sees %d accounts, want only its own", len(list.Msg.GetAccounts()))
	}

	// Family B must not be able to read family A's account by id.
	_, err = f.h.GetAccount(asB(), connect.NewRequest(&financev1.GetAccountRequest{Id: accA.GetId()}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("cross-family GetAccount code = %v, want not_found", connect.CodeOf(err))
	}
}

func TestListAccountsTotalsOnlyBaseCurrency(t *testing.T) {
	f := newFixture(t)
	f.account(t, asA(), "Euro wallet", "EUR", 1000)
	f.account(t, asA(), "Dollar wallet", "USD", 9999)

	list, err := f.h.ListAccounts(asA(), connect.NewRequest(&financev1.ListAccountsRequest{}))
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if got := list.Msg.GetTotal().GetAmountMinor(); got != 1000 {
		t.Errorf("total = %d, want 1000 (USD must not be summed without a rate)", got)
	}
	if got := list.Msg.GetTotal().GetCurrencyCode(); got != "EUR" {
		t.Errorf("total currency = %q, want EUR", got)
	}
}

func TestArchivedAccountsAreHiddenByDefault(t *testing.T) {
	f := newFixture(t)
	acc := f.account(t, asA(), "Old card", "EUR", 0)

	if _, err := f.h.UpdateAccount(asA(), connect.NewRequest(&financev1.UpdateAccountRequest{
		Id: acc.GetId(), Name: "Old card", Archived: true,
	})); err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}

	list, _ := f.h.ListAccounts(asA(), connect.NewRequest(&financev1.ListAccountsRequest{}))
	if len(list.Msg.GetAccounts()) != 0 {
		t.Errorf("archived account is still listed")
	}

	all, _ := f.h.ListAccounts(asA(),
		connect.NewRequest(&financev1.ListAccountsRequest{IncludeArchived: true}))
	if len(all.Msg.GetAccounts()) != 1 {
		t.Errorf("include_archived returned %d accounts, want 1", len(all.Msg.GetAccounts()))
	}
}

func TestDeleteAccountWithHistoryIsRefused(t *testing.T) {
	f := newFixture(t)
	acc := f.account(t, asA(), "Wallet", "EUR", 10000)
	cat := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	f.spend(t, asA(), acc, cat, 500, "2026-03-05", "lunch")

	_, err := f.h.DeleteAccount(asA(),
		connect.NewRequest(&financev1.DeleteAccountRequest{Id: acc.GetId()}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want failed_precondition", connect.CodeOf(err))
	}
}

/* ---------------------------------------------------------------- categories */

func TestCategoryKindMustBeIncomeOrExpense(t *testing.T) {
	f := newFixture(t)
	for _, kind := range []financev1.TransactionType{
		financev1.TransactionType_TRANSACTION_TYPE_UNSPECIFIED,
		financev1.TransactionType_TRANSACTION_TYPE_TRANSFER,
	} {
		_, err := f.h.CreateCategory(asA(), connect.NewRequest(&financev1.CreateCategoryRequest{
			Name: "X", Kind: kind,
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("kind %v: code = %v, want invalid_argument", kind, connect.CodeOf(err))
		}
	}
}

func TestListCategoriesFiltersByKind(t *testing.T) {
	f := newFixture(t)
	f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	f.category(t, asA(), "Salary", financev1.TransactionType_TRANSACTION_TYPE_INCOME)

	expenses, err := f.h.ListCategories(asA(), connect.NewRequest(&financev1.ListCategoriesRequest{
		Kind: financev1.TransactionType_TRANSACTION_TYPE_EXPENSE,
	}))
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(expenses.Msg.GetCategories()) != 1 ||
		expenses.Msg.GetCategories()[0].GetName() != "Food" {
		t.Errorf("expense filter returned %v", expenses.Msg.GetCategories())
	}

	both, _ := f.h.ListCategories(asA(), connect.NewRequest(&financev1.ListCategoriesRequest{}))
	if len(both.Msg.GetCategories()) != 2 {
		t.Errorf("unspecified kind returned %d, want both", len(both.Msg.GetCategories()))
	}
}

func TestCategoryCannotBeItsOwnParent(t *testing.T) {
	f := newFixture(t)
	cat := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)

	_, err := f.h.UpdateCategory(asA(), connect.NewRequest(&financev1.UpdateCategoryRequest{
		Id: cat.GetId(), Name: "Food", ParentId: cat.GetId(),
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

/* -------------------------------------------------------------- transactions */

func TestExpenseLowersTheBalanceAndIncomeRaisesIt(t *testing.T) {
	f := newFixture(t)
	acc := f.account(t, asA(), "Wallet", "EUR", 10000)
	food := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	salary := f.category(t, asA(), "Salary", financev1.TransactionType_TRANSACTION_TYPE_INCOME)

	f.spend(t, asA(), acc, food, 2500, "2026-03-05", "groceries")

	if _, err := f.h.CreateTransaction(asA(),
		connect.NewRequest(&financev1.CreateTransactionRequest{
			AccountId:  acc.GetId(),
			CategoryId: salary.GetId(),
			Type:       financev1.TransactionType_TRANSACTION_TYPE_INCOME,
			Amount:     money64(50000, "EUR"),
			OccurredOn: "2026-03-01",
		})); err != nil {
		t.Fatalf("income CreateTransaction: %v", err)
	}

	got, err := f.h.GetAccount(asA(), connect.NewRequest(&financev1.GetAccountRequest{
		Id: acc.GetId(),
	}))
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if want := int64(10000 - 2500 + 50000); got.Msg.GetAccount().GetBalance().GetAmountMinor() != want {
		t.Errorf("balance = %d, want %d", got.Msg.GetAccount().GetBalance().GetAmountMinor(), want)
	}
	if !f.bus.sawSubject(events.SubjectFinanceTransactionCreated) {
		t.Error("expected finance.transaction.created")
	}
}

func TestAmountMustBePositive(t *testing.T) {
	f := newFixture(t)
	acc := f.account(t, asA(), "Wallet", "EUR", 0)
	cat := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)

	for _, minor := range []int64{0, -100} {
		_, err := f.h.CreateTransaction(asA(),
			connect.NewRequest(&financev1.CreateTransactionRequest{
				AccountId:  acc.GetId(),
				CategoryId: cat.GetId(),
				Type:       financev1.TransactionType_TRANSACTION_TYPE_EXPENSE,
				Amount:     money64(minor, "EUR"),
				OccurredOn: "2026-03-05",
			}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("amount %d: code = %v, want invalid_argument", minor, connect.CodeOf(err))
		}
	}
}

func TestCategoryKindMustMatchTransactionType(t *testing.T) {
	f := newFixture(t)
	acc := f.account(t, asA(), "Wallet", "EUR", 10000)
	salary := f.category(t, asA(), "Salary", financev1.TransactionType_TRANSACTION_TYPE_INCOME)

	// Filing an expense under an income category would make every report wrong.
	_, err := f.h.CreateTransaction(asA(), connect.NewRequest(&financev1.CreateTransactionRequest{
		AccountId:  acc.GetId(),
		CategoryId: salary.GetId(),
		Type:       financev1.TransactionType_TRANSACTION_TYPE_EXPENSE,
		Amount:     money64(100, "EUR"),
		OccurredOn: "2026-03-05",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestTransactionCannotUseAnotherFamilysAccount(t *testing.T) {
	f := newFixture(t)
	accB := f.account(t, asB(), "B wallet", "EUR", 10000)
	catA := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)

	_, err := f.h.CreateTransaction(asA(), connect.NewRequest(&financev1.CreateTransactionRequest{
		AccountId:  accB.GetId(),
		CategoryId: catA.GetId(),
		Type:       financev1.TransactionType_TRANSACTION_TYPE_EXPENSE,
		Amount:     money64(100, "EUR"),
		OccurredOn: "2026-03-05",
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("code = %v, want not_found", connect.CodeOf(err))
	}
}

func TestAmountCurrencyMustMatchTheAccount(t *testing.T) {
	f := newFixture(t)
	acc := f.account(t, asA(), "Wallet", "EUR", 10000)
	cat := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)

	_, err := f.h.CreateTransaction(asA(), connect.NewRequest(&financev1.CreateTransactionRequest{
		AccountId:  acc.GetId(),
		CategoryId: cat.GetId(),
		Type:       financev1.TransactionType_TRANSACTION_TYPE_EXPENSE,
		Amount:     money64(100, "USD"),
		OccurredOn: "2026-03-05",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestOccurredOnMustBeACalendarDate(t *testing.T) {
	f := newFixture(t)
	acc := f.account(t, asA(), "Wallet", "EUR", 10000)
	cat := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)

	for _, day := range []string{"", "05/03/2026", "2026-02-30"} {
		_, err := f.h.CreateTransaction(asA(),
			connect.NewRequest(&financev1.CreateTransactionRequest{
				AccountId:  acc.GetId(),
				CategoryId: cat.GetId(),
				Type:       financev1.TransactionType_TRANSACTION_TYPE_EXPENSE,
				Amount:     money64(100, "EUR"),
				OccurredOn: day,
			}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("date %q: code = %v, want invalid_argument", day, connect.CodeOf(err))
		}
	}
}

func TestTransferMovesMoneyBetweenAccounts(t *testing.T) {
	f := newFixture(t)
	from := f.account(t, asA(), "Wallet", "EUR", 10000)
	to := f.account(t, asA(), "Savings", "EUR", 0)

	if _, err := f.h.CreateTransaction(asA(),
		connect.NewRequest(&financev1.CreateTransactionRequest{
			AccountId:        from.GetId(),
			CounterAccountId: to.GetId(),
			Type:             financev1.TransactionType_TRANSACTION_TYPE_TRANSFER,
			Amount:           money64(3000, "EUR"),
			OccurredOn:       "2026-03-10",
		})); err != nil {
		t.Fatalf("transfer: %v", err)
	}

	fromAfter, _ := f.h.GetAccount(asA(),
		connect.NewRequest(&financev1.GetAccountRequest{Id: from.GetId()}))
	toAfter, _ := f.h.GetAccount(asA(),
		connect.NewRequest(&financev1.GetAccountRequest{Id: to.GetId()}))

	if got := fromAfter.Msg.GetAccount().GetBalance().GetAmountMinor(); got != 7000 {
		t.Errorf("source balance = %d, want 7000", got)
	}
	if got := toAfter.Msg.GetAccount().GetBalance().GetAmountMinor(); got != 3000 {
		t.Errorf("destination balance = %d, want 3000", got)
	}
}

func TestTransferNeedsTwoDifferentAccounts(t *testing.T) {
	f := newFixture(t)
	acc := f.account(t, asA(), "Wallet", "EUR", 10000)

	_, err := f.h.CreateTransaction(asA(), connect.NewRequest(&financev1.CreateTransactionRequest{
		AccountId:        acc.GetId(),
		CounterAccountId: acc.GetId(),
		Type:             financev1.TransactionType_TRANSACTION_TYPE_TRANSFER,
		Amount:           money64(100, "EUR"),
		OccurredOn:       "2026-03-10",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestCrossCurrencyTransferIsRefused(t *testing.T) {
	f := newFixture(t)
	from := f.account(t, asA(), "Euro wallet", "EUR", 10000)
	to := f.account(t, asA(), "Dollar wallet", "USD", 0)

	_, err := f.h.CreateTransaction(asA(), connect.NewRequest(&financev1.CreateTransactionRequest{
		AccountId:        from.GetId(),
		CounterAccountId: to.GetId(),
		Type:             financev1.TransactionType_TRANSACTION_TYPE_TRANSFER,
		Amount:           money64(1000, "EUR"),
		OccurredOn:       "2026-03-10",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestDeleteTransactionRestoresTheBalance(t *testing.T) {
	f := newFixture(t)
	acc := f.account(t, asA(), "Wallet", "EUR", 10000)
	cat := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	tx := f.spend(t, asA(), acc, cat, 2500, "2026-03-05", "groceries")

	if _, err := f.h.DeleteTransaction(asA(),
		connect.NewRequest(&financev1.DeleteTransactionRequest{Id: tx.GetId()})); err != nil {
		t.Fatalf("DeleteTransaction: %v", err)
	}

	got, _ := f.h.GetAccount(asA(), connect.NewRequest(&financev1.GetAccountRequest{Id: acc.GetId()}))
	if b := got.Msg.GetAccount().GetBalance().GetAmountMinor(); b != 10000 {
		t.Errorf("balance after delete = %d, want 10000", b)
	}
	if !f.bus.sawSubject(events.SubjectFinanceTransactionDeleted) {
		t.Error("expected finance.transaction.deleted")
	}
}

func TestDeleteAnotherFamilysTransactionIsNotFound(t *testing.T) {
	f := newFixture(t)
	acc := f.account(t, asA(), "Wallet", "EUR", 10000)
	cat := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	tx := f.spend(t, asA(), acc, cat, 100, "2026-03-05", "x")

	_, err := f.h.DeleteTransaction(asB(),
		connect.NewRequest(&financev1.DeleteTransactionRequest{Id: tx.GetId()}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("code = %v, want not_found", connect.CodeOf(err))
	}
}

/* ---------------------------------------------------------------- pagination */

func TestListTransactionsPagesWithACursor(t *testing.T) {
	f := newFixture(t)
	f.h.defaultPageSize = 2
	acc := f.account(t, asA(), "Wallet", "EUR", 100000)
	cat := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)

	for _, day := range []string{"2026-03-01", "2026-03-02", "2026-03-03", "2026-03-04", "2026-03-05"} {
		f.spend(t, asA(), acc, cat, 100, day, "n")
	}

	var seen []string
	token := ""
	for pages := 0; pages < 10; pages++ {
		res, err := f.h.ListTransactions(asA(),
			connect.NewRequest(&financev1.ListTransactionsRequest{
				Range: march(), PageToken: token,
			}))
		if err != nil {
			t.Fatalf("ListTransactions: %v", err)
		}
		for _, tx := range res.Msg.GetTransactions() {
			seen = append(seen, tx.GetOccurredOn())
		}
		token = res.Msg.GetNextPageToken()
		if token == "" {
			break
		}
	}

	if len(seen) != 5 {
		t.Fatalf("paged through %d transactions, want 5 (%v)", len(seen), seen)
	}
	// Newest first, and no row served twice.
	for i := 1; i < len(seen); i++ {
		if seen[i] >= seen[i-1] {
			t.Fatalf("order broken at %d: %v", i, seen)
		}
	}
}

func TestListTransactionsRejectsAForeignCursor(t *testing.T) {
	f := newFixture(t)
	_, err := f.h.ListTransactions(asA(), connect.NewRequest(&financev1.ListTransactionsRequest{
		Range: march(), PageToken: "not-a-cursor!!",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestPageSizeIsCapped(t *testing.T) {
	f := newFixture(t)
	f.h.maxPageSize = 10
	if got := f.h.pageSize(5000); got != 10 {
		t.Errorf("pageSize(5000) = %d, want 10", got)
	}
	if got := f.h.pageSize(0); got != f.h.defaultPageSize {
		t.Errorf("pageSize(0) = %d, want the default", got)
	}
}

func TestListTransactionsFiltersBySearchAndRange(t *testing.T) {
	f := newFixture(t)
	acc := f.account(t, asA(), "Wallet", "EUR", 100000)
	cat := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	f.spend(t, asA(), acc, cat, 100, "2026-03-05", "Coffee beans")
	f.spend(t, asA(), acc, cat, 200, "2026-03-06", "Bus ticket")
	f.spend(t, asA(), acc, cat, 300, "2026-04-01", "Coffee machine")

	res, err := f.h.ListTransactions(asA(), connect.NewRequest(&financev1.ListTransactionsRequest{
		Range: march(), Search: "coffee",
	}))
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if len(res.Msg.GetTransactions()) != 1 {
		t.Fatalf("got %d results, want 1 (April must be out of range)", len(res.Msg.GetTransactions()))
	}
	if res.Msg.GetTransactions()[0].GetNote() != "Coffee beans" {
		t.Errorf("wrong row: %q", res.Msg.GetTransactions()[0].GetNote())
	}
}

func TestListTransactionsRejectsABackwardsRange(t *testing.T) {
	f := newFixture(t)
	_, err := f.h.ListTransactions(asA(), connect.NewRequest(&financev1.ListTransactionsRequest{
		Range: &financev1.DateRange{From: "2026-03-31", To: "2026-03-01"},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

/* ------------------------------------------------------------------- reports */

func TestSummaryExcludesTransfers(t *testing.T) {
	f := newFixture(t)
	wallet := f.account(t, asA(), "Wallet", "EUR", 100000)
	savings := f.account(t, asA(), "Savings", "EUR", 0)
	food := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	salary := f.category(t, asA(), "Salary", financev1.TransactionType_TRANSACTION_TYPE_INCOME)

	f.spend(t, asA(), wallet, food, 2500, "2026-03-05", "groceries")
	if _, err := f.h.CreateTransaction(asA(),
		connect.NewRequest(&financev1.CreateTransactionRequest{
			AccountId: wallet.GetId(), CategoryId: salary.GetId(),
			Type:       financev1.TransactionType_TRANSACTION_TYPE_INCOME,
			Amount:     money64(200000, "EUR"),
			OccurredOn: "2026-03-01",
		})); err != nil {
		t.Fatalf("income: %v", err)
	}
	if _, err := f.h.CreateTransaction(asA(),
		connect.NewRequest(&financev1.CreateTransactionRequest{
			AccountId: wallet.GetId(), CounterAccountId: savings.GetId(),
			Type:       financev1.TransactionType_TRANSACTION_TYPE_TRANSFER,
			Amount:     money64(50000, "EUR"),
			OccurredOn: "2026-03-15",
		})); err != nil {
		t.Fatalf("transfer: %v", err)
	}

	res, err := f.h.GetSummary(asA(), connect.NewRequest(&financev1.GetSummaryRequest{
		Range: march(),
	}))
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if got := res.Msg.GetIncome().GetAmountMinor(); got != 200000 {
		t.Errorf("income = %d, want 200000 (the transfer is not income)", got)
	}
	if got := res.Msg.GetExpense().GetAmountMinor(); got != 2500 {
		t.Errorf("expense = %d, want 2500 (the transfer is not spending)", got)
	}
	if got := res.Msg.GetNet().GetAmountMinor(); got != 197500 {
		t.Errorf("net = %d, want 197500", got)
	}
	// The balance is "now": opening 100000 + 200000 - 2500, and the transfer nets to zero.
	if got := res.Msg.GetBalance().GetAmountMinor(); got != 297500 {
		t.Errorf("balance = %d, want 297500", got)
	}
}

func TestCategoryBreakdownSharesSumToOne(t *testing.T) {
	f := newFixture(t)
	acc := f.account(t, asA(), "Wallet", "EUR", 100000)
	food := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	travel := f.category(t, asA(), "Travel", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)

	f.spend(t, asA(), acc, food, 7500, "2026-03-05", "groceries")
	f.spend(t, asA(), acc, travel, 2500, "2026-03-06", "train")

	res, err := f.h.GetCategoryBreakdown(asA(),
		connect.NewRequest(&financev1.GetCategoryBreakdownRequest{
			Range: march(), Type: financev1.TransactionType_TRANSACTION_TYPE_EXPENSE,
		}))
	if err != nil {
		t.Fatalf("GetCategoryBreakdown: %v", err)
	}

	slices := res.Msg.GetSlices()
	if len(slices) != 2 {
		t.Fatalf("got %d slices, want 2", len(slices))
	}
	// Ordered by size, so Food comes first.
	if slices[0].GetCategoryName() != "Food" || slices[0].GetShare() != 0.75 {
		t.Errorf("first slice = %s %.2f, want Food 0.75",
			slices[0].GetCategoryName(), slices[0].GetShare())
	}
	if total := res.Msg.GetTotal().GetAmountMinor(); total != 10000 {
		t.Errorf("total = %d, want 10000", total)
	}

	var sum float64
	for _, s := range slices {
		sum += s.GetShare()
	}
	if sum < 0.999 || sum > 1.001 {
		t.Errorf("shares sum to %v, want 1", sum)
	}
}

func TestEmptyBreakdownDoesNotDivideByZero(t *testing.T) {
	f := newFixture(t)
	res, err := f.h.GetCategoryBreakdown(asA(),
		connect.NewRequest(&financev1.GetCategoryBreakdownRequest{
			Range: march(), Type: financev1.TransactionType_TRANSACTION_TYPE_EXPENSE,
		}))
	if err != nil {
		t.Fatalf("GetCategoryBreakdown: %v", err)
	}
	if len(res.Msg.GetSlices()) != 0 || res.Msg.GetTotal().GetAmountMinor() != 0 {
		t.Errorf("empty period should produce no slices and a zero total")
	}
}

func TestBreakdownRequiresIncomeOrExpense(t *testing.T) {
	f := newFixture(t)
	for _, kind := range []financev1.TransactionType{
		financev1.TransactionType_TRANSACTION_TYPE_UNSPECIFIED,
		financev1.TransactionType_TRANSACTION_TYPE_TRANSFER,
	} {
		_, err := f.h.GetCategoryBreakdown(asA(),
			connect.NewRequest(&financev1.GetCategoryBreakdownRequest{Range: march(), Type: kind}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("type %v: code = %v, want invalid_argument", kind, connect.CodeOf(err))
		}
	}
}

func TestReportsAreScopedToTheCallersFamily(t *testing.T) {
	f := newFixture(t)
	accA := f.account(t, asA(), "A wallet", "EUR", 0)
	catA := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	f.spend(t, asA(), accA, catA, 5000, "2026-03-05", "groceries")

	res, err := f.h.GetSummary(asB(), connect.NewRequest(&financev1.GetSummaryRequest{
		Range: march(),
	}))
	if err != nil {
		t.Fatalf("GetSummary: %v", err)
	}
	if res.Msg.GetExpense().GetAmountMinor() != 0 {
		t.Errorf("family B sees family A's spending: %d", res.Msg.GetExpense().GetAmountMinor())
	}
}

/* -------------------------------------------------------------------- misc */

func TestPublishFailureDoesNotFailTheWrite(t *testing.T) {
	f := newFixture(t)
	acc := f.account(t, asA(), "Wallet", "EUR", 10000)
	cat := f.category(t, asA(), "Food", financev1.TransactionType_TRANSACTION_TYPE_EXPENSE)
	f.bus.err = errBoom

	tx := f.spend(t, asA(), acc, cat, 100, "2026-03-05", "x")
	if tx.GetId() == "" {
		t.Fatal("the transaction should exist despite the publish failure")
	}
}

func TestStoreFailureBecomesInternal(t *testing.T) {
	f := newFixture(t)
	f.store.failOn["CreateAccount"] = errBoom

	_, err := f.h.CreateAccount(asA(), connect.NewRequest(&financev1.CreateAccountRequest{
		Name: "Wallet", CurrencyCode: "EUR",
	}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code = %v, want internal", connect.CodeOf(err))
	}
	if strings.Contains(err.Error(), errBoom.Error()) {
		t.Fatalf("wire message leaked the cause: %q", err.Error())
	}
}

func TestCursorRoundTrip(t *testing.T) {
	token := encodeCursor("2026-03-05", "00000000-0000-4000-8000-000000000001")
	day, txID, err := decodeCursor(token)
	if err != nil {
		t.Fatalf("decodeCursor: %v", err)
	}
	if day != "2026-03-05" || txID != "00000000-0000-4000-8000-000000000001" {
		t.Errorf("round trip = %q %q", day, txID)
	}
	if _, _, err := decodeCursor("$$$"); err == nil {
		t.Error("expected an error for a malformed cursor")
	}
}
