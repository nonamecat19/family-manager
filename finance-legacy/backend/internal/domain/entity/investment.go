package entity

import "time"

// InvestmentType represents the type of investment
type InvestmentType string

const (
	InvestmentTypeCryptoStake InvestmentType = "crypto_stake"
	InvestmentTypeOther       InvestmentType = "other"
)

// InvestmentStatus represents the status of an investment
type InvestmentStatus string

const (
	InvestmentStatusActive    InvestmentStatus = "active"
	InvestmentStatusCompleted InvestmentStatus = "completed"
)

// Investment represents an investment such as crypto staking
type Investment struct {
	ID          int              `json:"id"`
	UserID      int              `json:"userId"`
	WalletID    int              `json:"walletId"`
	Type        InvestmentType   `json:"type"`
	Coin        string           `json:"coin"`
	Amount      float64          `json:"amount"`
	StartDate   time.Time        `json:"startDate"`
	EndDate     *time.Time       `json:"endDate"`
	DailyReward float64          `json:"dailyReward"`
	Status      InvestmentStatus `json:"status"`
}
