package handler

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nnc/finance-tracker/server/internal/db/sqlc"
)

// PgSummaryDB implements SummaryDB using sqlc-generated queries against PostgreSQL.
type PgSummaryDB struct {
	queries *sqlc.Queries
}

// NewPgSummaryDB creates a PgSummaryDB wrapping sqlc.Queries.
func NewPgSummaryDB(queries *sqlc.Queries) *PgSummaryDB {
	return &PgSummaryDB{queries: queries}
}

func (db *PgSummaryDB) GetCategoryTotals(userID string, dateFrom, dateTo time.Time) ([]CategoryTotal, error) {
	uid := stringToUUID(userID)

	rows, err := db.queries.GetCategoryTotals(context.Background(), sqlc.GetCategoryTotalsParams{
		UserID:        uid,
		ExpenseDate:   pgtype.Date{Time: dateFrom, Valid: true},
		ExpenseDate_2: pgtype.Date{Time: dateTo, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	totals := make([]CategoryTotal, len(rows))
	for i, row := range rows {
		totals[i] = CategoryTotal{
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

func (db *PgSummaryDB) GetDailyTotals(userID string, dateFrom, dateTo time.Time) ([]DateTotal, error) {
	uid := stringToUUID(userID)

	rows, err := db.queries.GetDailyTotals(context.Background(), sqlc.GetDailyTotalsParams{
		UserID:        uid,
		ExpenseDate:   pgtype.Date{Time: dateFrom, Valid: true},
		ExpenseDate_2: pgtype.Date{Time: dateTo, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	totals := make([]DateTotal, len(rows))
	for i, row := range rows {
		totals[i] = DateTotal{
			Date:       row.Date.Time.Format("2006-01-02"),
			TotalCents: row.TotalCents,
		}
	}
	return totals, nil
}
