package handler

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/nnc/family-manager/libs/go/database/pgconv"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
)

func TestSubscriptionRequiresManualPostingByDefault(t *testing.T) {
	h, store, _ := newTestHandler(t)
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	resp, err := h.CreateSubscription(ctxOf(sergiy), connect.NewRequest(&financev1.CreateSubscriptionRequest{
		Name: "Music", Amount: &financev1.Money{AmountMinor: 120_00, CurrencyCode: "UAH"},
		Type: financev1.TransactionType_TRANSACTION_TYPE_EXPENSE, AccountId: id(account.ID),
		Cadence:   &financev1.Cadence{Interval: 1, Unit: financev1.RecurrenceUnit_RECURRENCE_UNIT_MONTH, DayOfMonth: 1},
		NextDueOn: "2026-08-01",
	}))
	if err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}
	sub := resp.Msg.Subscription
	if sub.CategoryId == "" || store.categories[sub.CategoryId].Name != "Music" {
		t.Fatalf("owned category was not created: %+v", sub)
	}
	if got := len(installmentTransactions(store, sub.CategoryId)); got != 0 {
		t.Fatalf("payments = %d before manual posting, want 0", got)
	}
	if _, err := h.PostDueSubscriptions(ctxOf(sergiy)); err != nil {
		t.Fatalf("PostDueSubscriptions: %v", err)
	}
	if got := len(installmentTransactions(store, sub.CategoryId)); got != 0 {
		t.Fatalf("payments = %d after scheduler, want 0", got)
	}
	if _, err := h.PostSubscriptionOccurrence(ctxOf(sergiy), connect.NewRequest(&financev1.PostSubscriptionOccurrenceRequest{
		SubscriptionId: sub.Id, DueOn: "2026-08-01",
	})); err != nil {
		t.Fatalf("PostSubscriptionOccurrence: %v", err)
	}
	if got := len(installmentTransactions(store, sub.CategoryId)); got != 1 {
		t.Fatalf("payments = %d after manual posting, want 1", got)
	}
}

func TestSubscriptionOccurrenceCannotPostTwiceAfterRewind(t *testing.T) {
	h, store, _ := newTestHandler(t)
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	resp, err := h.CreateSubscription(ctxOf(sergiy), connect.NewRequest(&financev1.CreateSubscriptionRequest{
		Name: "Music", Amount: &financev1.Money{AmountMinor: 120_00, CurrencyCode: "UAH"},
		AccountId: id(account.ID), Cadence: &financev1.Cadence{Interval: 1, Unit: financev1.RecurrenceUnit_RECURRENCE_UNIT_MONTH, DayOfMonth: 1},
		NextDueOn: "2026-08-01",
	}))
	if err != nil {
		t.Fatal(err)
	}
	sub := resp.Msg.Subscription
	post := func() {
		t.Helper()
		if _, err := h.PostSubscriptionOccurrence(ctxOf(sergiy), connect.NewRequest(&financev1.PostSubscriptionOccurrenceRequest{
			SubscriptionId: sub.Id, DueOn: "2026-08-01",
		})); err != nil {
			t.Fatal(err)
		}
	}
	post()
	row := store.subscriptions[sub.Id]
	row.NextDueOn = pgDate(testNow.AddDate(0, -1, 2))
	store.subscriptions[sub.Id] = row
	post()
	if got := len(installmentTransactions(store, sub.CategoryId)); got != 1 {
		t.Fatalf("payments after rewind = %d, want 1", got)
	}
	if got := pgconv.DateString(store.subscriptions[sub.Id].NextDueOn); got != "2026-09-01" {
		t.Fatalf("next due = %q, want 2026-09-01", got)
	}
}

func TestAutomaticSubscriptionCannotPostTwiceAfterRewind(t *testing.T) {
	h, store, _ := newTestHandler(t)
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	resp, err := h.CreateSubscription(ctxOf(sergiy), connect.NewRequest(&financev1.CreateSubscriptionRequest{
		Name: "Music", Amount: &financev1.Money{AmountMinor: 120_00, CurrencyCode: "UAH"},
		AccountId: id(account.ID), Cadence: &financev1.Cadence{Interval: 1, Unit: financev1.RecurrenceUnit_RECURRENCE_UNIT_MONTH, DayOfMonth: 1},
		NextDueOn: "2026-08-01", AutoPost: true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	sub := resp.Msg.Subscription
	if got := len(installmentTransactions(store, sub.CategoryId)); got != 1 {
		t.Fatalf("initial payments = %d, want 1", got)
	}
	row := store.subscriptions[sub.Id]
	row.NextDueOn = pgDate(testNow.AddDate(0, -1, 2))
	store.subscriptions[sub.Id] = row
	if posted, err := h.PostDueSubscriptions(ctxOf(sergiy)); err != nil || posted != 0 {
		t.Fatalf("posted after rewind = %d, error = %v", posted, err)
	}
	if got := len(installmentTransactions(store, sub.CategoryId)); got != 1 {
		t.Fatalf("payments after rewind = %d, want 1", got)
	}
	if got := pgconv.DateString(store.subscriptions[sub.Id].NextDueOn); got != "2026-09-01" {
		t.Fatalf("next due = %q, want 2026-09-01", got)
	}
}

func TestSubscriptionRejectsMemberOutsideFamily(t *testing.T) {
	h, store, _ := newTestHandler(t)
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	request := &financev1.CreateSubscriptionRequest{
		Name: "Music", Amount: &financev1.Money{AmountMinor: 120_00, CurrencyCode: "UAH"},
		AccountId: id(account.ID), Cadence: &financev1.Cadence{Interval: 1, Unit: financev1.RecurrenceUnit_RECURRENCE_UNIT_MONTH, DayOfMonth: 1},
		NextDueOn: "2026-08-01", MemberId: otherFam,
	}
	if _, err := h.CreateSubscription(ctxOf(sergiy), connect.NewRequest(request)); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("create with foreign member: %v", err)
	}
	request.MemberId = olena
	resp, err := h.CreateSubscription(ctxOf(sergiy), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	foreign := otherFam
	if _, err := h.UpdateSubscription(ctxOf(sergiy), connect.NewRequest(&financev1.UpdateSubscriptionRequest{
		SubscriptionId: resp.Msg.Subscription.Id, MemberId: &foreign,
	})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("update with foreign member: %v", err)
	}
}

func TestSubscriptionMonthlyApproximation(t *testing.T) {
	if got := monthlyApproximate(120_00, 1, "year"); got != 10_00 {
		t.Fatalf("monthly approximation = %d, want 1000", got)
	}
}
