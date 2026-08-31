package handler

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

// The auth interceptor is not in the unit-test call path, so claims are stamped onto the
// context directly — the same shape the interceptor would have produced.
func withClaims(ctx context.Context, userID, familyID string) context.Context {
	return fmauth.WithClaims(ctx, &fmauth.Claims{UserID: userID, FamilyID: familyID})
}

const (
	testFamily = "00000000-0000-4000-8000-000000000001"
	sergiy     = "00000000-0000-4000-8000-000000000002"
	olena      = "00000000-0000-4000-8000-000000000003"
	otherFam   = "00000000-0000-4000-8000-000000000009"
)

// testNow is fixed so budget windows, "today" and the series buckets are the same on every
// run and in every timezone the test machine happens to sit in.
var testNow = time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC)

func newTestHandler(t *testing.T) (*Handler, *fakeStore, *recorder) {
	t.Helper()
	store := newFakeStore()
	rec := &recorder{}
	h := New(Options{
		Queries: store, Tx: store, Bus: rec,
		Now: func() time.Time { return testNow },
	})
	seedHousehold(store)
	return h, store, rec
}

// seedHousehold gives the store the settings row and the two members every screen assumes.
func seedHousehold(s *fakeStore) {
	family := pgconv.MustUUID(testFamily)
	s.settings[testFamily] = db.FinanceSetting{
		FamilyID: family, BaseCurrencyCode: "UAH", Timezone: "UTC",
		WeekStartsOn: "monday", OverspendNotificationsEnabled: true,
	}
	for _, m := range []struct {
		id, name, role string
	}{{sergiy, "Сергій", "owner"}, {olena, "Олена", "member"}} {
		uid := pgconv.MustUUID(m.id)
		s.members[memberKey(family, uid)] = db.FinanceMember{
			FamilyID: family, UserID: uid, DisplayName: m.name,
			Role: m.role, Status: "active",
		}
	}
}

func ctxOf(user string) context.Context {
	return withClaims(context.Background(), user, testFamily)
}

// seedAccount writes an account straight into the store, bypassing the handler, so a test can
// set up state the handler would refuse to create (another member's private account).
func seedAccount(s *fakeStore, name, kind, visibility, owner string, opening int64) db.Account {
	a := db.Account{
		ID: s.newUUID(), FamilyID: pgconv.MustUUID(testFamily), Name: name, Kind: kind,
		Visibility: visibility, CurrencyCode: "UAH", OpeningBalanceMinor: opening,
	}
	if owner != "" {
		a.OwnerMemberID = pgconv.MustUUID(owner)
		a.ExcludedFromFamilyTotal = visibility == visibilityPrivate
	}
	s.accounts[id(a.ID)] = a
	return a
}

func seedGroupAndCategory(s *fakeStore, groupName, categoryName string) (db.CategoryGroup, db.Category) {
	g := db.CategoryGroup{
		ID: s.newUUID(), FamilyID: pgconv.MustUUID(testFamily), Name: groupName, Kind: kindExpense,
	}
	s.groups[id(g.ID)] = g
	c := db.Category{
		ID: s.newUUID(), FamilyID: g.FamilyID, GroupID: g.ID, Name: categoryName, Kind: kindExpense,
	}
	s.categories[id(c.ID)] = c
	return g, c
}

func day(s string) pgtype.Date {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		panic(err)
	}
	return pgtype.Date{Time: t, Valid: true}
}

func seedTransaction(s *fakeStore, account db.Account, category db.Category, member string, amount int64, on string) db.Transaction {
	t := db.Transaction{
		ID: s.newUUID(), FamilyID: pgconv.MustUUID(testFamily), Type: kindExpense,
		AccountID: account.ID, CategoryID: category.ID, AmountMinor: amount,
		CurrencyCode: "UAH", OccurredOn: day(on), MemberID: pgconv.MustUUID(member),
		CreatedByUserID: pgconv.MustUUID(member),
	}
	s.transactions[id(t.ID)] = t
	return t
}

func codeOf(t *testing.T, err error) connect.Code {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	return connect.CodeOf(err)
}

/* ============================ the private-account boundary ================= */
//
// These are the tests the schema comment and the query comments both point at. Everything else
// in this service is a ledger; this is the part where getting it wrong shows another member's
// money on someone's phone.

func TestListAccountsHidesAnotherMembersPrivateAccounts(t *testing.T) {
	h, store, _ := newTestHandler(t)
	shared := seedAccount(store, "Mono", "card", visibilityShared, "", 1_000_00)
	mine := seedAccount(store, "Мій гаманець", "cash", visibilityPrivate, sergiy, 500_00)
	hers := seedAccount(store, "Олена USD", "bank", visibilityPrivate, olena, 142_000_00)

	resp, err := h.ListAccounts(ctxOf(sergiy), connect.NewRequest(&financev1.ListAccountsRequest{}))
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}

	if len(resp.Msg.Shared) != 1 || resp.Msg.Shared[0].Id != id(shared.ID) {
		t.Errorf("shared = %v, want just the shared account", resp.Msg.Shared)
	}
	if len(resp.Msg.PrivateOwn) != 1 || resp.Msg.PrivateOwn[0].Id != id(mine.ID) {
		t.Errorf("private_own = %v, want just the caller's own private account", resp.Msg.PrivateOwn)
	}
	if len(resp.Msg.Hidden) != 1 {
		t.Fatalf("hidden = %d summaries, want 1", len(resp.Msg.Hidden))
	}
	if got := resp.Msg.Hidden[0]; got.MemberId != olena || got.AccountCount != 1 {
		t.Errorf("hidden = %+v, want olena with 1 account", got)
	}
	// The count is the whole of what may cross the wire. Nothing else about hers may appear.
	for _, a := range append(resp.Msg.Shared, resp.Msg.PrivateOwn...) {
		if a.Id == id(hers.ID) {
			t.Fatal("another member's private account was serialised in full")
		}
	}
}

func TestSharedBalanceExcludesPrivateAccounts(t *testing.T) {
	h, store, _ := newTestHandler(t)
	seedAccount(store, "Mono", "card", visibilityShared, "", 1_000_00)
	seedAccount(store, "Скарбничка", accountKindSavings, visibilityShared, "", 5_000_00)
	seedAccount(store, "Олена USD", "bank", visibilityPrivate, olena, 999_999_00)

	resp, err := h.ListAccounts(ctxOf(sergiy), connect.NewRequest(&financev1.ListAccountsRequest{}))
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if got := resp.Msg.SharedBalance.AmountMinor; got != 1_000_00 {
		t.Errorf("shared_balance = %d, want 100000 (private and savings excluded)", got)
	}
	if got := resp.Msg.SavingsTotal.AmountMinor; got != 5_000_00 {
		t.Errorf("savings_total = %d, want 500000", got)
	}
}

func TestGetAccountOnAnotherMembersPrivateAccountIsNotFound(t *testing.T) {
	h, store, _ := newTestHandler(t)
	hers := seedAccount(store, "Олена USD", "bank", visibilityPrivate, olena, 142_000_00)

	_, err := h.GetAccount(ctxOf(sergiy), connect.NewRequest(&financev1.GetAccountRequest{
		AccountId: id(hers.ID),
	}))
	// NotFound, not PermissionDenied: "this exists but is not yours" is itself the leak.
	if got := codeOf(t, err); got != connect.CodeNotFound {
		t.Errorf("code = %v, want NotFound", got)
	}
}

// Reordering writes to the list the caller can see. An account they may not see does not move,
// which is what an id that does not exist already did.
func TestReorderAccountsLeavesAnotherMembersPrivateAccountWhereItWas(t *testing.T) {
	h, store, _ := newTestHandler(t)
	shared := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	hers := seedAccount(store, "Олена USD", "bank", visibilityPrivate, olena, 0)

	if _, err := h.ReorderAccounts(ctxOf(sergiy), connect.NewRequest(&financev1.ReorderAccountsRequest{
		AccountIdsInOrder: []string{id(hers.ID), id(shared.ID)},
	})); err != nil {
		t.Fatalf("ReorderAccounts: %v", err)
	}
	if got := store.accounts[id(hers.ID)].SortOrder; got != 0 {
		t.Errorf("her sort_order = %d, want 0 — untouched", got)
	}
	if got := store.accounts[id(shared.ID)].SortOrder; got != 1 {
		t.Errorf("shared sort_order = %d, want 1", got)
	}
}

func TestPrivateAccountIsAlwaysOwnedByTheCaller(t *testing.T) {
	h, _, _ := newTestHandler(t)

	resp, err := h.CreateAccount(ctxOf(sergiy), connect.NewRequest(&financev1.CreateAccountRequest{
		Name:       "Приватний",
		Kind:       financev1.AccountKind_ACCOUNT_KIND_CASH,
		Visibility: financev1.AccountVisibility_ACCOUNT_VISIBILITY_PRIVATE,
	}))
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if got := resp.Msg.Account.OwnerMemberId; got != sergiy {
		t.Errorf("owner_member_id = %q, want the caller %q", got, sergiy)
	}
	// A private account is out of the family headline by construction, not by the client
	// remembering to say so.
	if !resp.Msg.Account.ExcludedFromFamilyTotal {
		t.Error("a private account must be excluded from the family total")
	}
}

func TestAnotherMembersPrivateSpendIsInvisible(t *testing.T) {
	h, store, _ := newTestHandler(t)
	_, category := seedGroupAndCategory(store, "Розваги", "Кіно")
	shared := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	herPrivate := seedAccount(store, "Олена USD", "bank", visibilityPrivate, olena, 0)

	seedTransaction(store, shared, category, sergiy, 100_00, "2026-08-10")
	hidden := seedTransaction(store, herPrivate, category, olena, 900_00, "2026-08-11")

	resp, err := h.ListTransactions(ctxOf(sergiy), connect.NewRequest(&financev1.ListTransactionsRequest{}))
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	for _, section := range resp.Msg.Days {
		for _, tx := range section.Transactions {
			if tx.Id == id(hidden.ID) {
				t.Fatal("a transaction from another member's private account reached the feed")
			}
		}
	}
	if got := resp.Msg.PeriodTotal.AmountMinor; got != 100_00 {
		t.Errorf("period_total = %d, want 10000 — private spend must not be summed for others", got)
	}

	// The same transaction is visible to its own owner, which is what makes the exclusion a
	// privacy rule rather than a bug.
	hers, err := h.GetTransaction(ctxOf(olena), connect.NewRequest(&financev1.GetTransactionRequest{
		TransactionId: id(hidden.ID),
	}))
	if err != nil {
		t.Fatalf("GetTransaction as the owner: %v", err)
	}
	if hers.Msg.Transaction.Amount.AmountMinor != 900_00 {
		t.Errorf("amount = %d, want 90000", hers.Msg.Transaction.Amount.AmountMinor)
	}
}

func TestTransactionOnAnotherMembersPrivateAccountIsRefused(t *testing.T) {
	h, store, _ := newTestHandler(t)
	_, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	herPrivate := seedAccount(store, "Олена USD", "bank", visibilityPrivate, olena, 0)

	_, err := h.CreateTransaction(ctxOf(sergiy), connect.NewRequest(&financev1.CreateTransactionRequest{
		Type:       financev1.TransactionType_TRANSACTION_TYPE_EXPENSE,
		AccountId:  id(herPrivate.ID),
		CategoryId: id(category.ID),
		Amount:     &financev1.Money{AmountMinor: 50_00, CurrencyCode: "UAH"},
	}))
	if got := codeOf(t, err); got != connect.CodeNotFound {
		t.Errorf("code = %v, want NotFound", got)
	}
}

func TestAggregatesExcludeAnotherMembersPrivateSpend(t *testing.T) {
	h, store, _ := newTestHandler(t)
	group, category := seedGroupAndCategory(store, "Розваги", "Кіно")
	shared := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	herPrivate := seedAccount(store, "Олена USD", "bank", visibilityPrivate, olena, 0)
	seedTransaction(store, shared, category, sergiy, 100_00, "2026-08-10")
	seedTransaction(store, herPrivate, category, olena, 900_00, "2026-08-11")

	resp, err := h.GetHomeSummary(ctxOf(sergiy), connect.NewRequest(&financev1.GetHomeSummaryRequest{
		Kind: financev1.TransactionKind_TRANSACTION_KIND_EXPENSE,
	}))
	if err != nil {
		t.Fatalf("GetHomeSummary: %v", err)
	}
	if got := resp.Msg.PeriodTotal.AmountMinor; got != 100_00 {
		t.Errorf("period_total = %d, want 10000", got)
	}
	if len(resp.Msg.Slices) != 1 {
		t.Fatalf("slices = %d, want 1", len(resp.Msg.Slices))
	}
	if resp.Msg.Slices[0].GroupId != id(group.ID) || resp.Msg.Slices[0].Amount.AmountMinor != 100_00 {
		t.Errorf("slice = %+v, want the shared spend only", resp.Msg.Slices[0])
	}
}

func TestCrossFamilyAccessIsNotFound(t *testing.T) {
	h, store, _ := newTestHandler(t)
	shared := seedAccount(store, "Mono", "card", visibilityShared, "", 0)

	ctx := withClaims(context.Background(), sergiy, otherFam)
	_, err := h.GetAccount(ctx, connect.NewRequest(&financev1.GetAccountRequest{
		AccountId: id(shared.ID),
	}))
	// The other family has no settings row, so the household precondition fires before the
	// account read — either way, nothing about the account crosses.
	if got := codeOf(t, err); got != connect.CodeNotFound && got != connect.CodeFailedPrecondition {
		t.Errorf("code = %v, want NotFound or FailedPrecondition", got)
	}
}

/* ================================ identity ================================= */

func TestCallerWithoutFamilyIsRefused(t *testing.T) {
	h, _, _ := newTestHandler(t)
	ctx := withClaims(context.Background(), sergiy, "")

	_, err := h.ListAccounts(ctx, connect.NewRequest(&financev1.ListAccountsRequest{}))
	if got := codeOf(t, err); got != connect.CodeFailedPrecondition {
		t.Errorf("code = %v, want FailedPrecondition", got)
	}
}

func TestUnauthenticatedCallerIsRefused(t *testing.T) {
	h, _, _ := newTestHandler(t)

	_, err := h.ListAccounts(context.Background(), connect.NewRequest(&financev1.ListAccountsRequest{}))
	if got := codeOf(t, err); got != connect.CodeUnauthenticated {
		t.Errorf("code = %v, want Unauthenticated", got)
	}
}

/* ============================== transactions =============================== */

func TestCreateTransactionAttributesThreePeopleSeparately(t *testing.T) {
	h, store, rec := newTestHandler(t)
	_, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)

	// Сергій types it, Олена spent it, the account belongs to the household. All three are
	// different facts and the row has to keep them apart.
	resp, err := h.CreateTransaction(ctxOf(sergiy), connect.NewRequest(&financev1.CreateTransactionRequest{
		Type:       financev1.TransactionType_TRANSACTION_TYPE_EXPENSE,
		AccountId:  id(account.ID),
		CategoryId: id(category.ID),
		Amount:     &financev1.Money{AmountMinor: 50_00, CurrencyCode: "UAH"},
		MemberId:   olena,
		Merchant:   "jetbrains",
	}))
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}
	tx := resp.Msg.Transaction
	if tx.MemberId != olena {
		t.Errorf("member_id = %q, want %q (who spent)", tx.MemberId, olena)
	}
	if tx.CreatedByUserId != sergiy {
		t.Errorf("created_by_user_id = %q, want %q (who typed)", tx.CreatedByUserId, sergiy)
	}
	if tx.OccurredOn != "2026-08-30" {
		t.Errorf("occurred_on = %q, want today in the household timezone", tx.OccurredOn)
	}
	if !rec.sawSubject("finance.transaction.created") {
		t.Errorf("subjects = %v, want finance.transaction.created", rec.subjects)
	}
}

func TestCreateTransactionDefaultsMemberToCaller(t *testing.T) {
	h, store, _ := newTestHandler(t)
	_, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)

	resp, err := h.CreateTransaction(ctxOf(sergiy), connect.NewRequest(&financev1.CreateTransactionRequest{
		Type: financev1.TransactionType_TRANSACTION_TYPE_EXPENSE, AccountId: id(account.ID),
		CategoryId: id(category.ID),
		Amount:     &financev1.Money{AmountMinor: 50_00, CurrencyCode: "UAH"},
	}))
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}
	if resp.Msg.Transaction.MemberId != sergiy {
		t.Errorf("member_id = %q, want the caller", resp.Msg.Transaction.MemberId)
	}
}

func TestCreateTransactionValidation(t *testing.T) {
	h, store, _ := newTestHandler(t)
	_, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)

	cases := map[string]*financev1.CreateTransactionRequest{
		"no account": {
			Type:   financev1.TransactionType_TRANSACTION_TYPE_EXPENSE,
			Amount: &financev1.Money{AmountMinor: 1, CurrencyCode: "UAH"},
		},
		"zero amount": {
			Type: financev1.TransactionType_TRANSACTION_TYPE_EXPENSE, AccountId: id(account.ID),
			Amount: &financev1.Money{AmountMinor: 0, CurrencyCode: "UAH"},
		},
		"negative amount": {
			Type: financev1.TransactionType_TRANSACTION_TYPE_EXPENSE, AccountId: id(account.ID),
			Amount: &financev1.Money{AmountMinor: -1, CurrencyCode: "UAH"},
		},
		"transfer through the wrong rpc": {
			Type: financev1.TransactionType_TRANSACTION_TYPE_TRANSFER, AccountId: id(account.ID),
			Amount: &financev1.Money{AmountMinor: 1, CurrencyCode: "UAH"},
		},
		"account id is not a uuid": {
			Type: financev1.TransactionType_TRANSACTION_TYPE_EXPENSE, AccountId: "not-a-uuid",
			Amount: &financev1.Money{AmountMinor: 1, CurrencyCode: "UAH"},
		},
		"note too long": {
			Type: financev1.TransactionType_TRANSACTION_TYPE_EXPENSE, AccountId: id(account.ID),
			CategoryId: id(category.ID),
			Amount:     &financev1.Money{AmountMinor: 1, CurrencyCode: "UAH"},
			Note:       string(make([]rune, maxNoteRunes+1)),
		},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := h.CreateTransaction(ctxOf(sergiy), connect.NewRequest(req))
			if got := codeOf(t, err); got != connect.CodeInvalidArgument {
				t.Errorf("code = %v, want InvalidArgument", got)
			}
		})
	}
}

// A transaction is stored in the account's currency, so an amount labelled with another one is
// refused rather than silently relabelled — 50 USD into a UAH account is not 50 UAH.
func TestCreateTransactionRefusesForeignCurrency(t *testing.T) {
	h, store, _ := newTestHandler(t)
	_, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)

	_, err := h.CreateTransaction(ctxOf(sergiy), connect.NewRequest(&financev1.CreateTransactionRequest{
		Type: financev1.TransactionType_TRANSACTION_TYPE_EXPENSE, AccountId: id(account.ID),
		CategoryId: id(category.ID),
		Amount:     &financev1.Money{AmountMinor: 50_00, CurrencyCode: "USD"},
	}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", got)
	}
}

// An empty currency code means "whatever the account holds" and stays accepted: the field is
// informational on the way in, and clients that omit it are not sending a mismatch.
func TestCreateTransactionAcceptsAnEmptyCurrency(t *testing.T) {
	h, store, _ := newTestHandler(t)
	_, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)

	resp, err := h.CreateTransaction(ctxOf(sergiy), connect.NewRequest(&financev1.CreateTransactionRequest{
		Type: financev1.TransactionType_TRANSACTION_TYPE_EXPENSE, AccountId: id(account.ID),
		CategoryId: id(category.ID),
		Amount:     &financev1.Money{AmountMinor: 50_00},
	}))
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}
	if got := resp.Msg.Transaction.GetAmount().GetCurrencyCode(); got != "UAH" {
		t.Errorf("currency = %q, want the account's UAH", got)
	}
}

func TestListTransactionsGroupsByDayWithSubtotals(t *testing.T) {
	h, store, _ := newTestHandler(t)
	_, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	seedTransaction(store, account, category, sergiy, 100_00, "2026-08-29")
	seedTransaction(store, account, category, sergiy, 200_00, "2026-08-29")
	seedTransaction(store, account, category, olena, 50_00, "2026-08-28")

	resp, err := h.ListTransactions(ctxOf(sergiy), connect.NewRequest(&financev1.ListTransactionsRequest{
		Kind: financev1.TransactionKind_TRANSACTION_KIND_EXPENSE,
	}))
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if len(resp.Msg.Days) != 2 {
		t.Fatalf("days = %d, want 2", len(resp.Msg.Days))
	}
	// Newest first, and the subtotal is the sum of exactly the rows in the section.
	if resp.Msg.Days[0].Date != "2026-08-29" {
		t.Errorf("first section = %q, want 2026-08-29", resp.Msg.Days[0].Date)
	}
	if got := resp.Msg.Days[0].DayTotal.AmountMinor; got != 300_00 {
		t.Errorf("day_total = %d, want 30000", got)
	}
	if got := resp.Msg.PeriodTotal.AmountMinor; got != 350_00 {
		t.Errorf("period_total = %d, want 35000", got)
	}
}

func TestListTransactionsPagesWithACursor(t *testing.T) {
	h, store, _ := newTestHandler(t)
	_, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	for i := 0; i < 3; i++ {
		seedTransaction(store, account, category, sergiy, 100_00, "2026-08-2"+string(rune('1'+i)))
	}

	first, err := h.ListTransactions(ctxOf(sergiy), connect.NewRequest(&financev1.ListTransactionsRequest{
		PageSize: 2,
	}))
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if first.Msg.NextCursor == "" {
		t.Fatal("a full page must hand back a cursor")
	}
	second, err := h.ListTransactions(ctxOf(sergiy), connect.NewRequest(&financev1.ListTransactionsRequest{
		PageSize: 2, Cursor: first.Msg.NextCursor,
	}))
	if err != nil {
		t.Fatalf("ListTransactions page 2: %v", err)
	}
	if len(second.Msg.Days) != 1 || len(second.Msg.Days[0].Transactions) != 1 {
		t.Errorf("page 2 = %v, want the single remaining row", second.Msg.Days)
	}
	// A short page cannot have more, so it must not hand back a cursor that returns nothing.
	if second.Msg.NextCursor != "" {
		t.Errorf("next_cursor = %q, want empty on a short page", second.Msg.NextCursor)
	}
}

func TestTransferIsOneRowAndIsExcludedFromTotals(t *testing.T) {
	h, store, rec := newTestHandler(t)
	from := seedAccount(store, "Mono", "card", visibilityShared, "", 1_000_00)
	to := seedAccount(store, "Скарбничка", accountKindSavings, visibilityShared, "", 0)

	resp, err := h.TransferBetweenAccounts(ctxOf(sergiy),
		connect.NewRequest(&financev1.TransferBetweenAccountsRequest{
			FromAccountId: id(from.ID), ToAccountId: id(to.ID),
			Amount: &financev1.Money{AmountMinor: 300_00, CurrencyCode: "UAH"},
		}))
	if err != nil {
		t.Fatalf("TransferBetweenAccounts: %v", err)
	}
	if len(store.transactions) != 1 {
		t.Errorf("stored rows = %d, want 1 — a transfer is one row with a counter account", len(store.transactions))
	}
	if resp.Msg.Transaction.CounterAccountId != id(to.ID) {
		t.Errorf("counter_account_id = %q, want %q", resp.Msg.Transaction.CounterAccountId, id(to.ID))
	}
	if !rec.sawSubject("finance.transfer.created") {
		t.Errorf("subjects = %v, want finance.transfer.created", rec.subjects)
	}

	accounts, err := h.ListAccounts(ctxOf(sergiy), connect.NewRequest(&financev1.ListAccountsRequest{}))
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	balances := map[string]int64{}
	for _, a := range accounts.Msg.Shared {
		balances[a.Name] = a.Balance.AmountMinor
	}
	if balances["Mono"] != 700_00 {
		t.Errorf("Mono balance = %d, want 70000", balances["Mono"])
	}
	if balances["Скарбничка"] != 300_00 {
		t.Errorf("savings balance = %d, want 30000", balances["Скарбничка"])
	}

	feed, err := h.ListTransactions(ctxOf(sergiy), connect.NewRequest(&financev1.ListTransactionsRequest{
		Kind: financev1.TransactionKind_TRANSACTION_KIND_EXPENSE,
	}))
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if got := feed.Msg.PeriodTotal.AmountMinor; got != 0 {
		t.Errorf("period_total = %d, want 0 — a transfer is not spending", got)
	}
}

func TestTransferToTheSameAccountIsRefused(t *testing.T) {
	h, store, _ := newTestHandler(t)
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)

	_, err := h.TransferBetweenAccounts(ctxOf(sergiy),
		connect.NewRequest(&financev1.TransferBetweenAccountsRequest{
			FromAccountId: id(account.ID), ToAccountId: id(account.ID),
			Amount: &financev1.Money{AmountMinor: 1, CurrencyCode: "UAH"},
		}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", got)
	}
}

func TestCrossCurrencyTransferNeedsAReceivedAmount(t *testing.T) {
	h, store, _ := newTestHandler(t)
	from := seedAccount(store, "Mono", "card", visibilityShared, "", 100_000_00)
	to := seedAccount(store, "USD", "bank", visibilityShared, "", 0)
	to.CurrencyCode = "USD"
	store.accounts[id(to.ID)] = to

	_, err := h.TransferBetweenAccounts(ctxOf(sergiy),
		connect.NewRequest(&financev1.TransferBetweenAccountsRequest{
			FromAccountId: id(from.ID), ToAccountId: id(to.ID),
			Amount: &financev1.Money{AmountMinor: 41_000_00, CurrencyCode: "UAH"},
		}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", got)
	}

	resp, err := h.TransferBetweenAccounts(ctxOf(sergiy),
		connect.NewRequest(&financev1.TransferBetweenAccountsRequest{
			FromAccountId: id(from.ID), ToAccountId: id(to.ID),
			Amount:         &financev1.Money{AmountMinor: 41_000_00, CurrencyCode: "UAH"},
			ReceivedAmount: &financev1.Money{AmountMinor: 1_000_00, CurrencyCode: "USD"},
		}))
	if err != nil {
		t.Fatalf("TransferBetweenAccounts: %v", err)
	}
	got := resp.Msg.Transaction.ReceivedAmount
	if got.GetAmountMinor() != 1_000_00 || got.GetCurrencyCode() != "USD" {
		t.Errorf("received_amount = %v, want 100000 USD", got)
	}
}

/* ================================= budgets ================================= */

func TestBudgetStatusReportsOverspendUnclamped(t *testing.T) {
	h, store, _ := newTestHandler(t)
	group, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	store.budgets["b"] = db.Budget{
		ID:       pgconv.MustUUID("00000000-0000-4000-8000-0000000000b1"),
		FamilyID: pgconv.MustUUID(testFamily), TargetKind: targetGroup, GroupID: group.ID,
		LimitMinor: 1_000_00, CurrencyCode: "UAH", Period: budgetPeriodMonth,
		StartOn: day("2026-08-01"), NotifyOnExceed: true,
	}
	seedTransaction(store, account, category, sergiy, 2_560_00, "2026-08-15")

	resp, err := h.ListBudgets(ctxOf(sergiy), connect.NewRequest(&financev1.ListBudgetsRequest{}))
	if err != nil {
		t.Fatalf("ListBudgets: %v", err)
	}
	if len(resp.Msg.Budgets) != 1 {
		t.Fatalf("budgets = %d, want 1", len(resp.Msg.Budgets))
	}
	status := resp.Msg.Budgets[0]
	// 256%: the design clamps the bar and switches the colour, which it cannot do if the
	// server clamps the number first.
	if status.Share != 2.56 {
		t.Errorf("share = %v, want 2.56", status.Share)
	}
	if !status.Exceeded {
		t.Error("exceeded = false, want true")
	}
	if status.Remaining.AmountMinor != -1_560_00 {
		t.Errorf("remaining = %d, want -156000", status.Remaining.AmountMinor)
	}
	if resp.Msg.WithinLimitCount != 0 || resp.Msg.TotalCount != 1 {
		t.Errorf("within/total = %d/%d, want 0/1", resp.Msg.WithinLimitCount, resp.Msg.TotalCount)
	}
}

func TestCreateTransactionAnnouncesOverspendOnce(t *testing.T) {
	h, store, rec := newTestHandler(t)
	group, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	store.budgets["b"] = db.Budget{
		ID:       pgconv.MustUUID("00000000-0000-4000-8000-0000000000b1"),
		FamilyID: pgconv.MustUUID(testFamily), TargetKind: targetGroup, GroupID: group.ID,
		LimitMinor: 100_00, CurrencyCode: "UAH", Period: budgetPeriodMonth,
		StartOn: day("2026-08-01"), NotifyOnExceed: true,
	}

	spend := func(amount int64) *financev1.CreateTransactionResponse {
		t.Helper()
		resp, err := h.CreateTransaction(ctxOf(sergiy), connect.NewRequest(&financev1.CreateTransactionRequest{
			Type: financev1.TransactionType_TRANSACTION_TYPE_EXPENSE, AccountId: id(account.ID),
			CategoryId: id(category.ID),
			Amount:     &financev1.Money{AmountMinor: amount, CurrencyCode: "UAH"},
			OccurredOn: "2026-08-15",
		}))
		if err != nil {
			t.Fatalf("CreateTransaction: %v", err)
		}
		return resp.Msg
	}

	first := spend(50_00)
	if len(first.AffectedBudgets) != 1 {
		t.Fatalf("affected_budgets = %d, want 1", len(first.AffectedBudgets))
	}
	if first.AffectedBudgets[0].Exceeded {
		t.Error("50 of a 100 limit must not be exceeded")
	}
	if rec.sawSubject("finance.budget.exceeded") {
		t.Error("budget.exceeded published while still inside the limit")
	}

	second := spend(80_00)
	if !second.AffectedBudgets[0].Exceeded {
		t.Error("130 of a 100 limit must be exceeded")
	}
	exceeded := 0
	for _, s := range rec.subjects {
		if s == "finance.budget.exceeded" {
			exceeded++
		}
	}
	if exceeded != 1 {
		t.Errorf("budget.exceeded published %d times, want 1", exceeded)
	}

	// A third transaction on an already-blown budget must not re-announce: the edge is the
	// event, and republishing it is how a notification service ends up sending duplicates.
	spend(10_00)
	exceeded = 0
	for _, s := range rec.subjects {
		if s == "finance.budget.exceeded" {
			exceeded++
		}
	}
	if exceeded != 1 {
		t.Errorf("budget.exceeded published %d times after a third spend, want 1", exceeded)
	}
}

func TestDeleteTransactionRecoversABudget(t *testing.T) {
	h, store, rec := newTestHandler(t)
	group, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	store.budgets["b"] = db.Budget{
		ID:       pgconv.MustUUID("00000000-0000-4000-8000-0000000000b1"),
		FamilyID: pgconv.MustUUID(testFamily), TargetKind: targetGroup, GroupID: group.ID,
		LimitMinor: 100_00, CurrencyCode: "UAH", Period: budgetPeriodMonth,
		StartOn: day("2026-08-01"), NotifyOnExceed: true,
	}
	tx := seedTransaction(store, account, category, sergiy, 500_00, "2026-08-15")

	resp, err := h.DeleteTransaction(ctxOf(sergiy), connect.NewRequest(&financev1.DeleteTransactionRequest{
		TransactionId: id(tx.ID),
	}))
	if err != nil {
		t.Fatalf("DeleteTransaction: %v", err)
	}
	if len(resp.Msg.AffectedBudgets) != 1 || resp.Msg.AffectedBudgets[0].Exceeded {
		t.Errorf("affected_budgets = %v, want the budget back inside its limit", resp.Msg.AffectedBudgets)
	}
	if !rec.sawSubject("finance.budget.recovered") {
		t.Errorf("subjects = %v, want finance.budget.recovered", rec.subjects)
	}
}

func TestBudgetAppliesToBothItsCategoryAndItsGroup(t *testing.T) {
	h, store, _ := newTestHandler(t)
	group, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	store.budgets["group"] = db.Budget{
		ID:       pgconv.MustUUID("00000000-0000-4000-8000-0000000000b1"),
		FamilyID: pgconv.MustUUID(testFamily), TargetKind: targetGroup, GroupID: group.ID,
		LimitMinor: 1_000_00, CurrencyCode: "UAH", Period: budgetPeriodMonth,
		StartOn: day("2026-08-01"),
	}
	store.budgets["category"] = db.Budget{
		ID:       pgconv.MustUUID("00000000-0000-4000-8000-0000000000b2"),
		FamilyID: pgconv.MustUUID(testFamily), TargetKind: targetCategory, CategoryID: category.ID,
		LimitMinor: 200_00, CurrencyCode: "UAH", Period: budgetPeriodMonth,
		StartOn: day("2026-08-01"),
	}

	resp, err := h.CreateTransaction(ctxOf(sergiy), connect.NewRequest(&financev1.CreateTransactionRequest{
		Type: financev1.TransactionType_TRANSACTION_TYPE_EXPENSE, AccountId: id(account.ID),
		CategoryId: id(category.ID),
		Amount:     &financev1.Money{AmountMinor: 300_00, CurrencyCode: "UAH"},
		OccurredOn: "2026-08-15",
	}))
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}
	// Spend in a category counts toward the category's budget and its group's, which is why
	// both bars have to repaint from one response.
	if len(resp.Msg.AffectedBudgets) != 2 {
		t.Fatalf("affected_budgets = %d, want 2", len(resp.Msg.AffectedBudgets))
	}
	exceeded := 0
	for _, s := range resp.Msg.AffectedBudgets {
		if s.Exceeded {
			exceeded++
		}
	}
	if exceeded != 1 {
		t.Errorf("exceeded budgets = %d, want 1 (the category's, not the group's)", exceeded)
	}
}

func TestCreateBudgetNeedsExactlyOneTarget(t *testing.T) {
	h, _, _ := newTestHandler(t)

	_, err := h.CreateBudget(ctxOf(sergiy), connect.NewRequest(&financev1.CreateBudgetRequest{
		Limit: &financev1.Money{AmountMinor: 100_00, CurrencyCode: "UAH"},
	}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", got)
	}
}

/* =============================== categories ================================ */

func TestDeleteCategoryGroupNeedsAReassignmentTarget(t *testing.T) {
	h, store, _ := newTestHandler(t)
	group, _ := seedGroupAndCategory(store, "Їжа", "Продукти")

	_, err := h.DeleteCategoryGroup(ctxOf(sergiy), connect.NewRequest(&financev1.DeleteCategoryGroupRequest{
		GroupId: id(group.ID),
	}))
	if got := codeOf(t, err); got != connect.CodeFailedPrecondition {
		t.Fatalf("code = %v, want FailedPrecondition", got)
	}

	other, _ := seedGroupAndCategory(store, "Розваги", "Кіно")
	resp, err := h.DeleteCategoryGroup(ctxOf(sergiy), connect.NewRequest(&financev1.DeleteCategoryGroupRequest{
		GroupId: id(group.ID), ReassignToGroupId: id(other.ID),
	}))
	if err != nil {
		t.Fatalf("DeleteCategoryGroup: %v", err)
	}
	if resp.Msg.MovedCategories != 1 {
		t.Errorf("moved_categories = %d, want 1", resp.Msg.MovedCategories)
	}
	if _, ok := store.groups[id(group.ID)]; ok {
		t.Error("group was not deleted")
	}
}

func TestDeleteCategoryReassignsItsTransactions(t *testing.T) {
	h, store, _ := newTestHandler(t)
	group, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	tx := seedTransaction(store, account, category, sergiy, 100_00, "2026-08-15")

	other := db.Category{
		ID: store.newUUID(), FamilyID: pgconv.MustUUID(testFamily), GroupID: group.ID,
		Name: "Кафе", Kind: kindExpense,
	}
	store.categories[id(other.ID)] = other

	resp, err := h.DeleteCategory(ctxOf(sergiy), connect.NewRequest(&financev1.DeleteCategoryRequest{
		CategoryId: id(category.ID), ReassignToCategoryId: id(other.ID),
	}))
	if err != nil {
		t.Fatalf("DeleteCategory: %v", err)
	}
	if resp.Msg.MovedTransactions != 1 {
		t.Errorf("moved_transactions = %d, want 1", resp.Msg.MovedTransactions)
	}
	if got := store.transactions[id(tx.ID)].CategoryID; !same(got, other.ID) {
		t.Error("the transaction kept a category that no longer exists")
	}
}

func TestCategoryMustMatchItsGroupsKind(t *testing.T) {
	h, store, _ := newTestHandler(t)
	group, _ := seedGroupAndCategory(store, "Їжа", "Продукти")

	_, err := h.CreateCategory(ctxOf(sergiy), connect.NewRequest(&financev1.CreateCategoryRequest{
		GroupId: id(group.ID), Name: "Зарплата",
		Kind: financev1.TransactionKind_TRANSACTION_KIND_INCOME,
	}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", got)
	}
}

/* ================================ templates ================================ */

func TestTemplatesArePrivateToTheirOwner(t *testing.T) {
	h, store, _ := newTestHandler(t)
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	_, err := h.CreateTemplate(ctxOf(sergiy), connect.NewRequest(&financev1.CreateTemplateRequest{
		Label: "Кава", AccountId: id(account.ID),
		Amount: &financev1.Money{AmountMinor: 50_00, CurrencyCode: "UAH"},
	}))
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	// Олена sees none of Сергій's templates, and asking for them is an empty list rather than
	// an error: an error would confirm that he has some.
	hers, err := h.ListTemplates(ctxOf(olena), connect.NewRequest(&financev1.ListTemplatesRequest{
		OwnerUserId: sergiy,
	}))
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(hers.Msg.Templates) != 0 {
		t.Errorf("templates = %d, want 0", len(hers.Msg.Templates))
	}
}

func TestLogTemplateWritesATransactionAndBumpsUsage(t *testing.T) {
	h, store, rec := newTestHandler(t)
	_, category := seedGroupAndCategory(store, "Їжа", "Кафе")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	created, err := h.CreateTemplate(ctxOf(sergiy), connect.NewRequest(&financev1.CreateTemplateRequest{
		Label: "Кава", AccountId: id(account.ID), CategoryId: id(category.ID),
		Type:   financev1.TransactionType_TRANSACTION_TYPE_EXPENSE,
		Amount: &financev1.Money{AmountMinor: 50_00, CurrencyCode: "UAH"},
	}))
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	resp, err := h.LogTemplate(ctxOf(sergiy), connect.NewRequest(&financev1.LogTemplateRequest{
		TemplateId: created.Msg.Template.Id,
	}))
	if err != nil {
		t.Fatalf("LogTemplate: %v", err)
	}
	if resp.Msg.Transaction.Amount.AmountMinor != 50_00 {
		t.Errorf("amount = %d, want 5000", resp.Msg.Transaction.Amount.AmountMinor)
	}
	// The provenance badge on the feed row depends on this being set.
	if resp.Msg.Transaction.TemplateId != created.Msg.Template.Id {
		t.Errorf("template_id = %q, want the template's id", resp.Msg.Transaction.TemplateId)
	}
	if got := store.templates[created.Msg.Template.Id].UsageCount; got != 1 {
		t.Errorf("usage_count = %d, want 1", got)
	}
	if !rec.sawSubject("finance.template.used") {
		t.Errorf("subjects = %v, want finance.template.used", rec.subjects)
	}
}

func TestLogTemplateHonoursAnAmountOverride(t *testing.T) {
	h, store, _ := newTestHandler(t)
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	created, err := h.CreateTemplate(ctxOf(sergiy), connect.NewRequest(&financev1.CreateTemplateRequest{
		Label: "Кава", AccountId: id(account.ID),
		Amount: &financev1.Money{AmountMinor: 50_00, CurrencyCode: "UAH"},
	}))
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	resp, err := h.LogTemplate(ctxOf(sergiy), connect.NewRequest(&financev1.LogTemplateRequest{
		TemplateId:     created.Msg.Template.Id,
		AmountOverride: &financev1.Money{AmountMinor: 75_00, CurrencyCode: "UAH"},
	}))
	if err != nil {
		t.Fatalf("LogTemplate: %v", err)
	}
	if resp.Msg.Transaction.Amount.AmountMinor != 75_00 {
		t.Errorf("amount = %d, want 7500", resp.Msg.Transaction.Amount.AmountMinor)
	}
}

/* ================================ accounts ================================= */

func TestDeleteAccountWithHistoryIsRefused(t *testing.T) {
	h, store, _ := newTestHandler(t)
	_, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	seedTransaction(store, account, category, sergiy, 100_00, "2026-08-15")

	_, err := h.DeleteAccount(ctxOf(sergiy), connect.NewRequest(&financev1.DeleteAccountRequest{
		AccountId: id(account.ID),
	}))
	// A balance that silently loses its rows is a corrupt ledger.
	if got := codeOf(t, err); got != connect.CodeFailedPrecondition {
		t.Errorf("code = %v, want FailedPrecondition", got)
	}
}

func TestReorderAccountsAssignsPositionsInOrder(t *testing.T) {
	h, store, _ := newTestHandler(t)
	first := seedAccount(store, "Готівка", "cash", visibilityShared, "", 0)
	second := seedAccount(store, "Mono", "card", visibilityShared, "", 0)

	_, err := h.ReorderAccounts(ctxOf(sergiy), connect.NewRequest(&financev1.ReorderAccountsRequest{
		AccountIdsInOrder: []string{id(second.ID), id(first.ID)},
	}))
	if err != nil {
		t.Fatalf("ReorderAccounts: %v", err)
	}
	if store.accounts[id(second.ID)].SortOrder != 0 || store.accounts[id(first.ID)].SortOrder != 1 {
		t.Error("sort order does not follow the requested order")
	}
}

/* =============================== settings ================================== */

func TestBootstrapHouseholdIsIdempotent(t *testing.T) {
	store := newFakeStore()
	h := New(Options{Queries: store, Tx: store, Now: func() time.Time { return testNow }})

	req := connect.NewRequest(&financev1.BootstrapHouseholdRequest{
		BaseCurrencyCode: "UAH", Timezone: "Europe/Kyiv", SeedDefaultTaxonomy: true,
	})
	first, err := h.BootstrapHousehold(ctxOf(sergiy), req)
	if err != nil {
		t.Fatalf("BootstrapHousehold: %v", err)
	}
	if len(first.Msg.Groups) == 0 || len(first.Msg.Categories) == 0 {
		t.Fatal("the default taxonomy was not seeded")
	}
	groupsAfterFirst := len(store.groups)

	second, err := h.BootstrapHousehold(ctxOf(sergiy), req)
	if err != nil {
		t.Fatalf("BootstrapHousehold (again): %v", err)
	}
	if len(store.groups) != groupsAfterFirst {
		t.Errorf("groups = %d after a second bootstrap, want %d", len(store.groups), groupsAfterFirst)
	}
	if len(second.Msg.Groups) != groupsAfterFirst {
		t.Errorf("response groups = %d, want the existing %d", len(second.Msg.Groups), groupsAfterFirst)
	}
}

func TestReadsBeforeBootstrapAreAPrecondition(t *testing.T) {
	store := newFakeStore()
	h := New(Options{Queries: store, Tx: store, Now: func() time.Time { return testNow }})

	_, err := h.ListAccounts(ctxOf(sergiy), connect.NewRequest(&financev1.ListAccountsRequest{}))
	if got := codeOf(t, err); got != connect.CodeFailedPrecondition {
		t.Errorf("code = %v, want FailedPrecondition", got)
	}
}

func TestUpdateFinanceSettingsRejectsABadTimezone(t *testing.T) {
	h, _, _ := newTestHandler(t)
	tz := "Mars/Olympus_Mons"

	_, err := h.UpdateFinanceSettings(ctxOf(sergiy),
		connect.NewRequest(&financev1.UpdateFinanceSettingsRequest{Timezone: &tz}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", got)
	}
}

func TestSetOverspendNotifications(t *testing.T) {
	h, _, _ := newTestHandler(t)

	resp, err := h.SetOverspendNotifications(ctxOf(sergiy),
		connect.NewRequest(&financev1.SetOverspendNotificationsRequest{Enabled: false}))
	if err != nil {
		t.Fatalf("SetOverspendNotifications: %v", err)
	}
	if resp.Msg.Enabled {
		t.Error("enabled = true, want false")
	}
}

/* ============================= internal errors ============================= */

func TestInfrastructureFailureIsOpaque(t *testing.T) {
	h, store, _ := newTestHandler(t)
	store.failOn["ListVisibleAccounts"] = errBoom

	_, err := h.ListAccounts(ctxOf(sergiy), connect.NewRequest(&financev1.ListAccountsRequest{}))
	if got := codeOf(t, err); got != connect.CodeInternal {
		t.Fatalf("code = %v, want Internal", got)
	}
	// The pgx error must not reach the caller — only the reference that finds the log line.
	if msg := err.Error(); contains(msg, "boom") {
		t.Errorf("error %q leaks the underlying failure", msg)
	}
}

func contains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		(haystack == needle || indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

/* ================================= widgets ================================= */

func TestGetWidgetDataAppliesTheVisibilityBoundary(t *testing.T) {
	h, store, _ := newTestHandler(t)
	seedAccount(store, "Mono", "card", visibilityShared, "", 1_000_00)
	hers := seedAccount(store, "Олена USD", "bank", visibilityPrivate, olena, 999_00)

	added, err := h.AddWidget(ctxOf(sergiy), connect.NewRequest(&financev1.AddWidgetRequest{
		Type: financev1.WidgetType_WIDGET_TYPE_ACCOUNTS,
		Size: financev1.WidgetSize_WIDGET_SIZE_4X1,
	}))
	if err != nil {
		t.Fatalf("AddWidget: %v", err)
	}
	resp, err := h.GetWidgetData(ctxOf(sergiy), connect.NewRequest(&financev1.GetWidgetDataRequest{
		WidgetIds: []string{added.Msg.Widget.Id},
	}))
	if err != nil {
		t.Fatalf("GetWidgetData: %v", err)
	}
	if len(resp.Msg.Payloads) != 1 {
		t.Fatalf("payloads = %d, want 1", len(resp.Msg.Payloads))
	}
	// A home-screen widget is not a reason to skip the boundary: a locked phone showing
	// another member's private balance is the same leak with a smaller font.
	for _, a := range resp.Msg.Payloads[0].GetAccounts().GetAccounts() {
		if a.Id == id(hers.ID) {
			t.Fatal("the accounts widget carried another member's private account")
		}
	}
}

func TestWidgetsAreScopedToTheirPlacingUser(t *testing.T) {
	h, _, _ := newTestHandler(t)
	if _, err := h.AddWidget(ctxOf(sergiy), connect.NewRequest(&financev1.AddWidgetRequest{
		Type: financev1.WidgetType_WIDGET_TYPE_MONTH,
	})); err != nil {
		t.Fatalf("AddWidget: %v", err)
	}

	hers, err := h.ListWidgets(ctxOf(olena), connect.NewRequest(&financev1.ListWidgetsRequest{}))
	if err != nil {
		t.Fatalf("ListWidgets: %v", err)
	}
	if len(hers.Msg.Widgets) != 0 {
		t.Errorf("widgets = %d, want 0 — placements are per-user", len(hers.Msg.Widgets))
	}
}

/* ================================ recurring ================================ */

func TestPostRecurringOccurrenceIsIdempotentPerDueDate(t *testing.T) {
	h, store, rec := newTestHandler(t)
	_, category := seedGroupAndCategory(store, "Дім", "Оренда")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	payment := db.RecurringPayment{
		ID: store.newUUID(), FamilyID: pgconv.MustUUID(testFamily), Name: "Оренда",
		AmountMinor: 10_000_00, CurrencyCode: "UAH", Type: kindExpense,
		CategoryID: category.ID, AccountID: account.ID, MemberID: pgconv.MustUUID(sergiy),
		IntervalCount: 1, IntervalUnit: "month", DayOfMonth: 5,
		NextDueOn: day("2026-08-05"), Active: true,
	}
	store.recurring[id(payment.ID)] = payment

	first, err := h.PostRecurringOccurrence(ctxOf(sergiy),
		connect.NewRequest(&financev1.PostRecurringOccurrenceRequest{
			RecurringId: id(payment.ID), DueOn: "2026-08-05",
		}))
	if err != nil {
		t.Fatalf("PostRecurringOccurrence: %v", err)
	}
	if first.Msg.NextDueOn != "2026-09-05" {
		t.Errorf("next_due_on = %q, want 2026-09-05", first.Msg.NextDueOn)
	}
	if len(store.transactions) != 1 {
		t.Fatalf("transactions = %d, want 1", len(store.transactions))
	}
	if !rec.sawSubject("finance.recurring.posted") {
		t.Errorf("subjects = %v, want finance.recurring.posted", rec.subjects)
	}

	// A widget that retried on a flaky connection must not invent a second rent payment.
	if _, err := h.PostRecurringOccurrence(ctxOf(sergiy),
		connect.NewRequest(&financev1.PostRecurringOccurrenceRequest{
			RecurringId: id(payment.ID), DueOn: "2026-08-05",
		})); err != nil {
		t.Fatalf("PostRecurringOccurrence (again): %v", err)
	}
	if len(store.transactions) != 1 {
		t.Errorf("transactions = %d after a repeat post, want 1", len(store.transactions))
	}
}

func TestSkipRecurringOccurrenceWritesNoTransaction(t *testing.T) {
	h, store, _ := newTestHandler(t)
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	payment := db.RecurringPayment{
		ID: store.newUUID(), FamilyID: pgconv.MustUUID(testFamily), Name: "Netflix",
		AmountMinor: 300_00, CurrencyCode: "UAH", Type: kindExpense, AccountID: account.ID,
		MemberID: pgconv.MustUUID(sergiy), IntervalCount: 1, IntervalUnit: "month",
		NextDueOn: day("2026-08-05"), Active: true,
	}
	store.recurring[id(payment.ID)] = payment

	resp, err := h.SkipRecurringOccurrence(ctxOf(sergiy),
		connect.NewRequest(&financev1.SkipRecurringOccurrenceRequest{
			RecurringId: id(payment.ID), DueOn: "2026-08-05",
		}))
	if err != nil {
		t.Fatalf("SkipRecurringOccurrence: %v", err)
	}
	if resp.Msg.NextDueOn != "2026-09-05" {
		t.Errorf("next_due_on = %q, want 2026-09-05", resp.Msg.NextDueOn)
	}
	if len(store.transactions) != 0 {
		t.Errorf("transactions = %d, want 0 — a skipped payment did not happen", len(store.transactions))
	}
}

// seedRecurring writes a schedule straight into the store, bypassing the handler, so a test
// can set up one attached to an account the caller may not see.
func seedRecurring(s *fakeStore, name string, account db.Account, amount int64) db.RecurringPayment {
	r := db.RecurringPayment{
		ID: s.newUUID(), FamilyID: pgconv.MustUUID(testFamily), Name: name,
		AmountMinor: amount, CurrencyCode: "UAH", Type: kindExpense, AccountID: account.ID,
		MemberID: account.OwnerMemberID, IntervalCount: 1, IntervalUnit: "month",
		NextDueOn: day("2026-08-05"), Active: true,
	}
	s.recurring[id(r.ID)] = r
	return r
}

// A schedule inherits the visibility of the account it is attached to: its name, amount and
// cadence describe a private account as plainly as a transaction does.
func TestListRecurringPaymentsHidesSchedulesOnAnotherMembersPrivateAccount(t *testing.T) {
	h, store, _ := newTestHandler(t)
	shared := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	mine := seedAccount(store, "Мій гаманець", "cash", visibilityPrivate, sergiy, 0)
	hers := seedAccount(store, "Олена USD", "bank", visibilityPrivate, olena, 0)
	rent := seedRecurring(store, "Оренда", shared, 10_000_00)
	mySub := seedRecurring(store, "Spotify", mine, 200_00)
	herSub := seedRecurring(store, "Психотерапевт", hers, 2_500_00)

	resp, err := h.ListRecurringPayments(ctxOf(sergiy),
		connect.NewRequest(&financev1.ListRecurringPaymentsRequest{IncludeInactive: true}))
	if err != nil {
		t.Fatalf("ListRecurringPayments: %v", err)
	}
	got := map[string]bool{}
	for _, p := range resp.Msg.Payments {
		got[p.GetPayment().GetId()] = true
	}
	if !got[id(rent.ID)] || !got[id(mySub.ID)] {
		t.Errorf("payments = %v, want the shared and the caller's own private schedule", got)
	}
	if got[id(herSub.ID)] {
		t.Fatal("a schedule on another member's private account was serialised in full")
	}
}

// Every write path answers NotFound for a schedule the caller may not read — the same answer
// as an id that never existed, so none of them can be used as an existence oracle either.
func TestRecurringWritesOnAnotherMembersPrivateAccountAreNotFound(t *testing.T) {
	cases := []struct {
		name string
		call func(*Handler, db.RecurringPayment) error
	}{
		{"update", func(h *Handler, r db.RecurringPayment) error {
			active := true
			_, err := h.UpdateRecurringPayment(ctxOf(sergiy),
				connect.NewRequest(&financev1.UpdateRecurringPaymentRequest{
					RecurringId: id(r.ID), Active: &active,
				}))
			return err
		}},
		{"delete", func(h *Handler, r db.RecurringPayment) error {
			_, err := h.DeleteRecurringPayment(ctxOf(sergiy),
				connect.NewRequest(&financev1.DeleteRecurringPaymentRequest{RecurringId: id(r.ID)}))
			return err
		}},
		{"post", func(h *Handler, r db.RecurringPayment) error {
			_, err := h.PostRecurringOccurrence(ctxOf(sergiy),
				connect.NewRequest(&financev1.PostRecurringOccurrenceRequest{
					RecurringId: id(r.ID), DueOn: "2026-08-05",
				}))
			return err
		}},
		{"skip", func(h *Handler, r db.RecurringPayment) error {
			_, err := h.SkipRecurringOccurrence(ctxOf(sergiy),
				connect.NewRequest(&financev1.SkipRecurringOccurrenceRequest{
					RecurringId: id(r.ID), DueOn: "2026-08-05",
				}))
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, store, _ := newTestHandler(t)
			hers := seedAccount(store, "Олена USD", "bank", visibilityPrivate, olena, 0)
			payment := seedRecurring(store, "Психотерапевт", hers, 2_500_00)

			if got := codeOf(t, tc.call(h, payment)); got != connect.CodeNotFound {
				t.Fatalf("code = %v, want NotFound", got)
			}
			after, ok := store.recurring[id(payment.ID)]
			if !ok {
				t.Fatal("another member's private-account schedule was deleted")
			}
			if after.NextDueOn != payment.NextDueOn || after.AccountID != payment.AccountID {
				t.Errorf("schedule = %+v, want it untouched", after)
			}
			if len(store.transactions) != 0 {
				t.Errorf("transactions = %d, want 0", len(store.transactions))
			}
		})
	}
}

/* =============================== analytics ================================= */

func TestGetHouseholdOverviewCountsPrivateAccountsWithoutRevealingThem(t *testing.T) {
	h, store, _ := newTestHandler(t)
	_, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	shared := seedAccount(store, "Mono", "card", visibilityShared, "", 17_634_00)
	seedAccount(store, "Скарбничка", accountKindSavings, visibilityShared, "", 145_503_00)
	seedAccount(store, "Мій гаманець", "cash", visibilityPrivate, sergiy, 500_00)
	seedAccount(store, "Олена USD", "bank", visibilityPrivate, olena, 142_000_00)
	seedTransaction(store, shared, category, sergiy, 300_00, "2026-08-10")
	seedTransaction(store, shared, category, olena, 100_00, "2026-08-11")

	resp, err := h.GetHouseholdOverview(ctxOf(sergiy),
		connect.NewRequest(&financev1.GetHouseholdOverviewRequest{}))
	if err != nil {
		t.Fatalf("GetHouseholdOverview: %v", err)
	}
	// The opening balance less the two shared expenses: the balance is derived on every read,
	// never stored, so a spend moves the headline without anything writing to the account row.
	if got := resp.Msg.SharedBalance.AmountMinor; got != 17_234_00 {
		t.Errorf("shared_balance = %d, want 1723400 — private and savings are not in it", got)
	}
	if got := resp.Msg.SavingsTotal.AmountMinor; got != 145_503_00 {
		t.Errorf("savings_total = %d, want 14550300", got)
	}
	if got := resp.Msg.PeriodExpense.AmountMinor; got != 400_00 {
		t.Errorf("period_expense = %d, want 40000", got)
	}

	counts := map[string]int32{}
	spent := map[string]int64{}
	for _, m := range resp.Msg.Members {
		counts[m.GetMember().GetUserId()] = m.PrivateAccountCount
		spent[m.GetMember().GetUserId()] = m.GetSpent().GetAmountMinor()
	}
	// Both cards carry a count. The caller's comes from their own list, the other member's from
	// the hidden summary — and in neither case does a balance travel with it.
	if counts[sergiy] != 1 || counts[olena] != 1 {
		t.Errorf("private counts = %v, want one each", counts)
	}
	if spent[sergiy] != 300_00 || spent[olena] != 100_00 {
		t.Errorf("member spend = %v, want 30000/10000", spent)
	}
}

func TestListCategoryTreeCarriesEachGroupsBudget(t *testing.T) {
	h, store, _ := newTestHandler(t)
	group, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	seedGroupAndCategory(store, "Розваги", "Кіно")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	store.budgets["b"] = db.Budget{
		ID:       pgconv.MustUUID("00000000-0000-4000-8000-0000000000b1"),
		FamilyID: pgconv.MustUUID(testFamily), TargetKind: targetGroup, GroupID: group.ID,
		LimitMinor: 6_000_00, CurrencyCode: "UAH", Period: budgetPeriodMonth,
		StartOn: day("2026-08-01"),
	}
	seedTransaction(store, account, category, sergiy, 1_500_00, "2026-08-10")

	resp, err := h.ListCategoryTree(ctxOf(sergiy),
		connect.NewRequest(&financev1.ListCategoryTreeRequest{
			Kind: financev1.TransactionKind_TRANSACTION_KIND_EXPENSE,
		}))
	if err != nil {
		t.Fatalf("ListCategoryTree: %v", err)
	}
	if len(resp.Msg.Groups) != 2 {
		t.Fatalf("groups = %d, want 2", len(resp.Msg.Groups))
	}
	byName := map[string]*financev1.GroupNode{}
	for _, node := range resp.Msg.Groups {
		byName[node.GetGroup().GetName()] = node
	}
	// "бюджет ₴6,000" on one row and "без бюджету" on the other: the absent budget is a nil
	// message, not a zeroed one, so the app can tell the two rows apart.
	food := byName["Їжа"]
	if food.GetBudget() == nil {
		t.Fatal("the budgeted group carries no budget")
	}
	if got := food.GetBudget().GetSpent().GetAmountMinor(); got != 1_500_00 {
		t.Errorf("spent = %d, want 150000", got)
	}
	if food.CategoryCount != 1 {
		t.Errorf("category_count = %d, want 1", food.CategoryCount)
	}
	if byName["Розваги"].GetBudget() != nil {
		t.Error("the unbudgeted group must carry no budget")
	}
}

func TestGetSpendingSeriesBucketsAndStacks(t *testing.T) {
	h, store, _ := newTestHandler(t)
	_, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	seedTransaction(store, account, category, sergiy, 100_00, "2026-08-10")
	seedTransaction(store, account, category, olena, 200_00, "2026-08-11")
	seedTransaction(store, account, category, sergiy, 50_00, "2026-07-10")

	resp, err := h.GetSpendingSeries(ctxOf(sergiy),
		connect.NewRequest(&financev1.GetSpendingSeriesRequest{
			Granularity: financev1.PeriodGranularity_PERIOD_GRANULARITY_MONTH,
			BucketCount: 7,
			Kind:        financev1.TransactionKind_TRANSACTION_KIND_EXPENSE,
			StackedBy:   financev1.SeriesStacking_SERIES_STACKING_MEMBER,
		}))
	if err != nil {
		t.Fatalf("GetSpendingSeries: %v", err)
	}
	if len(resp.Msg.Buckets) != 7 {
		t.Fatalf("buckets = %d, want 7", len(resp.Msg.Buckets))
	}
	if resp.Msg.CurrentBucketIndex != 6 {
		t.Errorf("current_bucket_index = %d, want 6", resp.Msg.CurrentBucketIndex)
	}
	current := resp.Msg.Buckets[6]
	if got := current.Total.AmountMinor; got != 300_00 {
		t.Errorf("current bucket total = %d, want 30000", got)
	}
	if len(current.Segments) != 2 {
		t.Fatalf("segments = %d, want 2 (one per spending member)", len(current.Segments))
	}
	// The legend labels come with the series so a widget does not need the member list.
	for _, seg := range current.Segments {
		if seg.Label == "" {
			t.Errorf("segment %q has no label", seg.Key)
		}
	}
	if got := resp.Msg.Buckets[5].Total.AmountMinor; got != 50_00 {
		t.Errorf("previous bucket total = %d, want 5000", got)
	}
}

func TestGetSpendingSeriesRejectsACustomGranularity(t *testing.T) {
	h, _, _ := newTestHandler(t)

	_, err := h.GetSpendingSeries(ctxOf(sergiy),
		connect.NewRequest(&financev1.GetSpendingSeriesRequest{
			Granularity: financev1.PeriodGranularity_PERIOD_GRANULARITY_CUSTOM,
		}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", got)
	}
}

func TestGetMemberBreakdownSplitsEachGroupByMember(t *testing.T) {
	h, store, _ := newTestHandler(t)
	group, category := seedGroupAndCategory(store, "Розваги", "Кіно")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	seedTransaction(store, account, category, sergiy, 1_690_00, "2026-08-10")
	seedTransaction(store, account, category, olena, 3_410_00, "2026-08-11")

	resp, err := h.GetMemberBreakdown(ctxOf(sergiy),
		connect.NewRequest(&financev1.GetMemberBreakdownRequest{
			Kind: financev1.TransactionKind_TRANSACTION_KIND_EXPENSE,
		}))
	if err != nil {
		t.Fatalf("GetMemberBreakdown: %v", err)
	}
	if got := resp.Msg.Total.AmountMinor; got != 5_100_00 {
		t.Errorf("total = %d, want 510000", got)
	}
	if len(resp.Msg.Groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(resp.Msg.Groups))
	}
	split := resp.Msg.Groups[0]
	if split.GroupId != id(group.ID) || len(split.Members) != 2 {
		t.Fatalf("split = %+v, want the group with both members", split)
	}
	amounts := map[string]int64{}
	for _, m := range split.Members {
		amounts[m.MemberId] = m.Amount.AmountMinor
	}
	if amounts[sergiy] != 1_690_00 || amounts[olena] != 3_410_00 {
		t.Errorf("per-member amounts = %v, want 169000/341000", amounts)
	}

	// share is a ratio of the household total, so the split bar's two segments add up to 1.
	var shares float64
	for _, m := range resp.Msg.Members {
		shares += m.Share
	}
	if shares < 0.999 || shares > 1.001 {
		t.Errorf("member shares sum to %v, want 1", shares)
	}
}

// A category grid is reordered inside one group. An id belonging to another group is refused
// rather than renumbered, which would silently reshuffle a grid nobody was looking at.
func TestReorderCategoriesRefusesAnotherGroupsCategory(t *testing.T) {
	h, store, _ := newTestHandler(t)
	groupA, catA := seedGroupAndCategory(store, "Їжа", "Продукти")
	_, catB := seedGroupAndCategory(store, "Дім", "Оренда")

	_, err := h.ReorderCategories(ctxOf(sergiy), connect.NewRequest(&financev1.ReorderCategoriesRequest{
		GroupId:            id(groupA.ID),
		CategoryIdsInOrder: []string{id(catA.ID), id(catB.ID)},
	}))
	if got := codeOf(t, err); got != connect.CodeNotFound {
		t.Errorf("code = %v, want NotFound", got)
	}
}

// The roster is projected from family's events, and with no broker none arrive. A caller who
// has no member row still has to appear in the switcher, so the first read makes one from the
// only human-readable fact in the token.
func TestListMembersGivesTheCallerARow(t *testing.T) {
	h, store, _ := newTestHandler(t)
	newcomer := "00000000-0000-4000-8000-00000000000a"
	ctx := fmauth.WithClaims(context.Background(), &fmauth.Claims{
		UserID: newcomer, FamilyID: testFamily, Email: "taras@example.com",
	})

	resp, err := h.ListMembers(ctx, connect.NewRequest(&financev1.ListMembersRequest{}))
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	var found *financev1.Member
	for _, m := range resp.Msg.Members {
		if m.UserId == newcomer {
			found = m
		}
	}
	if found == nil {
		t.Fatalf("the caller is missing from the roster: %v", resp.Msg.Members)
	}
	if found.DisplayName != "taras" || found.Initial != "T" {
		t.Errorf("name = %q/%q, want taras/T", found.DisplayName, found.Initial)
	}
	if _, ok := store.members[memberKey(pgconv.MustUUID(testFamily), pgconv.MustUUID(newcomer))]; !ok {
		t.Error("the row was not persisted")
	}
	// A second read must not write a second row or reset the first.
	if _, err := h.ListMembers(ctx, connect.NewRequest(&financev1.ListMembersRequest{})); err != nil {
		t.Fatalf("ListMembers (again): %v", err)
	}
}

// A category belongs to one side of the taxonomy. An income logged under an expense category
// would be invisible in both trees while still moving a balance, so it is refused.
func TestCreateTransactionRefusesACategoryOfTheOtherKind(t *testing.T) {
	h, store, _ := newTestHandler(t)
	_, expenseCategory := seedGroupAndCategory(store, "Їжа", "Продукти")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)

	_, err := h.CreateTransaction(ctxOf(sergiy), connect.NewRequest(&financev1.CreateTransactionRequest{
		Type: financev1.TransactionType_TRANSACTION_TYPE_INCOME, AccountId: id(account.ID),
		CategoryId: id(expenseCategory.ID),
		Amount:     &financev1.Money{AmountMinor: 50_00, CurrencyCode: "UAH"},
	}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", got)
	}
}

// A currency code is uppercased at the door, because every aggregate compares it byte for byte:
// a "uah" account in a UAH household would otherwise be shown in the feed and in no total.
func TestCreateAccountNormalisesTheCurrencyCase(t *testing.T) {
	h, _, _ := newTestHandler(t)

	resp, err := h.CreateAccount(ctxOf(sergiy), connect.NewRequest(&financev1.CreateAccountRequest{
		Name: "Mono", Kind: financev1.AccountKind_ACCOUNT_KIND_CARD, CurrencyCode: "uah",
	}))
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if got := resp.Msg.Account.GetCurrencyCode(); got != "UAH" {
		t.Errorf("currency = %q, want UAH", got)
	}
}

// An opening balance is money like any other, so it cannot arrive in a currency the account
// does not hold — it feeds the shared headline directly.
func TestCreateAccountRefusesAForeignOpeningBalance(t *testing.T) {
	h, _, _ := newTestHandler(t)

	_, err := h.CreateAccount(ctxOf(sergiy), connect.NewRequest(&financev1.CreateAccountRequest{
		Name: "Mono", Kind: financev1.AccountKind_ACCOUNT_KIND_CARD, CurrencyCode: "UAH",
		OpeningBalance: &financev1.Money{AmountMinor: 1000_00, CurrencyCode: "USD"},
	}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", got)
	}
}

// The header total carries every predicate the page carries. Filtering the feed by group must
// move the total with it, or the number above the list describes a wider set than the list.
func TestListTransactionsTotalFollowsTheGroupFilter(t *testing.T) {
	h, store, _ := newTestHandler(t)
	food, groceries := seedGroupAndCategory(store, "Їжа", "Продукти")
	_, rent := seedGroupAndCategory(store, "Дім", "Оренда")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	seedTransaction(store, account, groceries, sergiy, 100_00, "2026-08-29")
	seedTransaction(store, account, rent, sergiy, 900_00, "2026-08-29")

	resp, err := h.ListTransactions(ctxOf(sergiy), connect.NewRequest(&financev1.ListTransactionsRequest{
		Kind: financev1.TransactionKind_TRANSACTION_KIND_EXPENSE, GroupIds: []string{id(food.ID)},
	}))
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if got := resp.Msg.PeriodTotal.GetAmountMinor(); got != 100_00 {
		t.Errorf("period total = %d, want only the filtered group's 10000", got)
	}
}

// A budget is spent against in the household's own currency, so one kept in another currency
// could never be exceeded and could never be corrected — UpdateBudget has no currency field.
func TestCreateBudgetRefusesAForeignCurrency(t *testing.T) {
	h, store, _ := newTestHandler(t)
	group, _ := seedGroupAndCategory(store, "Їжа", "Продукти")

	_, err := h.CreateBudget(ctxOf(sergiy), connect.NewRequest(&financev1.CreateBudgetRequest{
		Target: &financev1.CreateBudgetRequest_GroupId{GroupId: id(group.ID)},
		Limit:  &financev1.Money{AmountMinor: 5000_00, CurrencyCode: "USD"},
		Period: financev1.BudgetPeriod_BUDGET_PERIOD_MONTH,
	}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", got)
	}
}

// A template moved to an account in another currency has to bring a new amount, and the row's
// currency has to move with it — otherwise every tap of its chip books the new amount under the
// old code and it lands in a total it does not belong to.
func TestUpdateTemplateFollowsItsAccountCurrency(t *testing.T) {
	h, store, _ := newTestHandler(t)
	_, category := seedGroupAndCategory(store, "Їжа", "Кава")
	uah := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	usd := seedAccount(store, "Wise", "card", visibilityShared, "", 0)
	row := store.accounts[id(usd.ID)]
	row.CurrencyCode = "USD"
	store.accounts[id(usd.ID)] = row

	created, err := h.CreateTemplate(ctxOf(sergiy), connect.NewRequest(&financev1.CreateTemplateRequest{
		Label: "Кава", Type: financev1.TransactionType_TRANSACTION_TYPE_EXPENSE,
		AccountId: id(uah.ID), CategoryId: id(category.ID),
		Amount: &financev1.Money{AmountMinor: 50_00, CurrencyCode: "UAH"},
	}))
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	templateID := created.Msg.Template.Id

	// The move alone is refused: relabelling ₴50 as $50 is not a conversion.
	_, err = h.UpdateTemplate(ctxOf(sergiy), connect.NewRequest(&financev1.UpdateTemplateRequest{
		TemplateId: templateID, AccountId: strPtr(id(usd.ID)),
	}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument for a move with no new amount", got)
	}

	updated, err := h.UpdateTemplate(ctxOf(sergiy), connect.NewRequest(&financev1.UpdateTemplateRequest{
		TemplateId: templateID, AccountId: strPtr(id(usd.ID)),
		Amount: &financev1.Money{AmountMinor: 2_00, CurrencyCode: "USD"},
	}))
	if err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}
	if got := updated.Msg.Template.GetAmount().GetCurrencyCode(); got != "USD" {
		t.Errorf("currency = %q, want USD to travel with the account", got)
	}
}

// The same rule on a transaction: moving it to an account in another currency without a new
// amount would turn 500,00 UAH into 500,00 USD.
func TestUpdateTransactionRefusesABareCurrencyMove(t *testing.T) {
	h, store, _ := newTestHandler(t)
	_, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	uah := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	usd := seedAccount(store, "Wise", "card", visibilityShared, "", 0)
	row := store.accounts[id(usd.ID)]
	row.CurrencyCode = "USD"
	store.accounts[id(usd.ID)] = row
	tx := seedTransaction(store, uah, category, sergiy, 500_00, "2026-08-29")

	_, err := h.UpdateTransaction(ctxOf(sergiy), connect.NewRequest(&financev1.UpdateTransactionRequest{
		TransactionId: id(tx.ID), AccountId: strPtr(id(usd.ID)),
	}))
	if got := codeOf(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", got)
	}
}

// Spending with no category is real spending: it is counted in the period total and given a row
// of its own, so the group shares still add up to the headline.
func TestHomeSummaryAccountsForUncategorisedSpending(t *testing.T) {
	h, store, _ := newTestHandler(t)
	group, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	seedTransaction(store, account, category, sergiy, 700_00, "2026-08-29")
	loose := seedTransaction(store, account, category, sergiy, 300_00, "2026-08-29")
	loose.CategoryID = pgtype.UUID{}
	store.transactions[id(loose.ID)] = loose

	resp, err := h.GetHomeSummary(ctxOf(sergiy), connect.NewRequest(&financev1.GetHomeSummaryRequest{
		Kind: financev1.TransactionKind_TRANSACTION_KIND_EXPENSE,
	}))
	if err != nil {
		t.Fatalf("GetHomeSummary: %v", err)
	}
	if got := resp.Msg.PeriodTotal.GetAmountMinor(); got != 1000_00 {
		t.Fatalf("period total = %d, want 100000", got)
	}
	var rows int64
	var uncategorised *financev1.GroupRow
	for _, r := range resp.Msg.Groups {
		rows += r.GetAmount().GetAmountMinor()
		if r.GroupId == "" {
			uncategorised = r
		}
	}
	if rows != resp.Msg.PeriodTotal.GetAmountMinor() {
		t.Errorf("the rows sum to %d but the headline says %d", rows, resp.Msg.PeriodTotal.GetAmountMinor())
	}
	if uncategorised == nil {
		t.Fatal("no row stands for the uncategorised spending")
	}
	if uncategorised.GetAmount().GetAmountMinor() != 300_00 {
		t.Errorf("uncategorised = %d, want 30000", uncategorised.GetAmount().GetAmountMinor())
	}
	// The count names groups, and the bucket is not one.
	if resp.Msg.GroupCount != 1 {
		t.Errorf("group count = %d, want 1 (%q)", resp.Msg.GroupCount, group.Name)
	}
}

// With no kind filter the two sides of the ledger net, and every period total that answers for
// the same window has to net the same way — the feed header and the Home headline included.
func TestPeriodTotalsNetIncomeAgainstExpense(t *testing.T) {
	h, store, _ := newTestHandler(t)
	_, category := seedGroupAndCategory(store, "Їжа", "Продукти")
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	seedTransaction(store, account, category, sergiy, 500_00, "2026-08-29")
	income := seedTransaction(store, account, category, sergiy, 200_00, "2026-08-28")
	income.Type = kindIncome
	store.transactions[id(income.ID)] = income

	feed, err := h.ListTransactions(ctxOf(sergiy), connect.NewRequest(&financev1.ListTransactionsRequest{}))
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	home, err := h.GetHomeSummary(ctxOf(sergiy), connect.NewRequest(&financev1.GetHomeSummaryRequest{}))
	if err != nil {
		t.Fatalf("GetHomeSummary: %v", err)
	}
	if got := feed.Msg.PeriodTotal.GetAmountMinor(); got != 300_00 {
		t.Errorf("feed total = %d, want 50000-20000", got)
	}
	if got := home.Msg.PeriodTotal.GetAmountMinor(); got != feed.Msg.PeriodTotal.GetAmountMinor() {
		t.Errorf("home total = %d, feed total = %d: the same window must agree",
			got, feed.Msg.PeriodTotal.GetAmountMinor())
	}
}
