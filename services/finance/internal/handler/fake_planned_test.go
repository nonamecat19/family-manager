package handler

import (
	"context"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/nnc/family-manager/services/finance/db"
)

func (s *fakeStore) GetCategoryGroupByRole(_ context.Context, arg db.GetCategoryGroupByRoleParams) (db.CategoryGroup, error) {
	for _, g := range s.groups {
		if same(g.FamilyID, arg.FamilyID) && g.Role == arg.Role {
			return g, nil
		}
	}
	return db.CategoryGroup{}, pgx.ErrNoRows
}

func (s *fakeStore) AdoptCategoryGroupRole(_ context.Context, arg db.AdoptCategoryGroupRoleParams) (db.CategoryGroup, error) {
	for key, g := range s.groups {
		if same(g.FamilyID, arg.FamilyID) && g.Kind == kindExpense && g.Role == "" &&
			strings.EqualFold(strings.TrimSpace(g.Name), strings.TrimSpace(arg.Name)) {
			g.Role = arg.Role
			g.Archived = false
			s.groups[key] = g
			return g, nil
		}
	}
	return db.CategoryGroup{}, pgx.ErrNoRows
}

func (s *fakeStore) CreateRoleCategoryGroup(ctx context.Context, arg db.CreateRoleCategoryGroupParams) (db.CategoryGroup, error) {
	if _, err := s.GetCategoryGroupByRole(ctx, db.GetCategoryGroupByRoleParams{
		FamilyID: arg.FamilyID, Role: arg.Role,
	}); err == nil {
		return db.CategoryGroup{}, pgx.ErrNoRows
	}
	g := db.CategoryGroup{
		ID: s.newUUID(), FamilyID: arg.FamilyID, Name: arg.Name, Kind: kindExpense,
		Icon: arg.Icon, ColorStep: arg.ColorStep, Role: arg.Role, SortOrder: int32(len(s.groups)),
		CreatedAt: now(), UpdatedAt: now(),
	}
	s.groups[id(g.ID)] = g
	return g, nil
}

func (s *fakeStore) SetCategoryArchived(_ context.Context, arg db.SetCategoryArchivedParams) error {
	c, ok := s.categories[id(arg.ID)]
	if !ok || !same(c.FamilyID, arg.FamilyID) {
		return nil
	}
	c.Archived = arg.Archived
	s.categories[id(c.ID)] = c
	return nil
}

func (s *fakeStore) categorySpend(familyID, categoryID string, currency string, viewer *string) (int64, int64) {
	var sum, count int64
	for _, t := range s.transactions {
		if id(t.FamilyID) != familyID || id(t.CategoryID) != categoryID || t.Type != kindExpense ||
			t.CurrencyCode != currency {
			continue
		}
		if viewer != nil {
			a, ok := s.accounts[id(t.AccountID)]
			if !ok || (a.Visibility != visibilityShared && id(a.OwnerMemberID) != *viewer) {
				continue
			}
		}
		sum += t.AmountMinor
		count++
	}
	return sum, count
}

func (s *fakeStore) ListInvestments(_ context.Context, arg db.ListInvestmentsParams) ([]db.ListInvestmentsRow, error) {
	viewer := id(arg.ViewerMemberID)
	var out []db.ListInvestmentsRow
	for _, i := range s.investments {
		if !same(i.FamilyID, arg.FamilyID) || (i.Archived && !arg.IncludeArchived) {
			continue
		}
		invested, _ := s.categorySpend(id(i.FamilyID), id(i.CategoryID), i.CurrencyCode, &viewer)
		out = append(out, db.ListInvestmentsRow{
			ID: i.ID, FamilyID: i.FamilyID, Name: i.Name, Kind: i.Kind, CurrencyCode: i.CurrencyCode,
			CategoryID: i.CategoryID, CurrentValueMinor: i.CurrentValueMinor,
			ValueUpdatedOn: i.ValueUpdatedOn, Archived: i.Archived, SortOrder: i.SortOrder,
			CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt,
			GroupID: s.groupOf(i.CategoryID), InvestedMinor: invested,
		})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].SortOrder < out[b].SortOrder })
	return out, nil
}

func (s *fakeStore) GetInvestment(_ context.Context, arg db.GetInvestmentParams) (db.Investment, error) {
	i, ok := s.investments[id(arg.ID)]
	if !ok || !same(i.FamilyID, arg.FamilyID) {
		return db.Investment{}, pgx.ErrNoRows
	}
	return i, nil
}

func (s *fakeStore) CreateInvestment(_ context.Context, arg db.CreateInvestmentParams) (db.Investment, error) {
	for _, i := range s.investments {
		if same(i.FamilyID, arg.FamilyID) && strings.EqualFold(i.Name, arg.Name) {
			return db.Investment{}, &pgconn.PgError{Code: pgUniqueViolation}
		}
	}
	i := db.Investment{
		ID: s.newUUID(), FamilyID: arg.FamilyID, Name: arg.Name, Kind: arg.Kind,
		CurrencyCode: arg.CurrencyCode, CategoryID: arg.CategoryID,
		SortOrder: int32(len(s.investments)), CreatedAt: now(), UpdatedAt: now(),
	}
	s.investments[id(i.ID)] = i
	return i, nil
}

func (s *fakeStore) UpdateInvestment(_ context.Context, arg db.UpdateInvestmentParams) (db.Investment, error) {
	i, ok := s.investments[id(arg.ID)]
	if !ok || !same(i.FamilyID, arg.FamilyID) {
		return db.Investment{}, pgx.ErrNoRows
	}
	if arg.Name != nil {
		i.Name = *arg.Name
	}
	if arg.Kind != nil {
		i.Kind = *arg.Kind
	}
	if arg.Archived != nil {
		i.Archived = *arg.Archived
	}
	s.investments[id(i.ID)] = i
	return i, nil
}

func (s *fakeStore) SetInvestmentValue(_ context.Context, arg db.SetInvestmentValueParams) (db.Investment, error) {
	i, ok := s.investments[id(arg.ID)]
	if !ok || !same(i.FamilyID, arg.FamilyID) {
		return db.Investment{}, pgx.ErrNoRows
	}
	i.CurrentValueMinor = arg.CurrentValueMinor
	i.ValueUpdatedOn = arg.ValueUpdatedOn
	s.investments[id(i.ID)] = i
	return i, nil
}

func (s *fakeStore) DeleteInvestment(_ context.Context, arg db.DeleteInvestmentParams) (int64, error) {
	i, ok := s.investments[id(arg.ID)]
	if !ok || !same(i.FamilyID, arg.FamilyID) {
		return 0, nil
	}
	delete(s.investments, id(arg.ID))
	return 1, nil
}

func (s *fakeStore) installmentVisible(i db.Installment, viewer string) bool {
	a, ok := s.accounts[id(i.AccountID)]
	return ok && (a.Visibility == visibilityShared || id(a.OwnerMemberID) == viewer)
}

func (s *fakeStore) ListVisibleInstallments(_ context.Context, arg db.ListVisibleInstallmentsParams) ([]db.ListVisibleInstallmentsRow, error) {
	viewer := id(arg.ViewerMemberID)
	var out []db.ListVisibleInstallmentsRow
	for _, i := range s.installments {
		if !same(i.FamilyID, arg.FamilyID) || !s.installmentVisible(i, viewer) {
			continue
		}
		if !arg.IncludeClosed && i.Status != installmentActive {
			continue
		}
		paid, payments := s.categorySpend(id(i.FamilyID), id(i.CategoryID), i.CurrencyCode, &viewer)
		out = append(out, db.ListVisibleInstallmentsRow{
			Installment: i, GroupID: s.groupOf(i.CategoryID), PaidMinor: paid, Payments: payments,
		})
	}
	sort.Slice(out, func(a, b int) bool {
		return out[a].Installment.NextDueOn.Time.Before(out[b].Installment.NextDueOn.Time)
	})
	return out, nil
}

func (s *fakeStore) GetVisibleInstallment(_ context.Context, arg db.GetVisibleInstallmentParams) (db.Installment, error) {
	i, ok := s.installments[id(arg.ID)]
	if !ok || !same(i.FamilyID, arg.FamilyID) || !s.installmentVisible(i, id(arg.ViewerMemberID)) {
		return db.Installment{}, pgx.ErrNoRows
	}
	return i, nil
}

func (s *fakeStore) CreateInstallment(_ context.Context, arg db.CreateInstallmentParams) (db.Installment, error) {
	i := db.Installment{
		ID: s.newUUID(), FamilyID: arg.FamilyID, Name: arg.Name, TotalMinor: arg.TotalMinor,
		MonthlyMinor: arg.MonthlyMinor, Months: arg.Months, CurrencyCode: arg.CurrencyCode,
		AccountID: arg.AccountID, CategoryID: arg.CategoryID, MemberID: arg.MemberID,
		CreatedByUserID: arg.CreatedByUserID, PurchasedOn: arg.PurchasedOn,
		DayOfMonth: arg.DayOfMonth, NextDueOn: arg.NextDueOn, Status: installmentActive,
		CreatedAt: now(), UpdatedAt: now(),
	}
	s.installments[id(i.ID)] = i
	return i, nil
}

func (s *fakeStore) UpdateInstallment(_ context.Context, arg db.UpdateInstallmentParams) (db.Installment, error) {
	i, ok := s.installments[id(arg.ID)]
	if !ok || !same(i.FamilyID, arg.FamilyID) {
		return db.Installment{}, pgx.ErrNoRows
	}
	if arg.Name != nil {
		i.Name = *arg.Name
	}
	if arg.MonthlyMinor != nil {
		i.MonthlyMinor = *arg.MonthlyMinor
	}
	if arg.AccountID.Valid {
		i.AccountID = arg.AccountID
	}
	if arg.MemberID.Valid {
		i.MemberID = arg.MemberID
	}
	if arg.DayOfMonth != nil {
		i.DayOfMonth = *arg.DayOfMonth
	}
	if arg.NextDueOn.Valid {
		i.NextDueOn = arg.NextDueOn
	}
	if arg.Status != nil {
		i.Status = *arg.Status
	}
	s.installments[id(i.ID)] = i
	return i, nil
}

func (s *fakeStore) DeleteInstallment(_ context.Context, arg db.DeleteInstallmentParams) (int64, error) {
	i, ok := s.installments[id(arg.ID)]
	if !ok || !same(i.FamilyID, arg.FamilyID) {
		return 0, nil
	}
	delete(s.installments, id(arg.ID))
	return 1, nil
}

func (s *fakeStore) ListDueInstallments(_ context.Context, arg db.ListDueInstallmentsParams) ([]db.ListDueInstallmentsRow, error) {
	var out []db.ListDueInstallmentsRow
	for _, i := range s.installments {
		if i.Status != installmentActive || i.NextDueOn.Time.After(arg.LatestDueOn.Time) {
			continue
		}
		out = append(out, db.ListDueInstallmentsRow{
			ID: i.ID, FamilyID: i.FamilyID, CategoryID: i.CategoryID, NextDueOn: i.NextDueOn,
		})
	}
	return out, nil
}

func (s *fakeStore) GetInstallmentByID(_ context.Context, installmentID pgtype.UUID) (db.Installment, error) {
	i, ok := s.installments[id(installmentID)]
	if !ok {
		return db.Installment{}, pgx.ErrNoRows
	}
	return i, nil
}

func (s *fakeStore) LockDueInstallment(_ context.Context, installmentID pgtype.UUID) (db.Installment, error) {
	i, ok := s.installments[id(installmentID)]
	if !ok || i.Status != installmentActive {
		return db.Installment{}, pgx.ErrNoRows
	}
	return i, nil
}

func (s *fakeStore) SumInstallmentPayments(_ context.Context, arg db.SumInstallmentPaymentsParams) (db.SumInstallmentPaymentsRow, error) {
	paid, payments := s.categorySpend(id(arg.FamilyID), id(arg.CategoryID), arg.CurrencyCode, nil)
	return db.SumInstallmentPaymentsRow{PaidMinor: paid, Payments: payments}, nil
}

func (s *fakeStore) AdvanceInstallment(_ context.Context, arg db.AdvanceInstallmentParams) (db.Installment, error) {
	i, ok := s.installments[id(arg.ID)]
	if !ok || i.Status != installmentActive || !i.NextDueOn.Time.Equal(arg.ExpectedDueOn.Time) {
		return db.Installment{}, pgx.ErrNoRows
	}
	i.NextDueOn = arg.NextDueOn
	i.Status = arg.Status
	s.installments[id(i.ID)] = i
	return i, nil
}

func (s *fakeStore) IsCategoryOwned(_ context.Context, categoryID pgtype.UUID) (bool, error) {
	for _, i := range s.investments {
		if same(i.CategoryID, categoryID) {
			return true, nil
		}
	}
	for _, i := range s.installments {
		if same(i.CategoryID, categoryID) {
			return true, nil
		}
	}
	return false, nil
}
