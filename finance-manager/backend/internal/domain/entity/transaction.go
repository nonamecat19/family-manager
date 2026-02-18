package entity

import "time"

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypeIncome   TransactionType = "income"
	TransactionTypeExpense  TransactionType = "expense"
	TransactionTypeTransfer TransactionType = "transfer"
)

// Transaction represents a financial transaction
type Transaction struct {
	ID          int             `json:"id"`
	WalletID    int             `json:"walletId"`
	Type        TransactionType `json:"type"`
	Amount      float64         `json:"amount"`
	Currency    string          `json:"currency"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	CreatedAt   time.Time       `json:"createdAt"`
	ExecutedAt  *time.Time      `json:"executedAt"`
}
