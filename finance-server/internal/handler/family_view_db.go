package handler

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nnc/finance-tracker/server/internal/db/sqlc"
)

// PgFamilyViewDB implements FamilyViewDB using sqlc-generated queries against PostgreSQL.
type PgFamilyViewDB struct {
	queries *sqlc.Queries
}

// NewPgFamilyViewDB creates a PgFamilyViewDB wrapping sqlc.Queries.
func NewPgFamilyViewDB(queries *sqlc.Queries) *PgFamilyViewDB {
	return &PgFamilyViewDB{queries: queries}
}

func (db *PgFamilyViewDB) GetFamilyExpenses(familyID string, limit, offset int) ([]FamilyExpense, error) {
	fid := stringToUUID(familyID)

	rows, err := db.queries.GetFamilyExpenses(context.Background(), sqlc.GetFamilyExpensesParams{
		FamilyID: fid,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}

	expenses := make([]FamilyExpense, len(rows))
	for i, row := range rows {
		expenses[i] = FamilyExpense{
			ID:            uuidToString(row.ID),
			UserID:        uuidToString(row.UserID),
			UserEmail:     row.UserEmail,
			CategoryID:    uuidToString(row.CategoryID),
			CategoryName:  row.CategoryName,
			CategoryColor: row.CategoryColor,
			CategoryIcon:  row.CategoryIcon,
			AmountCents:   row.AmountCents,
			Note:          row.Note,
			ExpenseDate:   row.ExpenseDate.Time,
			CreatedAt:     row.CreatedAt.Time,
		}
	}
	return expenses, nil
}

func (db *PgFamilyViewDB) GetFamilyMemberTotals(familyID string, dateFrom, dateTo time.Time) ([]FamilyMemberTotal, error) {
	fid := stringToUUID(familyID)

	rows, err := db.queries.GetFamilyMemberTotals(context.Background(), sqlc.GetFamilyMemberTotalsParams{
		FamilyID:      fid,
		ExpenseDate:   pgtype.Date{Time: dateFrom, Valid: true},
		ExpenseDate_2: pgtype.Date{Time: dateTo, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	totals := make([]FamilyMemberTotal, len(rows))
	for i, row := range rows {
		totals[i] = FamilyMemberTotal{
			UserID:     uuidToString(row.UserID),
			UserEmail:  row.UserEmail,
			TotalCents: row.TotalCents,
			Count:      int(row.ExpenseCount),
		}
	}
	return totals, nil
}

func (db *PgFamilyViewDB) GetFamilyCategoryTotals(familyID string, dateFrom, dateTo time.Time) ([]FamilyCategoryTotal, error) {
	fid := stringToUUID(familyID)

	rows, err := db.queries.GetFamilyCategoryTotals(context.Background(), sqlc.GetFamilyCategoryTotalsParams{
		FamilyID:      fid,
		ExpenseDate:   pgtype.Date{Time: dateFrom, Valid: true},
		ExpenseDate_2: pgtype.Date{Time: dateTo, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	totals := make([]FamilyCategoryTotal, len(rows))
	for i, row := range rows {
		totals[i] = FamilyCategoryTotal{
			CategoryID:    uuidToString(row.CategoryID),
			CategoryName:  row.CategoryName,
			CategoryColor: row.CategoryColor,
			CategoryIcon:  row.CategoryIcon,
			TotalCents:    row.TotalCents,
			Count:         int(row.ExpenseCount),
		}
	}
	return totals, nil
}
