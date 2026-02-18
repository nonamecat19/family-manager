package entity

import "time"

// RecurringType represents the type of recurring transaction
type RecurringType string

const (
	RecurringTypeIncome  RecurringType = "income"
	RecurringTypeExpense RecurringType = "expense"
)

// RecurrencePattern represents the frequency pattern
type RecurrencePattern string

const (
	RecurrenceDaily   RecurrencePattern = "daily"
	RecurrenceWeekly  RecurrencePattern = "weekly"
	RecurrenceMonthly RecurrencePattern = "monthly"
)

// Recurring represents a recurring payment or income
type Recurring struct {
	ID            int               `json:"id"`
	UserID        int               `json:"userId"`
	WalletID      int               `json:"walletId"`
	Type          RecurringType     `json:"type"`
	Amount        float64           `json:"amount"`
	Currency      string            `json:"currency"`
	Description   string            `json:"description"`
	Recurrence    RecurrencePattern `json:"recurrence"`
	StartDate     time.Time         `json:"startDate"`
	EndDate       *time.Time        `json:"endDate"`
	NextExecution *time.Time        `json:"nextExecution"`
}
