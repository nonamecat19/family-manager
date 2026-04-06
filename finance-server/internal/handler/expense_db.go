package handler

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nnc/finance-tracker/server/internal/db/sqlc"
)

// PgExpenseDB implements ExpenseDB using sqlc-generated queries against PostgreSQL.
type PgExpenseDB struct {
	queries *sqlc.Queries
}

// NewPgExpenseDB creates a PgExpenseDB wrapping sqlc.Queries.
func NewPgExpenseDB(queries *sqlc.Queries) *PgExpenseDB {
	return &PgExpenseDB{queries: queries}
}

func (db *PgExpenseDB) CreateExpense(userID, categoryID string, amountCents int64, note string, expenseDate time.Time) (MockExpense, error) {
	uid := stringToUUID(userID)
	cid := stringToUUID(categoryID)

	dateVal := pgtype.Date{
		Time:  expenseDate,
		Valid: true,
	}

	row, err := db.queries.CreateExpense(context.Background(), sqlc.CreateExpenseParams{
		UserID:      uid,
		CategoryID:  cid,
		AmountCents: amountCents,
		Note:        note,
		ExpenseDate: dateVal,
	})
	if err != nil {
		return MockExpense{}, err
	}

	return MockExpense{
		ID:          uuidToString(row.ID),
		UserID:      uuidToString(row.UserID),
		CategoryID:  uuidToString(row.CategoryID),
		AmountCents: row.AmountCents,
		Note:        row.Note,
		ExpenseDate: row.ExpenseDate.Time,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}

func (db *PgExpenseDB) GetExpensesByUser(userID string, limit, offset int) ([]MockExpense, error) {
	uid := stringToUUID(userID)

	rows, err := db.queries.GetExpensesByUser(context.Background(), sqlc.GetExpensesByUserParams{
		UserID: uid,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}

	expenses := make([]MockExpense, len(rows))
	for i, row := range rows {
		expenses[i] = MockExpense{
			ID:          uuidToString(row.ID),
			UserID:      uuidToString(row.UserID),
			CategoryID:  uuidToString(row.CategoryID),
			AmountCents: row.AmountCents,
			Note:        row.Note,
			ExpenseDate: row.ExpenseDate.Time,
			CreatedAt:   row.CreatedAt.Time,
			UpdatedAt:   row.UpdatedAt.Time,
		}
	}
	return expenses, nil
}

func (db *PgExpenseDB) UpdateExpense(id, userID, categoryID string, amountCents int64, note string, expenseDate time.Time) (MockExpense, error) {
	uid := stringToUUID(id)
	uidUser := stringToUUID(userID)
	cid := stringToUUID(categoryID)

	dateVal := pgtype.Date{
		Time:  expenseDate,
		Valid: true,
	}

	row, err := db.queries.UpdateExpense(context.Background(), sqlc.UpdateExpenseParams{
		ID:          uid,
		UserID:      uidUser,
		CategoryID:  cid,
		AmountCents: amountCents,
		Note:        note,
		ExpenseDate: dateVal,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return MockExpense{}, ErrExpenseNotFound
		}
		return MockExpense{}, err
	}

	return MockExpense{
		ID:          uuidToString(row.ID),
		UserID:      uuidToString(row.UserID),
		CategoryID:  uuidToString(row.CategoryID),
		AmountCents: row.AmountCents,
		Note:        row.Note,
		ExpenseDate: row.ExpenseDate.Time,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}

func dateToPgDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func stringToNullableUUID(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{Valid: false}
	}
	return stringToUUID(s)
}

func (db *PgExpenseDB) GetExpensesByUserFiltered(userID string, limit, offset int, dateFrom, dateTo *time.Time, categoryID string) ([]MockExpense, error) {
	uid := stringToUUID(userID)

	rows, err := db.queries.GetExpensesByUserFiltered(context.Background(), sqlc.GetExpensesByUserFilteredParams{
		UserID:     uid,
		Limit:      int32(limit),
		Offset:     int32(offset),
		DateFrom:   dateToPgDate(dateFrom),
		DateTo:     dateToPgDate(dateTo),
		CategoryID: stringToNullableUUID(categoryID),
	})
	if err != nil {
		return nil, err
	}

	expenses := make([]MockExpense, len(rows))
	for i, row := range rows {
		expenses[i] = MockExpense{
			ID:          uuidToString(row.ID),
			UserID:      uuidToString(row.UserID),
			CategoryID:  uuidToString(row.CategoryID),
			AmountCents: row.AmountCents,
			Note:        row.Note,
			ExpenseDate: row.ExpenseDate.Time,
			CreatedAt:   row.CreatedAt.Time,
			UpdatedAt:   row.UpdatedAt.Time,
		}
	}
	return expenses, nil
}

func (db *PgExpenseDB) DeleteExpense(id, userID string) error {
	uid := stringToUUID(id)
	uidUser := stringToUUID(userID)

	rowsAffected, err := db.queries.DeleteExpense(context.Background(), sqlc.DeleteExpenseParams{
		ID:     uid,
		UserID: uidUser,
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrExpenseNotFound
	}
	return nil
}
