package entity

import "time"

// WalletType represents the type of wallet
type WalletType string

const (
	WalletTypeCash       WalletType = "cash"
	WalletTypeCreditCard WalletType = "credit_card"
	WalletTypeCrypto     WalletType = "crypto"
)

// Wallet represents a wallet/account in the system
type Wallet struct {
	ID        int        `json:"id"`
	UserID    int        `json:"userId"`
	Name      string     `json:"name"`
	Type      WalletType `json:"type"`
	Currency  string     `json:"currency"`
	Balance   float64    `json:"balance"`
	Category  *string    `json:"category,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}
