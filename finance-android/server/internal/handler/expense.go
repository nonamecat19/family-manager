package handler

import (
	"time"
)

// MockExpense is the expense representation used by the ExpenseDB interface.
type MockExpense struct {
	ID          string
	UserID      string
	CategoryID  string
	AmountCents int64
	Note        string
	ExpenseDate time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ExpenseDB abstracts database operations for expenses.
// This allows testing with mock implementations.
type ExpenseDB interface {
	CreateExpense(userID, categoryID string, amountCents int64, note string, expenseDate time.Time) (MockExpense, error)
	GetExpensesByUser(userID string, limit, offset int) ([]MockExpense, error)
}

// ExpenseHandler handles expense HTTP requests.
type ExpenseHandler struct {
	db ExpenseDB
}

// NewExpenseHandler creates an ExpenseHandler with the given database.
func NewExpenseHandler(db ExpenseDB) *ExpenseHandler {
	return &ExpenseHandler{db: db}
}
