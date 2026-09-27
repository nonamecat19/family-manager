package handler

import (
	"testing"

	"connectrpc.com/connect"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	"github.com/nnc/family-manager/services/finance/db"
)

func createInvestment(t *testing.T, h *Handler, name string) *financev1.Investment {
	t.Helper()
	resp, err := h.CreateInvestment(ctxOf(sergiy), connect.NewRequest(&financev1.CreateInvestmentRequest{
		Name: name, Kind: financev1.InvestmentKind_INVESTMENT_KIND_STOCKS,
	}))
	if err != nil {
		t.Fatalf("CreateInvestment(%s): %v", name, err)
	}
	return resp.Msg.Investment
}

func groupsWithRole(store *fakeStore, role string) []db.CategoryGroup {
	var out []db.CategoryGroup
	for _, g := range store.groups {
		if g.Role == role {
			out = append(out, g)
		}
	}
	return out
}

func TestCreateInvestmentOwnsACategoryInOneSystemGroup(t *testing.T) {
	h, store, _ := newTestHandler(t)

	first := createInvestment(t, h, "ETF")
	second := createInvestment(t, h, "ОВДП")

	groups := groupsWithRole(store, roleInvestments)
	if len(groups) != 1 {
		t.Fatalf("investment groups = %d, want 1", len(groups))
	}
	for _, inv := range []*financev1.Investment{first, second} {
		cat, ok := store.categories[inv.CategoryId]
		if !ok {
			t.Fatalf("category %s of %s not created", inv.CategoryId, inv.Name)
		}
		if cat.Name != inv.Name || id(cat.GroupID) != id(groups[0].ID) || cat.Kind != kindExpense {
			t.Errorf("category = %+v, want expense %q in the investments group", cat, inv.Name)
		}
	}
	if first.CategoryId == second.CategoryId {
		t.Error("two investments share one category")
	}
}

func TestCreateInvestmentAdoptsAnExistingGroupWithTheSameName(t *testing.T) {
	h, store, _ := newTestHandler(t)
	existing, _ := seedGroupAndCategory(store, "Інвестиції", "Старе")

	inv := createInvestment(t, h, "ETF")

	if inv.GroupId != id(existing.ID) {
		t.Errorf("group = %s, want the existing group %s", inv.GroupId, id(existing.ID))
	}
	if store.groups[id(existing.ID)].Role != roleInvestments {
		t.Errorf("role = %q, want %q", store.groups[id(existing.ID)].Role, roleInvestments)
	}
}

func TestCreateInvestmentRejectsADuplicateName(t *testing.T) {
	h, _, _ := newTestHandler(t)
	createInvestment(t, h, "ETF")

	_, err := h.CreateInvestment(ctxOf(sergiy), connect.NewRequest(&financev1.CreateInvestmentRequest{Name: "etf"}))
	if got := codeOf(t, err); got != connect.CodeAlreadyExists {
		t.Errorf("code = %v, want AlreadyExists", got)
	}
}

func TestInvestmentProfitCountsOnlyVisibleContributions(t *testing.T) {
	h, store, _ := newTestHandler(t)
	shared := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	hers := seedAccount(store, "Олена", "card", visibilityPrivate, olena, 0)
	inv := createInvestment(t, h, "ETF")
	cat := store.categories[inv.CategoryId]
	seedTransaction(store, shared, cat, sergiy, 1_000_00, "2026-08-01")
	seedTransaction(store, hers, cat, olena, 500_00, "2026-08-02")

	resp, err := h.SetInvestmentValue(ctxOf(sergiy), connect.NewRequest(&financev1.SetInvestmentValueRequest{
		InvestmentId: inv.Id, Value: &financev1.Money{AmountMinor: 1_100_00, CurrencyCode: "UAH"},
	}))
	if err != nil {
		t.Fatalf("SetInvestmentValue: %v", err)
	}
	got := resp.Msg.Investment
	if got.Invested.AmountMinor != 1_000_00 {
		t.Errorf("invested = %d, want 100000 (Olena's private contribution hidden)", got.Invested.AmountMinor)
	}
	if got.Profit.AmountMinor != 100_00 || got.ProfitBps != 1000 {
		t.Errorf("profit = %d (%d bps), want 10000 (1000 bps)", got.Profit.AmountMinor, got.ProfitBps)
	}
	if got.ValueUpdatedOn != "2026-08-30" {
		t.Errorf("value_updated_on = %q, want today", got.ValueUpdatedOn)
	}
}

func TestDeleteInvestmentWithContributionsIsRefused(t *testing.T) {
	h, store, _ := newTestHandler(t)
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	inv := createInvestment(t, h, "ETF")
	seedTransaction(store, account, store.categories[inv.CategoryId], sergiy, 100_00, "2026-08-01")

	_, err := h.DeleteInvestment(ctxOf(sergiy), connect.NewRequest(&financev1.DeleteInvestmentRequest{InvestmentId: inv.Id}))
	if got := codeOf(t, err); got != connect.CodeFailedPrecondition {
		t.Errorf("code = %v, want FailedPrecondition", got)
	}

	empty := createInvestment(t, h, "Крипта")
	if _, err := h.DeleteInvestment(ctxOf(sergiy), connect.NewRequest(&financev1.DeleteInvestmentRequest{
		InvestmentId: empty.Id,
	})); err != nil {
		t.Fatalf("DeleteInvestment: %v", err)
	}
	if _, ok := store.categories[empty.CategoryId]; ok {
		t.Error("category of the deleted investment still exists")
	}
}

func TestArchiveInvestmentArchivesItsCategory(t *testing.T) {
	h, store, _ := newTestHandler(t)
	inv := createInvestment(t, h, "ETF")
	archived := true

	if _, err := h.UpdateInvestment(ctxOf(sergiy), connect.NewRequest(&financev1.UpdateInvestmentRequest{
		InvestmentId: inv.Id, Archived: &archived,
	})); err != nil {
		t.Fatalf("UpdateInvestment: %v", err)
	}
	if !store.categories[inv.CategoryId].Archived {
		t.Error("category is not archived")
	}
}

func createInstallment(t *testing.T, h *Handler, account db.Account, total int64, months int32, firstDue string) *financev1.Installment {
	t.Helper()
	resp, err := h.CreateInstallment(ctxOf(sergiy), connect.NewRequest(&financev1.CreateInstallmentRequest{
		Name: "iPhone", Total: &financev1.Money{AmountMinor: total, CurrencyCode: "UAH"},
		Months: months, AccountId: id(account.ID), PurchasedOn: "2026-05-20", FirstDueOn: firstDue,
	}))
	if err != nil {
		t.Fatalf("CreateInstallment: %v", err)
	}
	return resp.Msg.Installment
}

func installmentTransactions(store *fakeStore, categoryID string) []db.Transaction {
	var out []db.Transaction
	for _, tx := range store.transactions {
		if id(tx.CategoryID) == categoryID {
			out = append(out, tx)
		}
	}
	return out
}

func TestCreateInstallmentCatchesUpPastDuePaymentsAndPaysOff(t *testing.T) {
	h, store, rec := newTestHandler(t)
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)

	inst := createInstallment(t, h, account, 1_000_00, 3, "2026-06-15")

	txs := installmentTransactions(store, inst.CategoryId)
	if len(txs) != 3 {
		t.Fatalf("payments = %d, want 3", len(txs))
	}
	var sum int64
	for _, tx := range txs {
		sum += tx.AmountMinor
		if tx.Type != kindExpense {
			t.Errorf("payment type = %q, want expense", tx.Type)
		}
	}
	if sum != 1_000_00 {
		t.Errorf("paid = %d, want exactly the total 100000", sum)
	}
	if inst.Status != financev1.InstallmentStatus_INSTALLMENT_STATUS_PAID_OFF {
		t.Errorf("status = %v, want PAID_OFF", inst.Status)
	}
	if inst.Remaining.AmountMinor != 0 || inst.PaymentsMade != 3 {
		t.Errorf("remaining = %d, payments = %d, want 0 and 3", inst.Remaining.AmountMinor, inst.PaymentsMade)
	}
	if !store.categories[inst.CategoryId].Archived {
		t.Error("category of a paid-off installment is not archived")
	}
	if !rec.sawSubject(string(subjectInstallmentPaidOff)) || !rec.sawSubject("finance.transaction.created") {
		t.Errorf("subjects = %v, want transaction.created and installment.paid_off", rec.subjects)
	}
}

func TestPostDueInstallmentsIsIdempotent(t *testing.T) {
	h, store, _ := newTestHandler(t)
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	inst := createInstallment(t, h, account, 1_200_00, 12, "2026-08-30")

	for range 2 {
		if _, err := h.PostDueInstallments(ctxOf(sergiy)); err != nil {
			t.Fatalf("PostDueInstallments: %v", err)
		}
	}

	txs := installmentTransactions(store, inst.CategoryId)
	if len(txs) != 1 || txs[0].AmountMinor != 100_00 {
		t.Fatalf("payments = %+v, want one of 10000", txs)
	}
	if txs[0].Note != "iPhone 1/12" {
		t.Errorf("note = %q, want %q", txs[0].Note, "iPhone 1/12")
	}
	if got := pgconv.DateString(store.installments[inst.Id].NextDueOn); got != "2026-09-30" {
		t.Errorf("next_due_on = %s, want 2026-09-30", got)
	}
}

func TestPostDueInstallmentsUsesTheHouseholdTimezone(t *testing.T) {
	h, store, _ := newTestHandler(t)
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	inst := createInstallment(t, h, account, 1_200_00, 12, "2026-08-31")
	if n := len(installmentTransactions(store, inst.CategoryId)); n != 0 {
		t.Fatalf("payments in UTC = %d, want 0 before the due day", n)
	}

	settings := store.settings[testFamily]
	settings.Timezone = "Pacific/Kiritimati"
	store.settings[testFamily] = settings
	if _, err := h.PostDueInstallments(ctxOf(sergiy)); err != nil {
		t.Fatalf("PostDueInstallments: %v", err)
	}
	if n := len(installmentTransactions(store, inst.CategoryId)); n != 1 {
		t.Errorf("payments at UTC+14 = %d, want 1 (it is already 31 August there)", n)
	}
}

func TestManualPaymentShortensTheLastInstallment(t *testing.T) {
	h, store, _ := newTestHandler(t)
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	inst := createInstallment(t, h, account, 300_00, 3, "2026-09-15")
	seedTransaction(store, account, store.categories[inst.CategoryId], sergiy, 250_00, "2026-08-20")

	installment := store.installments[inst.Id]
	installment.NextDueOn = day("2026-08-15")
	store.installments[inst.Id] = installment
	if _, err := h.PostDueInstallments(ctxOf(sergiy)); err != nil {
		t.Fatalf("PostDueInstallments: %v", err)
	}

	var auto []db.Transaction
	for _, tx := range installmentTransactions(store, inst.CategoryId) {
		if tx.Note != "" {
			auto = append(auto, tx)
		}
	}
	if len(auto) != 1 || auto[0].AmountMinor != 50_00 {
		t.Fatalf("automatic payments = %+v, want one of the 5000 remainder", auto)
	}
	if store.installments[inst.Id].Status != installmentPaidOff {
		t.Errorf("status = %q, want paid_off", store.installments[inst.Id].Status)
	}
}

func TestCancelInstallmentStopsPostingAndArchivesTheCategory(t *testing.T) {
	h, store, _ := newTestHandler(t)
	account := seedAccount(store, "Mono", "card", visibilityShared, "", 0)
	inst := createInstallment(t, h, account, 1_200_00, 12, "2026-09-30")

	if _, err := h.CancelInstallment(ctxOf(sergiy), connect.NewRequest(&financev1.CancelInstallmentRequest{
		InstallmentId: inst.Id,
	})); err != nil {
		t.Fatalf("CancelInstallment: %v", err)
	}
	installment := store.installments[inst.Id]
	installment.NextDueOn = day("2026-08-01")
	store.installments[inst.Id] = installment
	if _, err := h.PostDueInstallments(ctxOf(sergiy)); err != nil {
		t.Fatalf("PostDueInstallments: %v", err)
	}
	if n := len(installmentTransactions(store, inst.CategoryId)); n != 0 {
		t.Errorf("payments after cancel = %d, want 0", n)
	}
	if !store.categories[inst.CategoryId].Archived {
		t.Error("category is not archived")
	}
}

func TestInstallmentOnAnotherMembersPrivateAccountIsHidden(t *testing.T) {
	h, store, _ := newTestHandler(t)
	hers := seedAccount(store, "Олена", "card", visibilityPrivate, olena, 0)
	resp, err := h.CreateInstallment(ctxOf(olena), connect.NewRequest(&financev1.CreateInstallmentRequest{
		Name: "Пилосос", Total: &financev1.Money{AmountMinor: 600_00}, Months: 6,
		AccountId: id(hers.ID), FirstDueOn: "2026-09-10",
	}))
	if err != nil {
		t.Fatalf("CreateInstallment: %v", err)
	}

	list, err := h.ListInstallments(ctxOf(sergiy), connect.NewRequest(&financev1.ListInstallmentsRequest{}))
	if err != nil {
		t.Fatalf("ListInstallments: %v", err)
	}
	if len(list.Msg.Installments) != 0 {
		t.Errorf("Sergiy sees %d installments, want 0", len(list.Msg.Installments))
	}
	_, err = h.CancelInstallment(ctxOf(sergiy), connect.NewRequest(&financev1.CancelInstallmentRequest{
		InstallmentId: resp.Msg.Installment.Id,
	}))
	if got := codeOf(t, err); got != connect.CodeNotFound {
		t.Errorf("code = %v, want NotFound", got)
	}
}
