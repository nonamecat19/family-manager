package handler

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	"github.com/nnc/family-manager/services/finance/db"
)

// fakeStore is an in-memory db.Querier. What is under test here is the handler's rules —
// family scoping, the private-account boundary, budget arithmetic, transfer shape — not
// Postgres, so the queries are reimplemented in Go with the same predicates the .sql files
// carry. The visibility predicate in particular is copied deliberately: a fake that were more
// permissive than the SQL would make the security tests prove nothing.
type fakeStore struct {
	settings     map[string]db.FinanceSetting
	members      map[string]db.FinanceMember
	accounts     map[string]db.Account
	groups       map[string]db.CategoryGroup
	categories   map[string]db.Category
	transactions map[string]db.Transaction
	budgets      map[string]db.Budget
	templates    map[string]db.QuickTemplate
	recurring    map[string]db.RecurringPayment
	reminders    map[string]db.Reminder
	widgets      map[string]db.WidgetInstance

	failOn map[string]error
	seq    int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		settings:     map[string]db.FinanceSetting{},
		members:      map[string]db.FinanceMember{},
		accounts:     map[string]db.Account{},
		groups:       map[string]db.CategoryGroup{},
		categories:   map[string]db.Category{},
		transactions: map[string]db.Transaction{},
		budgets:      map[string]db.Budget{},
		templates:    map[string]db.QuickTemplate{},
		recurring:    map[string]db.RecurringPayment{},
		reminders:    map[string]db.Reminder{},
		widgets:      map[string]db.WidgetInstance{},
		failOn:       map[string]error{},
	}
}

// fail injects a failure for one operation, so a handler's error path can be reached without
// a database that can be made to misbehave.
func (s *fakeStore) fail(op string) error { return s.failOn[op] }

// newUUID is per-store, not package-level: `go test -shuffle=on` reorders tests, and a shared
// counter would make ids depend on which test ran first.
func (s *fakeStore) newUUID() pgtype.UUID {
	s.seq++
	return pgconv.MustUUID(fmt.Sprintf("%08x-0000-4000-8000-%012x", s.seq, s.seq))
}

// InTx runs the callback against this same store: the fake has no transaction, and a test that
// needed one would be testing pgx rather than the handler.
func (s *fakeStore) InTx(_ context.Context, fn func(q db.Querier) error) error { return fn(s) }

func id(u pgtype.UUID) string { return pgconv.UUIDString(u) }

func same(a, b pgtype.UUID) bool { return a.Valid && b.Valid && id(a) == id(b) }

func now() pgtype.Timestamptz { return pgtype.Timestamptz{Valid: true} }

// visible mirrors `(a.visibility = 'shared' OR a.owner_member_id = @viewer_member_id)`.
func (s *fakeStore) visible(a db.Account, viewer pgtype.UUID) bool {
	return a.Visibility == visibilityShared || same(a.OwnerMemberID, viewer)
}

// balanceOf mirrors the derived balance expression: opening, plus income, minus expense and
// every transfer leaving the account, plus what arrived on every transfer into it.
func (s *fakeStore) balanceOf(a db.Account) int64 {
	balance := a.OpeningBalanceMinor
	for _, t := range s.transactions {
		if same(t.AccountID, a.ID) {
			if t.Type == kindIncome {
				balance += t.AmountMinor
			} else {
				balance -= t.AmountMinor
			}
		}
		if same(t.CounterAccountID, a.ID) {
			if t.ReceivedAmountMinor != nil {
				balance += *t.ReceivedAmountMinor
			} else {
				balance += t.AmountMinor
			}
		}
	}
	return balance
}

/* ------------------------------------------------------------------ settings */

func (s *fakeStore) GetFinanceSettings(_ context.Context, familyID pgtype.UUID) (db.FinanceSetting, error) {
	if err := s.fail("GetFinanceSettings"); err != nil {
		return db.FinanceSetting{}, err
	}
	row, ok := s.settings[id(familyID)]
	if !ok {
		return db.FinanceSetting{}, pgx.ErrNoRows
	}
	return row, nil
}

func (s *fakeStore) BootstrapFinanceSettings(_ context.Context, arg db.BootstrapFinanceSettingsParams) (db.FinanceSetting, error) {
	if err := s.fail("BootstrapFinanceSettings"); err != nil {
		return db.FinanceSetting{}, err
	}
	if row, ok := s.settings[id(arg.FamilyID)]; ok {
		return row, nil
	}
	row := db.FinanceSetting{
		FamilyID: arg.FamilyID, BaseCurrencyCode: arg.BaseCurrencyCode,
		Timezone: arg.Timezone, WeekStartsOn: arg.WeekStartsOn,
		OverspendNotificationsEnabled: true,
		CreatedAt:                     now(), UpdatedAt: now(),
	}
	s.settings[id(arg.FamilyID)] = row
	return row, nil
}

func (s *fakeStore) UpdateFinanceSettings(_ context.Context, arg db.UpdateFinanceSettingsParams) (db.FinanceSetting, error) {
	row, ok := s.settings[id(arg.FamilyID)]
	if !ok {
		return db.FinanceSetting{}, pgx.ErrNoRows
	}
	if arg.BaseCurrencyCode != nil {
		row.BaseCurrencyCode = *arg.BaseCurrencyCode
	}
	if arg.Timezone != nil {
		row.Timezone = *arg.Timezone
	}
	if arg.WeekStartsOn != nil {
		row.WeekStartsOn = *arg.WeekStartsOn
	}
	if arg.OverspendNotificationsEnabled != nil {
		row.OverspendNotificationsEnabled = *arg.OverspendNotificationsEnabled
	}
	if arg.PinLockEnabled != nil {
		row.PinLockEnabled = *arg.PinLockEnabled
	}
	s.settings[id(arg.FamilyID)] = row
	return row, nil
}

func (s *fakeStore) SetOverspendNotifications(_ context.Context, arg db.SetOverspendNotificationsParams) (db.FinanceSetting, error) {
	row, ok := s.settings[id(arg.FamilyID)]
	if !ok {
		return db.FinanceSetting{}, pgx.ErrNoRows
	}
	row.OverspendNotificationsEnabled = arg.OverspendNotificationsEnabled
	s.settings[id(arg.FamilyID)] = row
	return row, nil
}

/* ------------------------------------------------------------------- members */

func memberKey(family, user pgtype.UUID) string { return id(family) + "|" + id(user) }

func (s *fakeStore) ListMembers(_ context.Context, arg db.ListMembersParams) ([]db.FinanceMember, error) {
	if err := s.fail("ListMembers"); err != nil {
		return nil, err
	}
	var out []db.FinanceMember
	for _, m := range s.members {
		if !same(m.FamilyID, arg.FamilyID) {
			continue
		}
		if !arg.IncludePending && m.Status != "active" {
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return id(out[i].UserID) < id(out[j].UserID) })
	return out, nil
}

func (s *fakeStore) GetMember(_ context.Context, arg db.GetMemberParams) (db.FinanceMember, error) {
	m, ok := s.members[memberKey(arg.FamilyID, arg.UserID)]
	if !ok {
		return db.FinanceMember{}, pgx.ErrNoRows
	}
	return m, nil
}

func (s *fakeStore) UpsertMember(_ context.Context, arg db.UpsertMemberParams) (db.FinanceMember, error) {
	m := db.FinanceMember{
		FamilyID: arg.FamilyID, UserID: arg.UserID, DisplayName: arg.DisplayName,
		Initial: arg.Initial, AvatarColorStep: arg.AvatarColorStep, Role: arg.Role,
		Status: arg.Status, Email: arg.Email, JoinedAt: now(),
	}
	s.members[memberKey(arg.FamilyID, arg.UserID)] = m
	return m, nil
}

func (s *fakeStore) DeleteMember(_ context.Context, arg db.DeleteMemberParams) (int64, error) {
	key := memberKey(arg.FamilyID, arg.UserID)
	if _, ok := s.members[key]; !ok {
		return 0, nil
	}
	delete(s.members, key)
	return 1, nil
}

func (s *fakeStore) CountMembers(_ context.Context, familyID pgtype.UUID) (int64, error) {
	var n int64
	for _, m := range s.members {
		if same(m.FamilyID, familyID) && m.Status == "active" {
			n++
		}
	}
	return n, nil
}

/* ------------------------------------------------------------------ accounts */

func (s *fakeStore) ListVisibleAccounts(_ context.Context, arg db.ListVisibleAccountsParams) ([]db.ListVisibleAccountsRow, error) {
	if err := s.fail("ListVisibleAccounts"); err != nil {
		return nil, err
	}
	var out []db.ListVisibleAccountsRow
	for _, a := range s.accounts {
		if !same(a.FamilyID, arg.FamilyID) || !s.visible(a, arg.ViewerMemberID) {
			continue
		}
		if !arg.IncludeArchived && a.Archived {
			continue
		}
		out = append(out, db.ListVisibleAccountsRow{
			ID: a.ID, FamilyID: a.FamilyID, Name: a.Name, Kind: a.Kind,
			Visibility: a.Visibility, OwnerMemberID: a.OwnerMemberID,
			CurrencyCode: a.CurrencyCode, OpeningBalanceMinor: a.OpeningBalanceMinor,
			Icon: a.Icon, ColorStep: a.ColorStep,
			ExcludedFromFamilyTotal: a.ExcludedFromFamilyTotal, Archived: a.Archived,
			SortOrder: a.SortOrder, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
			BalanceMinor: s.balanceOf(a),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return id(out[i].ID) < id(out[j].ID)
	})
	return out, nil
}

func (s *fakeStore) GetVisibleAccount(_ context.Context, arg db.GetVisibleAccountParams) (db.GetVisibleAccountRow, error) {
	a, ok := s.accounts[id(arg.ID)]
	if !ok || !same(a.FamilyID, arg.FamilyID) || !s.visible(a, arg.ViewerMemberID) {
		return db.GetVisibleAccountRow{}, pgx.ErrNoRows
	}
	return db.GetVisibleAccountRow{
		ID: a.ID, FamilyID: a.FamilyID, Name: a.Name, Kind: a.Kind,
		Visibility: a.Visibility, OwnerMemberID: a.OwnerMemberID,
		CurrencyCode: a.CurrencyCode, OpeningBalanceMinor: a.OpeningBalanceMinor,
		Icon: a.Icon, ColorStep: a.ColorStep,
		ExcludedFromFamilyTotal: a.ExcludedFromFamilyTotal, Archived: a.Archived,
		SortOrder: a.SortOrder, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
		BalanceMinor: s.balanceOf(a),
	}, nil
}

func (s *fakeStore) CountHiddenPrivateAccounts(_ context.Context, arg db.CountHiddenPrivateAccountsParams) ([]db.CountHiddenPrivateAccountsRow, error) {
	counts := map[string]int32{}
	owners := map[string]pgtype.UUID{}
	for _, a := range s.accounts {
		if !same(a.FamilyID, arg.FamilyID) || a.Visibility != visibilityPrivate || a.Archived {
			continue
		}
		if same(a.OwnerMemberID, arg.ViewerMemberID) {
			continue
		}
		counts[id(a.OwnerMemberID)]++
		owners[id(a.OwnerMemberID)] = a.OwnerMemberID
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]db.CountHiddenPrivateAccountsRow, 0, len(keys))
	for _, k := range keys {
		out = append(out, db.CountHiddenPrivateAccountsRow{
			OwnerMemberID: owners[k], AccountCount: counts[k],
		})
	}
	return out, nil
}

func (s *fakeStore) SumFamilyBalances(_ context.Context, arg db.SumFamilyBalancesParams) (db.SumFamilyBalancesRow, error) {
	var out db.SumFamilyBalancesRow
	for _, a := range s.accounts {
		// Private accounts never reach this sum: the headline is the household's shared money.
		if !same(a.FamilyID, arg.FamilyID) || a.Visibility != visibilityShared || a.Archived {
			continue
		}
		if a.CurrencyCode != arg.CurrencyCode {
			continue
		}
		out.SharedAccountCount++
		if a.ExcludedFromFamilyTotal {
			continue
		}
		if a.Kind == accountKindSavings {
			out.SavingsMinor += s.balanceOf(a)
			continue
		}
		out.SharedBalanceMinor += s.balanceOf(a)
	}
	return out, nil
}

func (s *fakeStore) CreateAccount(_ context.Context, arg db.CreateAccountParams) (db.Account, error) {
	if err := s.fail("CreateAccount"); err != nil {
		return db.Account{}, err
	}
	a := db.Account{
		ID: s.newUUID(), FamilyID: arg.FamilyID, Name: arg.Name, Kind: arg.Kind,
		Visibility: arg.Visibility, OwnerMemberID: arg.OwnerMemberID,
		CurrencyCode: arg.CurrencyCode, OpeningBalanceMinor: arg.OpeningBalanceMinor,
		Icon: arg.Icon, ColorStep: arg.ColorStep,
		ExcludedFromFamilyTotal: arg.ExcludedFromFamilyTotal,
		SortOrder:               int32(len(s.accounts)), CreatedAt: now(), UpdatedAt: now(),
	}
	s.accounts[id(a.ID)] = a
	return a, nil
}

func (s *fakeStore) UpdateAccount(_ context.Context, arg db.UpdateAccountParams) (db.Account, error) {
	a, ok := s.accounts[id(arg.ID)]
	if !ok || !same(a.FamilyID, arg.FamilyID) {
		return db.Account{}, pgx.ErrNoRows
	}
	if arg.Name != nil {
		a.Name = *arg.Name
	}
	if arg.Kind != nil {
		a.Kind = *arg.Kind
	}
	if arg.Icon != nil {
		a.Icon = *arg.Icon
	}
	if arg.ColorStep != nil {
		a.ColorStep = *arg.ColorStep
	}
	if arg.ExcludedFromFamilyTotal != nil {
		a.ExcludedFromFamilyTotal = *arg.ExcludedFromFamilyTotal
	}
	if arg.OpeningBalanceMinor != nil {
		a.OpeningBalanceMinor = *arg.OpeningBalanceMinor
	}
	s.accounts[id(a.ID)] = a
	return a, nil
}

func (s *fakeStore) SetAccountVisibility(_ context.Context, arg db.SetAccountVisibilityParams) (db.Account, error) {
	a, ok := s.accounts[id(arg.ID)]
	if !ok || !same(a.FamilyID, arg.FamilyID) {
		return db.Account{}, pgx.ErrNoRows
	}
	a.Visibility = arg.Visibility
	a.OwnerMemberID = arg.OwnerMemberID
	a.ExcludedFromFamilyTotal = arg.Visibility == visibilityPrivate || a.ExcludedFromFamilyTotal
	s.accounts[id(a.ID)] = a
	return a, nil
}

func (s *fakeStore) SetAccountArchived(_ context.Context, arg db.SetAccountArchivedParams) (db.Account, error) {
	a, ok := s.accounts[id(arg.ID)]
	if !ok || !same(a.FamilyID, arg.FamilyID) {
		return db.Account{}, pgx.ErrNoRows
	}
	a.Archived = arg.Archived
	s.accounts[id(a.ID)] = a
	return a, nil
}

func (s *fakeStore) DeleteAccount(_ context.Context, arg db.DeleteAccountParams) (int64, error) {
	a, ok := s.accounts[id(arg.ID)]
	if !ok || !same(a.FamilyID, arg.FamilyID) {
		return 0, nil
	}
	delete(s.accounts, id(arg.ID))
	return 1, nil
}

func (s *fakeStore) ReorderAccount(_ context.Context, arg db.ReorderAccountParams) error {
	a, ok := s.accounts[id(arg.ID)]
	if !ok || !same(a.FamilyID, arg.FamilyID) || !s.visible(a, arg.ViewerMemberID) {
		return nil
	}
	a.SortOrder = arg.SortOrder
	s.accounts[id(a.ID)] = a
	return nil
}

func (s *fakeStore) CountAccountTransactions(_ context.Context, arg db.CountAccountTransactionsParams) (int64, error) {
	var n int64
	for _, t := range s.transactions {
		if !same(t.FamilyID, arg.FamilyID) {
			continue
		}
		if same(t.AccountID, arg.AccountID) || same(t.CounterAccountID, arg.AccountID) {
			n++
		}
	}
	return n, nil
}

/* ---------------------------------------------------------------- categories */

func (s *fakeStore) ListCategoryGroups(_ context.Context, arg db.ListCategoryGroupsParams) ([]db.CategoryGroup, error) {
	var out []db.CategoryGroup
	for _, g := range s.groups {
		if !same(g.FamilyID, arg.FamilyID) {
			continue
		}
		if arg.Kind != nil && g.Kind != *arg.Kind {
			continue
		}
		if !arg.IncludeArchived && g.Archived {
			continue
		}
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return id(out[i].ID) < id(out[j].ID)
	})
	return out, nil
}

func (s *fakeStore) GetCategoryGroup(_ context.Context, arg db.GetCategoryGroupParams) (db.CategoryGroup, error) {
	g, ok := s.groups[id(arg.ID)]
	if !ok || !same(g.FamilyID, arg.FamilyID) {
		return db.CategoryGroup{}, pgx.ErrNoRows
	}
	return g, nil
}

func (s *fakeStore) CreateCategoryGroup(_ context.Context, arg db.CreateCategoryGroupParams) (db.CategoryGroup, error) {
	if err := s.fail("CreateCategoryGroup"); err != nil {
		return db.CategoryGroup{}, err
	}
	g := db.CategoryGroup{
		ID: s.newUUID(), FamilyID: arg.FamilyID, Name: arg.Name, Kind: arg.Kind,
		Icon: arg.Icon, ColorStep: arg.ColorStep, SortOrder: int32(len(s.groups)),
		CreatedAt: now(), UpdatedAt: now(),
	}
	s.groups[id(g.ID)] = g
	return g, nil
}

func (s *fakeStore) UpdateCategoryGroup(_ context.Context, arg db.UpdateCategoryGroupParams) (db.CategoryGroup, error) {
	g, ok := s.groups[id(arg.ID)]
	if !ok || !same(g.FamilyID, arg.FamilyID) {
		return db.CategoryGroup{}, pgx.ErrNoRows
	}
	if arg.Name != nil {
		g.Name = *arg.Name
	}
	if arg.Icon != nil {
		g.Icon = *arg.Icon
	}
	if arg.ColorStep != nil {
		g.ColorStep = *arg.ColorStep
	}
	if arg.Archived != nil {
		g.Archived = *arg.Archived
	}
	s.groups[id(g.ID)] = g
	return g, nil
}

func (s *fakeStore) DeleteCategoryGroup(_ context.Context, arg db.DeleteCategoryGroupParams) (int64, error) {
	g, ok := s.groups[id(arg.ID)]
	if !ok || !same(g.FamilyID, arg.FamilyID) {
		return 0, nil
	}
	delete(s.groups, id(arg.ID))
	return 1, nil
}

func (s *fakeStore) ReorderCategoryGroup(_ context.Context, arg db.ReorderCategoryGroupParams) error {
	g, ok := s.groups[id(arg.ID)]
	if !ok || !same(g.FamilyID, arg.FamilyID) {
		return nil
	}
	g.SortOrder = arg.SortOrder
	s.groups[id(g.ID)] = g
	return nil
}

func (s *fakeStore) CountCategoriesInGroup(_ context.Context, arg db.CountCategoriesInGroupParams) (int64, error) {
	var n int64
	for _, c := range s.categories {
		if same(c.GroupID, arg.GroupID) && same(c.FamilyID, arg.FamilyID) {
			n++
		}
	}
	return n, nil
}

func (s *fakeStore) MoveCategoriesToGroup(_ context.Context, arg db.MoveCategoriesToGroupParams) (int64, error) {
	var n int64
	for key, c := range s.categories {
		if same(c.GroupID, arg.GroupID) && same(c.FamilyID, arg.FamilyID) {
			c.GroupID = arg.GroupID_2
			s.categories[key] = c
			n++
		}
	}
	return n, nil
}

func (s *fakeStore) ListCategories(_ context.Context, arg db.ListCategoriesParams) ([]db.Category, error) {
	var out []db.Category
	for _, c := range s.categories {
		if !same(c.FamilyID, arg.FamilyID) {
			continue
		}
		if arg.GroupID.Valid && !same(c.GroupID, arg.GroupID) {
			continue
		}
		if arg.Kind != nil && c.Kind != *arg.Kind {
			continue
		}
		if !arg.IncludeArchived && c.Archived {
			continue
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return id(out[i].ID) < id(out[j].ID)
	})
	return out, nil
}

func (s *fakeStore) GetCategory(_ context.Context, arg db.GetCategoryParams) (db.Category, error) {
	c, ok := s.categories[id(arg.ID)]
	if !ok || !same(c.FamilyID, arg.FamilyID) {
		return db.Category{}, pgx.ErrNoRows
	}
	return c, nil
}

func (s *fakeStore) CreateCategory(_ context.Context, arg db.CreateCategoryParams) (db.Category, error) {
	if err := s.fail("CreateCategory"); err != nil {
		return db.Category{}, err
	}
	c := db.Category{
		ID: s.newUUID(), FamilyID: arg.FamilyID, GroupID: arg.GroupID, Name: arg.Name,
		Kind: arg.Kind, Icon: arg.Icon, SortOrder: int32(len(s.categories)),
		CreatedAt: now(), UpdatedAt: now(),
	}
	s.categories[id(c.ID)] = c
	return c, nil
}

func (s *fakeStore) UpdateCategory(_ context.Context, arg db.UpdateCategoryParams) (db.Category, error) {
	c, ok := s.categories[id(arg.ID)]
	if !ok || !same(c.FamilyID, arg.FamilyID) {
		return db.Category{}, pgx.ErrNoRows
	}
	if arg.Name != nil {
		c.Name = *arg.Name
	}
	if arg.Icon != nil {
		c.Icon = *arg.Icon
	}
	if arg.Archived != nil {
		c.Archived = *arg.Archived
	}
	s.categories[id(c.ID)] = c
	return c, nil
}

func (s *fakeStore) MoveCategory(_ context.Context, arg db.MoveCategoryParams) (db.Category, error) {
	c, ok := s.categories[id(arg.ID)]
	if !ok || !same(c.FamilyID, arg.FamilyID) {
		return db.Category{}, pgx.ErrNoRows
	}
	c.GroupID = arg.GroupID
	s.categories[id(c.ID)] = c
	return c, nil
}

func (s *fakeStore) DeleteCategory(_ context.Context, arg db.DeleteCategoryParams) (int64, error) {
	c, ok := s.categories[id(arg.ID)]
	if !ok || !same(c.FamilyID, arg.FamilyID) {
		return 0, nil
	}
	delete(s.categories, id(arg.ID))
	return 1, nil
}

func (s *fakeStore) ReorderCategory(_ context.Context, arg db.ReorderCategoryParams) (int64, error) {
	c, ok := s.categories[id(arg.ID)]
	if !ok || !same(c.FamilyID, arg.FamilyID) || !same(c.GroupID, arg.GroupID) {
		return 0, nil
	}
	c.SortOrder = arg.SortOrder
	s.categories[id(c.ID)] = c
	return 1, nil
}

func (s *fakeStore) MoveTransactionsToCategory(_ context.Context, arg db.MoveTransactionsToCategoryParams) (int64, error) {
	var n int64
	for key, t := range s.transactions {
		if same(t.CategoryID, arg.CategoryID) && same(t.FamilyID, arg.FamilyID) {
			t.CategoryID = arg.TargetCategoryID
			s.transactions[key] = t
			n++
		}
	}
	return n, nil
}

func (s *fakeStore) CountCategoryTransactions(_ context.Context, arg db.CountCategoryTransactionsParams) (int64, error) {
	var n int64
	for _, t := range s.transactions {
		if same(t.CategoryID, arg.CategoryID) && same(t.FamilyID, arg.FamilyID) {
			n++
		}
	}
	return n, nil
}

/* -------------------------------------------------------------- transactions */

func (s *fakeStore) groupOf(categoryID pgtype.UUID) pgtype.UUID {
	if !categoryID.Valid {
		return pgtype.UUID{}
	}
	if c, ok := s.categories[id(categoryID)]; ok {
		return c.GroupID
	}
	return pgtype.UUID{}
}

// txVisible mirrors the JOIN on accounts every read in transactions.sql carries: a transaction
// is readable when the account it was paid from is readable.
func (s *fakeStore) txVisible(t db.Transaction, viewer pgtype.UUID) bool {
	a, ok := s.accounts[id(t.AccountID)]
	return ok && s.visible(a, viewer)
}

func (s *fakeStore) CreateTransaction(_ context.Context, arg db.CreateTransactionParams) (db.Transaction, error) {
	if err := s.fail("CreateTransaction"); err != nil {
		return db.Transaction{}, err
	}
	t := db.Transaction{
		ID: s.newUUID(), FamilyID: arg.FamilyID, Type: arg.Type, AccountID: arg.AccountID,
		CounterAccountID: arg.CounterAccountID, CategoryID: arg.CategoryID,
		AmountMinor: arg.AmountMinor, CurrencyCode: arg.CurrencyCode,
		ReceivedAmountMinor: arg.ReceivedAmountMinor, ReceivedCurrencyCode: arg.ReceivedCurrencyCode,
		Note: arg.Note, Merchant: arg.Merchant, OccurredOn: arg.OccurredOn,
		MemberID: arg.MemberID, CreatedByUserID: arg.CreatedByUserID,
		TemplateID: arg.TemplateID, RecurringID: arg.RecurringID,
		CreatedAt: now(), UpdatedAt: now(),
	}
	s.transactions[id(t.ID)] = t
	return t, nil
}

func (s *fakeStore) GetVisibleTransaction(_ context.Context, arg db.GetVisibleTransactionParams) (db.GetVisibleTransactionRow, error) {
	t, ok := s.transactions[id(arg.ID)]
	if !ok || !same(t.FamilyID, arg.FamilyID) || !s.txVisible(t, arg.ViewerMemberID) {
		return db.GetVisibleTransactionRow{}, pgx.ErrNoRows
	}
	return db.GetVisibleTransactionRow{
		ID: t.ID, FamilyID: t.FamilyID, Type: t.Type, AccountID: t.AccountID,
		CounterAccountID: t.CounterAccountID, CategoryID: t.CategoryID,
		AmountMinor: t.AmountMinor, CurrencyCode: t.CurrencyCode,
		ReceivedAmountMinor: t.ReceivedAmountMinor, ReceivedCurrencyCode: t.ReceivedCurrencyCode,
		Note: t.Note, Merchant: t.Merchant, OccurredOn: t.OccurredOn, MemberID: t.MemberID,
		CreatedByUserID: t.CreatedByUserID, TemplateID: t.TemplateID, RecurringID: t.RecurringID,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, GroupID: s.groupOf(t.CategoryID),
	}, nil
}

func (s *fakeStore) UpdateTransaction(_ context.Context, arg db.UpdateTransactionParams) (db.Transaction, error) {
	t, ok := s.transactions[id(arg.ID)]
	if !ok || !same(t.FamilyID, arg.FamilyID) {
		return db.Transaction{}, pgx.ErrNoRows
	}
	if arg.Type != nil {
		t.Type = *arg.Type
	}
	if arg.AccountID.Valid {
		t.AccountID = arg.AccountID
	}
	if arg.CategoryID.Valid {
		t.CategoryID = arg.CategoryID
	}
	if arg.AmountMinor != nil {
		t.AmountMinor = *arg.AmountMinor
	}
	if arg.CurrencyCode != nil {
		t.CurrencyCode = *arg.CurrencyCode
	}
	if arg.Note != nil {
		t.Note = *arg.Note
	}
	if arg.Merchant != nil {
		t.Merchant = *arg.Merchant
	}
	if arg.OccurredOn.Valid {
		t.OccurredOn = arg.OccurredOn
	}
	if arg.MemberID.Valid {
		t.MemberID = arg.MemberID
	}
	s.transactions[id(t.ID)] = t
	return t, nil
}

func (s *fakeStore) DeleteTransaction(_ context.Context, arg db.DeleteTransactionParams) (int64, error) {
	t, ok := s.transactions[id(arg.ID)]
	if !ok || !same(t.FamilyID, arg.FamilyID) {
		return 0, nil
	}
	delete(s.transactions, id(arg.ID))
	return 1, nil
}

// matchesIDs is the `cardinality(...) = 0 OR col = ANY(...)` sentinel: an empty list is no
// filter at all, which is what lets the app send one request shape.
func matchesIDs(value pgtype.UUID, list []pgtype.UUID) bool {
	if len(list) == 0 {
		return true
	}
	for _, want := range list {
		if same(value, want) {
			return true
		}
	}
	return false
}

func (s *fakeStore) feedRows(familyID, viewer pgtype.UUID) []db.Transaction {
	var out []db.Transaction
	for _, t := range s.transactions {
		if same(t.FamilyID, familyID) && s.txVisible(t, viewer) {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].OccurredOn.Time.Equal(out[j].OccurredOn.Time) {
			return out[i].OccurredOn.Time.After(out[j].OccurredOn.Time)
		}
		return id(out[i].ID) > id(out[j].ID)
	})
	return out
}

func (s *fakeStore) inWindow(t db.Transaction, from, to pgtype.Date) bool {
	day := t.OccurredOn.Time
	return !day.Before(from.Time) && !day.After(to.Time)
}

func (s *fakeStore) ListVisibleTransactions(_ context.Context, arg db.ListVisibleTransactionsParams) ([]db.ListVisibleTransactionsRow, error) {
	if err := s.fail("ListVisibleTransactions"); err != nil {
		return nil, err
	}
	var out []db.ListVisibleTransactionsRow
	for _, t := range s.feedRows(arg.FamilyID, arg.ViewerMemberID) {
		if !s.inWindow(t, arg.FromDate, arg.ToDate) {
			continue
		}
		if arg.Kind != nil && t.Type != *arg.Kind {
			continue
		}
		if !arg.IncludeTransfers && t.Type == kindTransfer {
			continue
		}
		if !matchesIDs(t.MemberID, arg.MemberIds) || !matchesIDs(t.AccountID, arg.AccountIds) {
			continue
		}
		if !matchesIDs(t.CategoryID, arg.CategoryIds) || !matchesIDs(s.groupOf(t.CategoryID), arg.GroupIds) {
			continue
		}
		if arg.Query != nil {
			q := strings.ToLower(*arg.Query)
			if !strings.Contains(strings.ToLower(t.Note), q) &&
				!strings.Contains(strings.ToLower(t.Merchant), q) {
				continue
			}
		}
		if arg.CursorDate.Valid {
			after := t.OccurredOn.Time.Before(arg.CursorDate.Time) ||
				(t.OccurredOn.Time.Equal(arg.CursorDate.Time) && id(t.ID) < id(arg.CursorID))
			if !after {
				continue
			}
		}
		out = append(out, db.ListVisibleTransactionsRow{
			ID: t.ID, FamilyID: t.FamilyID, Type: t.Type, AccountID: t.AccountID,
			CounterAccountID: t.CounterAccountID, CategoryID: t.CategoryID,
			AmountMinor: t.AmountMinor, CurrencyCode: t.CurrencyCode,
			ReceivedAmountMinor: t.ReceivedAmountMinor, ReceivedCurrencyCode: t.ReceivedCurrencyCode,
			Note: t.Note, Merchant: t.Merchant, OccurredOn: t.OccurredOn, MemberID: t.MemberID,
			CreatedByUserID: t.CreatedByUserID, TemplateID: t.TemplateID,
			RecurringID: t.RecurringID, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
			GroupID: s.groupOf(t.CategoryID),
		})
		if int32(len(out)) == arg.PageSize {
			break
		}
	}
	return out, nil
}

// aggregable is the predicate shared by every Sum* query: visible, inside the window, in the
// household's currency, and never a transfer.
func (s *fakeStore) aggregable(t db.Transaction, familyID, viewer pgtype.UUID, from, to pgtype.Date, currency string, kind *string) bool {
	if !same(t.FamilyID, familyID) || !s.txVisible(t, viewer) {
		return false
	}
	if !s.inWindow(t, from, to) || t.Type == kindTransfer || t.CurrencyCode != currency {
		return false
	}
	return kind == nil || t.Type == *kind
}

func (s *fakeStore) SumVisibleTransactions(_ context.Context, arg db.SumVisibleTransactionsParams) (db.SumVisibleTransactionsRow, error) {
	var out db.SumVisibleTransactionsRow
	for _, t := range s.transactions {
		if !s.aggregable(t, arg.FamilyID, arg.ViewerMemberID, arg.FromDate, arg.ToDate, arg.CurrencyCode, arg.Kind) {
			continue
		}
		if !matchesIDs(t.MemberID, arg.MemberIds) || !matchesIDs(t.AccountID, arg.AccountIds) ||
			!matchesIDs(t.CategoryID, arg.CategoryIds) ||
			!matchesIDs(s.groupOf(t.CategoryID), arg.GroupIds) {
			continue
		}
		if arg.Query != nil {
			q := strings.ToLower(*arg.Query)
			if !strings.Contains(strings.ToLower(t.Note), q) &&
				!strings.Contains(strings.ToLower(t.Merchant), q) {
				continue
			}
		}
		out.TotalMinor += signed(t, arg.Kind)
		out.TransactionCount++
	}
	return out, nil
}

func (s *fakeStore) SumByGroup(_ context.Context, arg db.SumByGroupParams) ([]db.SumByGroupRow, error) {
	totals := map[string]*db.SumByGroupRow{}
	for _, t := range s.transactions {
		if !s.aggregable(t, arg.FamilyID, arg.ViewerMemberID, arg.FromDate, arg.ToDate, arg.CurrencyCode, arg.Kind) {
			continue
		}
		if !matchesIDs(t.MemberID, arg.MemberIds) || !matchesIDs(t.AccountID, arg.AccountIds) {
			continue
		}
		// LEFT JOIN semantics: a transaction with no category is summed under a NULL group
		// rather than dropped, which is what puts it in the period total.
		group := s.groupOf(t.CategoryID)
		row := totals[id(group)]
		if row == nil {
			row = &db.SumByGroupRow{GroupID: group}
			totals[id(group)] = row
		}
		row.TotalMinor += signed(t, arg.Kind)
		row.TransactionCount++
	}
	return sortedRows(totals), nil
}

func (s *fakeStore) SumByCategory(_ context.Context, arg db.SumByCategoryParams) ([]db.SumByCategoryRow, error) {
	totals := map[string]*db.SumByCategoryRow{}
	for _, t := range s.transactions {
		if !s.aggregable(t, arg.FamilyID, arg.ViewerMemberID, arg.FromDate, arg.ToDate, arg.CurrencyCode, arg.Kind) {
			continue
		}
		if !matchesIDs(t.MemberID, arg.MemberIds) || !t.CategoryID.Valid {
			continue
		}
		group := s.groupOf(t.CategoryID)
		if arg.GroupID.Valid && !same(group, arg.GroupID) {
			continue
		}
		row := totals[id(t.CategoryID)]
		if row == nil {
			row = &db.SumByCategoryRow{CategoryID: t.CategoryID}
			totals[id(t.CategoryID)] = row
		}
		row.TotalMinor += signed(t, arg.Kind)
		row.TransactionCount++
	}
	return sortedRows(totals), nil
}

func (s *fakeStore) SumByMember(_ context.Context, arg db.SumByMemberParams) ([]db.SumByMemberRow, error) {
	totals := map[string]*db.SumByMemberRow{}
	for _, t := range s.transactions {
		if !s.aggregable(t, arg.FamilyID, arg.ViewerMemberID, arg.FromDate, arg.ToDate, arg.CurrencyCode, arg.Kind) {
			continue
		}
		if !matchesIDs(t.AccountID, arg.AccountIds) {
			continue
		}
		group := s.groupOf(t.CategoryID)
		key := id(t.MemberID) + "|" + id(group)
		row := totals[key]
		if row == nil {
			row = &db.SumByMemberRow{MemberID: t.MemberID, GroupID: group}
			totals[key] = row
		}
		row.TotalMinor += signed(t, arg.Kind)
		row.TransactionCount++
	}
	return sortedRows(totals), nil
}

func (s *fakeStore) SumDailyTotals(_ context.Context, arg db.SumDailyTotalsParams) ([]db.SumDailyTotalsRow, error) {
	totals := map[string]*db.SumDailyTotalsRow{}
	for _, t := range s.transactions {
		if !s.aggregable(t, arg.FamilyID, arg.ViewerMemberID, arg.FromDate, arg.ToDate, arg.CurrencyCode, arg.Kind) {
			continue
		}
		if !matchesIDs(t.MemberID, arg.MemberIds) || !matchesIDs(t.AccountID, arg.AccountIds) {
			continue
		}
		group := s.groupOf(t.CategoryID)
		key := t.OccurredOn.Time.Format(dateLayout) + "|" + id(t.MemberID) + "|" + id(group)
		row := totals[key]
		if row == nil {
			row = &db.SumDailyTotalsRow{OccurredOn: t.OccurredOn, MemberID: t.MemberID, GroupID: group}
			totals[key] = row
		}
		row.TotalMinor += signed(t, arg.Kind)
	}
	return sortedRows(totals), nil
}

// signed mirrors the CASE every aggregate query carries: with no kind filter the two sides of
// the ledger net, so "the period cost 3,000" rather than "5,000 and 2,000 summed as
// magnitudes". With a filter every row is on one side and the sum is a plain magnitude.
func signed(t db.Transaction, kind *string) int64 {
	if kind == nil && t.Type == kindIncome {
		return -t.AmountMinor
	}
	return t.AmountMinor
}

// sortedRows makes every aggregate deterministic: it returns the rows in map-key order. Map
// iteration order is random, and a fake that returned rows in a different order each run would
// make `go test -shuffle=on` flake for a reason that has nothing to do with the handler.
func sortedRows[T any](m map[string]*T) []T {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]T, 0, len(keys))
	for _, k := range keys {
		out = append(out, *m[k])
	}
	return out
}

func (s *fakeStore) SumBudgetSpend(_ context.Context, arg db.SumBudgetSpendParams) (int64, error) {
	expense := kindExpense
	var total int64
	for _, t := range s.transactions {
		if !s.aggregable(t, arg.FamilyID, arg.ViewerMemberID, arg.FromDate, arg.ToDate, arg.CurrencyCode, &expense) {
			continue
		}
		if !t.CategoryID.Valid {
			continue
		}
		if arg.GroupID.Valid && !same(s.groupOf(t.CategoryID), arg.GroupID) {
			continue
		}
		if arg.CategoryID.Valid && !same(t.CategoryID, arg.CategoryID) {
			continue
		}
		if arg.MemberID.Valid && !same(t.MemberID, arg.MemberID) {
			continue
		}
		total += t.AmountMinor
	}
	return total, nil
}

func (s *fakeStore) CountTransactionsForRecurringOccurrence(_ context.Context, arg db.CountTransactionsForRecurringOccurrenceParams) (int64, error) {
	var n int64
	for _, t := range s.transactions {
		if same(t.FamilyID, arg.FamilyID) && same(t.RecurringID, arg.RecurringID) &&
			t.OccurredOn.Time.Equal(arg.OccurredOn.Time) {
			n++
		}
	}
	return n, nil
}

/* -------------------------------------------------------------------- budgets */

func (s *fakeStore) ListBudgets(_ context.Context, arg db.ListBudgetsParams) ([]db.Budget, error) {
	var out []db.Budget
	for _, b := range s.budgets {
		if !same(b.FamilyID, arg.FamilyID) {
			continue
		}
		if arg.TargetKind != nil && b.TargetKind != *arg.TargetKind {
			continue
		}
		if !arg.IncludeArchived && b.Archived {
			continue
		}
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return id(out[i].ID) < id(out[j].ID) })
	return out, nil
}

func (s *fakeStore) GetBudget(_ context.Context, arg db.GetBudgetParams) (db.Budget, error) {
	b, ok := s.budgets[id(arg.ID)]
	if !ok || !same(b.FamilyID, arg.FamilyID) {
		return db.Budget{}, pgx.ErrNoRows
	}
	return b, nil
}

func (s *fakeStore) ListBudgetsForCategory(_ context.Context, arg db.ListBudgetsForCategoryParams) ([]db.Budget, error) {
	group := s.groupOf(arg.CategoryID)
	var out []db.Budget
	for _, b := range s.budgets {
		if !same(b.FamilyID, arg.FamilyID) || b.Archived {
			continue
		}
		// Both the category's own budget and its group's: spend in a category counts toward
		// both, which is the rule the app's overspend toast depends on.
		if (b.TargetKind == targetCategory && same(b.CategoryID, arg.CategoryID)) ||
			(b.TargetKind == targetGroup && same(b.GroupID, group)) {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return id(out[i].ID) < id(out[j].ID) })
	return out, nil
}

func (s *fakeStore) CreateBudget(_ context.Context, arg db.CreateBudgetParams) (db.Budget, error) {
	if err := s.fail("CreateBudget"); err != nil {
		return db.Budget{}, err
	}
	b := db.Budget{
		ID: s.newUUID(), FamilyID: arg.FamilyID, TargetKind: arg.TargetKind,
		GroupID: arg.GroupID, CategoryID: arg.CategoryID, LimitMinor: arg.LimitMinor,
		CurrencyCode: arg.CurrencyCode, Period: arg.Period, StartOn: arg.StartOn,
		MemberID: arg.MemberID, NotifyOnExceed: arg.NotifyOnExceed,
		SortOrder: int32(len(s.budgets)), CreatedAt: now(), UpdatedAt: now(),
	}
	s.budgets[id(b.ID)] = b
	return b, nil
}

func (s *fakeStore) UpdateBudget(_ context.Context, arg db.UpdateBudgetParams) (db.Budget, error) {
	b, ok := s.budgets[id(arg.ID)]
	if !ok || !same(b.FamilyID, arg.FamilyID) {
		return db.Budget{}, pgx.ErrNoRows
	}
	if arg.LimitMinor != nil {
		b.LimitMinor = *arg.LimitMinor
	}
	if arg.Period != nil {
		b.Period = *arg.Period
	}
	if arg.StartOn.Valid {
		b.StartOn = arg.StartOn
	}
	if arg.NotifyOnExceed != nil {
		b.NotifyOnExceed = *arg.NotifyOnExceed
	}
	if arg.Archived != nil {
		b.Archived = *arg.Archived
	}
	s.budgets[id(b.ID)] = b
	return b, nil
}

func (s *fakeStore) DeleteBudget(_ context.Context, arg db.DeleteBudgetParams) (int64, error) {
	b, ok := s.budgets[id(arg.ID)]
	if !ok || !same(b.FamilyID, arg.FamilyID) {
		return 0, nil
	}
	delete(s.budgets, id(arg.ID))
	return 1, nil
}

/* ------------------------------------------------------------------ templates */

func (s *fakeStore) ListTemplates(_ context.Context, arg db.ListTemplatesParams) ([]db.QuickTemplate, error) {
	var out []db.QuickTemplate
	for _, t := range s.templates {
		if same(t.FamilyID, arg.FamilyID) && same(t.OwnerUserID, arg.OwnerUserID) {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return id(out[i].ID) < id(out[j].ID)
	})
	return out, nil
}

func (s *fakeStore) GetTemplate(_ context.Context, arg db.GetTemplateParams) (db.QuickTemplate, error) {
	t, ok := s.templates[id(arg.ID)]
	if !ok || !same(t.FamilyID, arg.FamilyID) || !same(t.OwnerUserID, arg.OwnerUserID) {
		return db.QuickTemplate{}, pgx.ErrNoRows
	}
	return t, nil
}

func (s *fakeStore) CreateTemplate(_ context.Context, arg db.CreateTemplateParams) (db.QuickTemplate, error) {
	t := db.QuickTemplate{
		ID: s.newUUID(), FamilyID: arg.FamilyID, OwnerUserID: arg.OwnerUserID,
		Label: arg.Label, Icon: arg.Icon, AmountMinor: arg.AmountMinor,
		CurrencyCode: arg.CurrencyCode, Type: arg.Type, CategoryID: arg.CategoryID,
		AccountID: arg.AccountID, MemberID: arg.MemberID,
		SortOrder: int32(len(s.templates)), CreatedAt: now(), UpdatedAt: now(),
	}
	s.templates[id(t.ID)] = t
	return t, nil
}

func (s *fakeStore) UpdateTemplate(_ context.Context, arg db.UpdateTemplateParams) (db.QuickTemplate, error) {
	t, ok := s.templates[id(arg.ID)]
	if !ok || !same(t.FamilyID, arg.FamilyID) || !same(t.OwnerUserID, arg.OwnerUserID) {
		return db.QuickTemplate{}, pgx.ErrNoRows
	}
	if arg.Label != nil {
		t.Label = *arg.Label
	}
	if arg.Icon != nil {
		t.Icon = *arg.Icon
	}
	if arg.AmountMinor != nil {
		t.AmountMinor = *arg.AmountMinor
	}
	if arg.CurrencyCode != nil {
		t.CurrencyCode = *arg.CurrencyCode
	}
	if arg.CategoryID.Valid {
		t.CategoryID = arg.CategoryID
	}
	if arg.AccountID.Valid {
		t.AccountID = arg.AccountID
	}
	if arg.MemberID.Valid {
		t.MemberID = arg.MemberID
	}
	s.templates[id(t.ID)] = t
	return t, nil
}

func (s *fakeStore) DeleteTemplate(_ context.Context, arg db.DeleteTemplateParams) (int64, error) {
	t, ok := s.templates[id(arg.ID)]
	if !ok || !same(t.FamilyID, arg.FamilyID) || !same(t.OwnerUserID, arg.OwnerUserID) {
		return 0, nil
	}
	delete(s.templates, id(arg.ID))
	return 1, nil
}

func (s *fakeStore) ReorderTemplate(_ context.Context, arg db.ReorderTemplateParams) error {
	t, ok := s.templates[id(arg.ID)]
	if !ok || !same(t.FamilyID, arg.FamilyID) || !same(t.OwnerUserID, arg.OwnerUserID) {
		return nil
	}
	t.SortOrder = arg.SortOrder
	s.templates[id(t.ID)] = t
	return nil
}

func (s *fakeStore) RecordTemplateUse(_ context.Context, arg db.RecordTemplateUseParams) (db.QuickTemplate, error) {
	t, ok := s.templates[id(arg.ID)]
	if !ok || !same(t.FamilyID, arg.FamilyID) || !same(t.OwnerUserID, arg.OwnerUserID) {
		return db.QuickTemplate{}, pgx.ErrNoRows
	}
	t.UsageCount++
	t.LastUsedAt = now()
	s.templates[id(t.ID)] = t
	return t, nil
}

/* ------------------------------------------------------------------ recurring */

// recurringVisible mirrors the JOIN in recurring.sql: a schedule is readable when the account
// it is attached to is readable.
func (s *fakeStore) recurringVisible(r db.RecurringPayment, viewer pgtype.UUID) bool {
	a, ok := s.accounts[id(r.AccountID)]
	return ok && s.visible(a, viewer)
}

func (s *fakeStore) ListVisibleRecurringPayments(_ context.Context, arg db.ListVisibleRecurringPaymentsParams) ([]db.RecurringPayment, error) {
	var out []db.RecurringPayment
	for _, r := range s.recurring {
		if !same(r.FamilyID, arg.FamilyID) || !s.recurringVisible(r, arg.ViewerMemberID) {
			continue
		}
		if !arg.IncludeInactive && !r.Active {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return id(out[i].ID) < id(out[j].ID) })
	return out, nil
}

func (s *fakeStore) GetVisibleRecurringPayment(_ context.Context, arg db.GetVisibleRecurringPaymentParams) (db.RecurringPayment, error) {
	r, ok := s.recurring[id(arg.ID)]
	if !ok || !same(r.FamilyID, arg.FamilyID) || !s.recurringVisible(r, arg.ViewerMemberID) {
		return db.RecurringPayment{}, pgx.ErrNoRows
	}
	return r, nil
}

func (s *fakeStore) CreateRecurringPayment(_ context.Context, arg db.CreateRecurringPaymentParams) (db.RecurringPayment, error) {
	r := db.RecurringPayment{
		ID: s.newUUID(), FamilyID: arg.FamilyID, Name: arg.Name, AmountMinor: arg.AmountMinor,
		CurrencyCode: arg.CurrencyCode, Type: arg.Type, CategoryID: arg.CategoryID,
		AccountID: arg.AccountID, MemberID: arg.MemberID, IntervalCount: arg.IntervalCount,
		IntervalUnit: arg.IntervalUnit, DayOfMonth: arg.DayOfMonth, DayOfWeek: arg.DayOfWeek,
		NextDueOn: arg.NextDueOn, EndOn: arg.EndOn, AutoPost: arg.AutoPost, Active: true,
		CreatedAt: now(), UpdatedAt: now(),
	}
	s.recurring[id(r.ID)] = r
	return r, nil
}

func (s *fakeStore) UpdateRecurringPayment(_ context.Context, arg db.UpdateRecurringPaymentParams) (db.RecurringPayment, error) {
	r, ok := s.recurring[id(arg.ID)]
	if !ok || !same(r.FamilyID, arg.FamilyID) {
		return db.RecurringPayment{}, pgx.ErrNoRows
	}
	if arg.Name != nil {
		r.Name = *arg.Name
	}
	if arg.AmountMinor != nil {
		r.AmountMinor = *arg.AmountMinor
	}
	if arg.CurrencyCode != nil {
		r.CurrencyCode = *arg.CurrencyCode
	}
	if arg.CategoryID.Valid {
		r.CategoryID = arg.CategoryID
	}
	if arg.AccountID.Valid {
		r.AccountID = arg.AccountID
	}
	if arg.MemberID.Valid {
		r.MemberID = arg.MemberID
	}
	if arg.IntervalCount != nil {
		r.IntervalCount = *arg.IntervalCount
	}
	if arg.IntervalUnit != nil {
		r.IntervalUnit = *arg.IntervalUnit
	}
	if arg.DayOfMonth != nil {
		r.DayOfMonth = *arg.DayOfMonth
	}
	if arg.DayOfWeek != nil {
		r.DayOfWeek = *arg.DayOfWeek
	}
	if arg.NextDueOn.Valid {
		r.NextDueOn = arg.NextDueOn
	}
	if arg.EndOn.Valid {
		r.EndOn = arg.EndOn
	}
	if arg.AutoPost != nil {
		r.AutoPost = *arg.AutoPost
	}
	if arg.Active != nil {
		r.Active = *arg.Active
	}
	s.recurring[id(r.ID)] = r
	return r, nil
}

func (s *fakeStore) AdvanceRecurringPayment(_ context.Context, arg db.AdvanceRecurringPaymentParams) (db.RecurringPayment, error) {
	r, ok := s.recurring[id(arg.ID)]
	if !ok || !same(r.FamilyID, arg.FamilyID) {
		return db.RecurringPayment{}, pgx.ErrNoRows
	}
	r.NextDueOn = arg.NextDueOn
	r.LastPostedOn = arg.LastPostedOn
	s.recurring[id(r.ID)] = r
	return r, nil
}

func (s *fakeStore) DeleteRecurringPayment(_ context.Context, arg db.DeleteRecurringPaymentParams) (int64, error) {
	r, ok := s.recurring[id(arg.ID)]
	if !ok || !same(r.FamilyID, arg.FamilyID) {
		return 0, nil
	}
	delete(s.recurring, id(arg.ID))
	return 1, nil
}

/* ------------------------------------------------------------------ reminders */

func (s *fakeStore) ListReminders(_ context.Context, arg db.ListRemindersParams) ([]db.Reminder, error) {
	var out []db.Reminder
	for _, r := range s.reminders {
		if !same(r.FamilyID, arg.FamilyID) || !same(r.UserID, arg.UserID) {
			continue
		}
		if !arg.IncludeDisabled && !r.Enabled {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return id(out[i].ID) < id(out[j].ID) })
	return out, nil
}

func (s *fakeStore) GetReminder(_ context.Context, arg db.GetReminderParams) (db.Reminder, error) {
	r, ok := s.reminders[id(arg.ID)]
	if !ok || !same(r.FamilyID, arg.FamilyID) || !same(r.UserID, arg.UserID) {
		return db.Reminder{}, pgx.ErrNoRows
	}
	return r, nil
}

func (s *fakeStore) CreateReminder(_ context.Context, arg db.CreateReminderParams) (db.Reminder, error) {
	r := db.Reminder{
		ID: s.newUUID(), FamilyID: arg.FamilyID, UserID: arg.UserID, Kind: arg.Kind,
		Title: arg.Title, DueAt: arg.DueAt, RepeatInterval: arg.RepeatInterval,
		RepeatUnit: arg.RepeatUnit, Enabled: arg.Enabled, CreatedAt: now(), UpdatedAt: now(),
	}
	s.reminders[id(r.ID)] = r
	return r, nil
}

func (s *fakeStore) UpdateReminder(_ context.Context, arg db.UpdateReminderParams) (db.Reminder, error) {
	r, ok := s.reminders[id(arg.ID)]
	if !ok || !same(r.FamilyID, arg.FamilyID) || !same(r.UserID, arg.UserID) {
		return db.Reminder{}, pgx.ErrNoRows
	}
	r.Kind = arg.Kind
	r.Title = arg.Title
	r.DueAt = arg.DueAt
	r.RepeatInterval = arg.RepeatInterval
	r.RepeatUnit = arg.RepeatUnit
	r.Enabled = arg.Enabled
	s.reminders[id(r.ID)] = r
	return r, nil
}

func (s *fakeStore) DeleteReminder(_ context.Context, arg db.DeleteReminderParams) (int64, error) {
	r, ok := s.reminders[id(arg.ID)]
	if !ok || !same(r.FamilyID, arg.FamilyID) || !same(r.UserID, arg.UserID) {
		return 0, nil
	}
	delete(s.reminders, id(arg.ID))
	return 1, nil
}

/* -------------------------------------------------------------------- widgets */

func (s *fakeStore) ListWidgets(_ context.Context, arg db.ListWidgetsParams) ([]db.WidgetInstance, error) {
	var out []db.WidgetInstance
	for _, w := range s.widgets {
		if same(w.FamilyID, arg.FamilyID) && same(w.UserID, arg.UserID) {
			out = append(out, w)
		}
	}
	sort.Slice(out, func(i, j int) bool { return id(out[i].ID) < id(out[j].ID) })
	return out, nil
}

func (s *fakeStore) ListWidgetsByIDs(_ context.Context, arg db.ListWidgetsByIDsParams) ([]db.WidgetInstance, error) {
	var out []db.WidgetInstance
	for _, w := range s.widgets {
		if same(w.FamilyID, arg.FamilyID) && same(w.UserID, arg.UserID) && matchesIDs(w.ID, arg.WidgetIds) {
			out = append(out, w)
		}
	}
	sort.Slice(out, func(i, j int) bool { return id(out[i].ID) < id(out[j].ID) })
	return out, nil
}

func (s *fakeStore) GetWidget(_ context.Context, arg db.GetWidgetParams) (db.WidgetInstance, error) {
	w, ok := s.widgets[id(arg.ID)]
	if !ok || !same(w.FamilyID, arg.FamilyID) || !same(w.UserID, arg.UserID) {
		return db.WidgetInstance{}, pgx.ErrNoRows
	}
	return w, nil
}

func (s *fakeStore) CreateWidget(_ context.Context, arg db.CreateWidgetParams) (db.WidgetInstance, error) {
	w := db.WidgetInstance{
		ID: s.newUUID(), FamilyID: arg.FamilyID, UserID: arg.UserID, Type: arg.Type,
		Size: arg.Size, ScopeKind: arg.ScopeKind, ScopeMemberID: arg.ScopeMemberID,
		ScopeAccountID: arg.ScopeAccountID, TargetRef: arg.TargetRef,
		TargetAccountIds: arg.TargetAccountIds, SortOrder: int32(len(s.widgets)),
		CreatedAt: now(), UpdatedAt: now(),
	}
	s.widgets[id(w.ID)] = w
	return w, nil
}

func (s *fakeStore) UpdateWidget(_ context.Context, arg db.UpdateWidgetParams) (db.WidgetInstance, error) {
	w, ok := s.widgets[id(arg.ID)]
	if !ok || !same(w.FamilyID, arg.FamilyID) || !same(w.UserID, arg.UserID) {
		return db.WidgetInstance{}, pgx.ErrNoRows
	}
	if arg.Size != nil {
		w.Size = *arg.Size
	}
	if arg.ScopeKind != nil {
		w.ScopeKind = *arg.ScopeKind
	}
	if arg.ScopeMemberID.Valid {
		w.ScopeMemberID = arg.ScopeMemberID
	}
	if arg.ScopeAccountID.Valid {
		w.ScopeAccountID = arg.ScopeAccountID
	}
	if arg.TargetRef.Valid {
		w.TargetRef = arg.TargetRef
	}
	if len(arg.TargetAccountIds) > 0 {
		w.TargetAccountIds = arg.TargetAccountIds
	}
	s.widgets[id(w.ID)] = w
	return w, nil
}

func (s *fakeStore) DeleteWidget(_ context.Context, arg db.DeleteWidgetParams) (int64, error) {
	w, ok := s.widgets[id(arg.ID)]
	if !ok || !same(w.FamilyID, arg.FamilyID) || !same(w.UserID, arg.UserID) {
		return 0, nil
	}
	delete(s.widgets, id(arg.ID))
	return 1, nil
}

/* ------------------------------------------------------------------- recorder */

// recorder captures published events so a test can assert on the subject rather than on a
// NATS server.
type recorder struct {
	subjects []string
	messages []proto.Message
}

func (r *recorder) Publish(_ context.Context, subject events.Subject, msg proto.Message) error {
	r.subjects = append(r.subjects, string(subject))
	r.messages = append(r.messages, msg)
	return nil
}

func (r *recorder) sawSubject(want string) bool {
	for _, s := range r.subjects {
		if s == want {
			return true
		}
	}
	return false
}

// errBoom is the injected failure used to reach a handler's internal-error path.
var errBoom = errors.New("boom")

// The fake is the Querier the handler is built against; this line is what fails the build if a
// query is added to a .sql file and not reflected here.
var _ db.Querier = (*fakeStore)(nil)
