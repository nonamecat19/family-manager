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

// fakeStore is an in-memory db.Querier. It reimplements the parts of the SQL the handler's
// behaviour depends on — family scoping, balance arithmetic, keyset ordering — so handler
// rules can be tested without Postgres. The SQL itself is verified by the migrations and by
// `sqlc generate` refusing to compile a bad query.
type fakeStore struct {
	accounts     map[string]db.Account
	categories   map[string]db.Category
	transactions map[string]db.Transaction

	failOn map[string]error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		accounts:     map[string]db.Account{},
		categories:   map[string]db.Category{},
		transactions: map[string]db.Transaction{},
		failOn:       map[string]error{},
	}
}

func (s *fakeStore) fail(op string) error { return s.failOn[op] }

func id(u pgtype.UUID) string { return pgconv.UUIDString(u) }

/* ----------------------------------------------------------------- accounts */

func (s *fakeStore) CreateAccount(_ context.Context, arg db.CreateAccountParams) (db.Account, error) {
	if err := s.fail("CreateAccount"); err != nil {
		return db.Account{}, err
	}
	a := db.Account{
		ID:                  pgconv.MustUUID(newUUID()),
		FamilyID:            arg.FamilyID,
		Name:                arg.Name,
		Type:                arg.Type,
		CurrencyCode:        arg.CurrencyCode,
		OpeningBalanceMinor: arg.OpeningBalanceMinor,
		Color:               arg.Color,
		Icon:                arg.Icon,
		SortOrder:           arg.SortOrder,
		CreatedAt:           pgtype.Timestamptz{Valid: true},
		UpdatedAt:           pgtype.Timestamptz{Valid: true},
	}
	s.accounts[id(a.ID)] = a
	return a, nil
}

func (s *fakeStore) GetAccount(_ context.Context, arg db.GetAccountParams) (db.Account, error) {
	a, ok := s.accounts[id(arg.ID)]
	if !ok || id(a.FamilyID) != id(arg.FamilyID) {
		return db.Account{}, pgx.ErrNoRows
	}
	return a, nil
}

// balanceOf mirrors the SQL: opening, minus everything that left, plus incoming transfers.
func (s *fakeStore) balanceOf(accountID pgtype.UUID) int64 {
	a := s.accounts[id(accountID)]
	balance := a.OpeningBalanceMinor
	for _, t := range s.transactions {
		if id(t.AccountID) == id(accountID) {
			if t.Type == typeIncome {
				balance += t.AmountMinor
			} else {
				balance -= t.AmountMinor
			}
		}
		if id(t.CounterAccountID) == id(accountID) {
			balance += t.AmountMinor
		}
	}
	return balance
}

func (s *fakeStore) ListAccountsWithBalance(
	_ context.Context, arg db.ListAccountsWithBalanceParams,
) ([]db.ListAccountsWithBalanceRow, error) {
	var out []db.ListAccountsWithBalanceRow
	for _, a := range s.accounts {
		if id(a.FamilyID) != id(arg.FamilyID) {
			continue
		}
		if a.Archived && !arg.IncludeArchived {
			continue
		}
		out = append(out, db.ListAccountsWithBalanceRow{
			ID: a.ID, FamilyID: a.FamilyID, Name: a.Name, Type: a.Type,
			CurrencyCode: a.CurrencyCode, OpeningBalanceMinor: a.OpeningBalanceMinor,
			Color: a.Color, Icon: a.Icon, Archived: a.Archived, SortOrder: a.SortOrder,
			CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
			BalanceMinor: s.balanceOf(a.ID),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *fakeStore) GetAccountWithBalance(
	_ context.Context, arg db.GetAccountWithBalanceParams,
) (db.GetAccountWithBalanceRow, error) {
	a, ok := s.accounts[id(arg.ID)]
	if !ok || id(a.FamilyID) != id(arg.FamilyID) {
		return db.GetAccountWithBalanceRow{}, pgx.ErrNoRows
	}
	return db.GetAccountWithBalanceRow{
		ID: a.ID, FamilyID: a.FamilyID, Name: a.Name, Type: a.Type,
		CurrencyCode: a.CurrencyCode, OpeningBalanceMinor: a.OpeningBalanceMinor,
		Color: a.Color, Icon: a.Icon, Archived: a.Archived, SortOrder: a.SortOrder,
		CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
		BalanceMinor: s.balanceOf(a.ID),
	}, nil
}

func (s *fakeStore) UpdateAccount(_ context.Context, arg db.UpdateAccountParams) (db.Account, error) {
	a, ok := s.accounts[id(arg.ID)]
	if !ok || id(a.FamilyID) != id(arg.FamilyID) {
		return db.Account{}, pgx.ErrNoRows
	}
	a.Name, a.Type, a.Color, a.Icon = arg.Name, arg.Type, arg.Color, arg.Icon
	a.Archived, a.SortOrder = arg.Archived, arg.SortOrder
	s.accounts[id(arg.ID)] = a
	return a, nil
}

func (s *fakeStore) DeleteAccount(_ context.Context, arg db.DeleteAccountParams) (int64, error) {
	a, ok := s.accounts[id(arg.ID)]
	if !ok || id(a.FamilyID) != id(arg.FamilyID) {
		return 0, nil
	}
	delete(s.accounts, id(arg.ID))
	return 1, nil
}

func (s *fakeStore) CountAccountTransactions(_ context.Context, accountID pgtype.UUID) (int64, error) {
	var n int64
	for _, t := range s.transactions {
		if id(t.AccountID) == id(accountID) || id(t.CounterAccountID) == id(accountID) {
			n++
		}
	}
	return n, nil
}

/* --------------------------------------------------------------- categories */

func (s *fakeStore) CreateCategory(
	_ context.Context, arg db.CreateCategoryParams,
) (db.Category, error) {
	if err := s.fail("CreateCategory"); err != nil {
		return db.Category{}, err
	}
	c := db.Category{
		ID:        pgconv.MustUUID(newUUID()),
		FamilyID:  arg.FamilyID,
		Name:      arg.Name,
		Kind:      arg.Kind,
		Color:     arg.Color,
		Icon:      arg.Icon,
		ParentID:  arg.ParentID,
		SortOrder: arg.SortOrder,
		CreatedAt: pgtype.Timestamptz{Valid: true},
		UpdatedAt: pgtype.Timestamptz{Valid: true},
	}
	s.categories[id(c.ID)] = c
	return c, nil
}

func (s *fakeStore) GetCategory(_ context.Context, arg db.GetCategoryParams) (db.Category, error) {
	c, ok := s.categories[id(arg.ID)]
	if !ok || id(c.FamilyID) != id(arg.FamilyID) {
		return db.Category{}, pgx.ErrNoRows
	}
	return c, nil
}

func (s *fakeStore) ListCategories(
	_ context.Context, arg db.ListCategoriesParams,
) ([]db.Category, error) {
	var out []db.Category
	for _, c := range s.categories {
		if id(c.FamilyID) != id(arg.FamilyID) {
			continue
		}
		if arg.Kind != "" && c.Kind != arg.Kind {
			continue
		}
		if c.Archived && !arg.IncludeArchived {
			continue
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *fakeStore) UpdateCategory(
	_ context.Context, arg db.UpdateCategoryParams,
) (db.Category, error) {
	c, ok := s.categories[id(arg.ID)]
	if !ok || id(c.FamilyID) != id(arg.FamilyID) {
		return db.Category{}, pgx.ErrNoRows
	}
	c.Name, c.Color, c.Icon, c.ParentID = arg.Name, arg.Color, arg.Icon, arg.ParentID
	c.Archived, c.SortOrder = arg.Archived, arg.SortOrder
	s.categories[id(arg.ID)] = c
	return c, nil
}

func (s *fakeStore) DeleteCategory(_ context.Context, arg db.DeleteCategoryParams) (int64, error) {
	c, ok := s.categories[id(arg.ID)]
	if !ok || id(c.FamilyID) != id(arg.FamilyID) {
		return 0, nil
	}
	delete(s.categories, id(arg.ID))
	return 1, nil
}

func (s *fakeStore) CountCategoryTransactions(
	_ context.Context, categoryID pgtype.UUID,
) (int64, error) {
	var n int64
	for _, t := range s.transactions {
		if id(t.CategoryID) == id(categoryID) {
			n++
		}
	}
	return n, nil
}

/* ------------------------------------------------------------- transactions */

func (s *fakeStore) CreateTransaction(
	_ context.Context, arg db.CreateTransactionParams,
) (db.Transaction, error) {
	if err := s.fail("CreateTransaction"); err != nil {
		return db.Transaction{}, err
	}
	t := db.Transaction{
		ID:               pgconv.MustUUID(newUUID()),
		FamilyID:         arg.FamilyID,
		AccountID:        arg.AccountID,
		CounterAccountID: arg.CounterAccountID,
		CategoryID:       arg.CategoryID,
		Type:             arg.Type,
		AmountMinor:      arg.AmountMinor,
		CurrencyCode:     arg.CurrencyCode,
		Note:             arg.Note,
		OccurredOn:       arg.OccurredOn,
		CreatedByUserID:  arg.CreatedByUserID,
		CreatedAt:        pgtype.Timestamptz{Valid: true},
		UpdatedAt:        pgtype.Timestamptz{Valid: true},
	}
	s.transactions[id(t.ID)] = t
	return t, nil
}

func (s *fakeStore) GetTransaction(
	_ context.Context, arg db.GetTransactionParams,
) (db.Transaction, error) {
	t, ok := s.transactions[id(arg.ID)]
	if !ok || id(t.FamilyID) != id(arg.FamilyID) {
		return db.Transaction{}, pgx.ErrNoRows
	}
	return t, nil
}

func (s *fakeStore) ListTransactions(
	_ context.Context, arg db.ListTransactionsParams,
) ([]db.Transaction, error) {
	var out []db.Transaction
	for _, t := range s.transactions {
		if id(t.FamilyID) != id(arg.FamilyID) {
			continue
		}
		day := pgconv.DateString(t.OccurredOn)
		if day < pgconv.DateString(arg.FromDate) || day > pgconv.DateString(arg.ToDate) {
			continue
		}
		if len(arg.AccountIds) > 0 && !containsID(arg.AccountIds, t.AccountID) &&
			!containsID(arg.AccountIds, t.CounterAccountID) {
			continue
		}
		if len(arg.CategoryIds) > 0 && !containsID(arg.CategoryIds, t.CategoryID) {
			continue
		}
		if arg.Type != "" && t.Type != arg.Type {
			continue
		}
		if arg.Search != "" && !strings.Contains(strings.ToLower(t.Note), strings.ToLower(arg.Search)) {
			continue
		}
		if arg.UseCursor {
			cursor := pgconv.DateString(arg.CursorDate) + "|" + id(arg.CursorID)
			if day+"|"+id(t.ID) >= cursor {
				continue
			}
		}
		out = append(out, t)
	}

	// (occurred_on, id) DESC, exactly like the query's ORDER BY.
	sort.Slice(out, func(i, j int) bool {
		a := pgconv.DateString(out[i].OccurredOn) + "|" + id(out[i].ID)
		b := pgconv.DateString(out[j].OccurredOn) + "|" + id(out[j].ID)
		return a > b
	})
	if int(arg.PageSize) < len(out) {
		out = out[:arg.PageSize]
	}
	return out, nil
}

func containsID(ids []pgtype.UUID, want pgtype.UUID) bool {
	for _, got := range ids {
		if id(got) == id(want) && want.Valid {
			return true
		}
	}
	return false
}

func (s *fakeStore) UpdateTransaction(
	_ context.Context, arg db.UpdateTransactionParams,
) (db.Transaction, error) {
	t, ok := s.transactions[id(arg.ID)]
	if !ok || id(t.FamilyID) != id(arg.FamilyID) {
		return db.Transaction{}, pgx.ErrNoRows
	}
	t.AccountID, t.CounterAccountID, t.CategoryID = arg.AccountID, arg.CounterAccountID, arg.CategoryID
	t.Type, t.AmountMinor, t.CurrencyCode = arg.Type, arg.AmountMinor, arg.CurrencyCode
	t.Note, t.OccurredOn = arg.Note, arg.OccurredOn
	s.transactions[id(arg.ID)] = t
	return t, nil
}

func (s *fakeStore) DeleteTransaction(
	_ context.Context, arg db.DeleteTransactionParams,
) (int64, error) {
	t, ok := s.transactions[id(arg.ID)]
	if !ok || id(t.FamilyID) != id(arg.FamilyID) {
		return 0, nil
	}
	delete(s.transactions, id(arg.ID))
	return 1, nil
}

func (s *fakeStore) GetSummary(
	_ context.Context, arg db.GetSummaryParams,
) (db.GetSummaryRow, error) {
	var row db.GetSummaryRow
	for _, t := range s.transactions {
		if id(t.FamilyID) != id(arg.FamilyID) {
			continue
		}
		day := pgconv.DateString(t.OccurredOn)
		if day < pgconv.DateString(arg.FromDate) || day > pgconv.DateString(arg.ToDate) {
			continue
		}
		if len(arg.AccountIds) > 0 && !containsID(arg.AccountIds, t.AccountID) {
			continue
		}
		switch t.Type {
		case typeIncome:
			row.IncomeMinor += t.AmountMinor
		case typeExpense:
			row.ExpenseMinor += t.AmountMinor
		}
	}
	return row, nil
}

func (s *fakeStore) GetCategoryBreakdown(
	_ context.Context, arg db.GetCategoryBreakdownParams,
) ([]db.GetCategoryBreakdownRow, error) {
	totals := map[string]*db.GetCategoryBreakdownRow{}
	for _, t := range s.transactions {
		if id(t.FamilyID) != id(arg.FamilyID) || t.Type != arg.Type {
			continue
		}
		day := pgconv.DateString(t.OccurredOn)
		if day < pgconv.DateString(arg.FromDate) || day > pgconv.DateString(arg.ToDate) {
			continue
		}
		if len(arg.AccountIds) > 0 && !containsID(arg.AccountIds, t.AccountID) {
			continue
		}
		cat, ok := s.categories[id(t.CategoryID)]
		if !ok {
			continue // JOIN categories drops uncategorised rows
		}
		row, seen := totals[id(cat.ID)]
		if !seen {
			row = &db.GetCategoryBreakdownRow{
				CategoryID: cat.ID, CategoryName: cat.Name, Color: cat.Color,
			}
			totals[id(cat.ID)] = row
		}
		row.TotalMinor += t.AmountMinor
		row.TransactionCount++
	}

	out := make([]db.GetCategoryBreakdownRow, 0, len(totals))
	for _, row := range totals {
		out = append(out, *row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TotalMinor > out[j].TotalMinor })
	return out, nil
}

/* -------------------------------------------------------------------- other */

type recorder struct {
	published []events.Subject
	err       error
}

func (r *recorder) Publish(_ context.Context, subject events.Subject, _ proto.Message) error {
	if r.err != nil {
		return r.err
	}
	r.published = append(r.published, subject)
	return nil
}

func (r *recorder) sawSubject(s events.Subject) bool {
	for _, got := range r.published {
		if got == s {
			return true
		}
	}
	return false
}

var errBoom = errors.New("boom")

// newUUID hands out deterministic, valid v4-shaped ids so failures are reproducible.
var uuidCounter int

func newUUID() string {
	uuidCounter++
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", uuidCounter)
}
