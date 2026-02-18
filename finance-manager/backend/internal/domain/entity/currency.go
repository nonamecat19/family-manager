package entity

// CurrencyType represents the type of currency
type CurrencyType string

const (
	CurrencyTypeFIAT   CurrencyType = "fiat"
	CurrencyTypeCrypto CurrencyType = "crypto"
)

// Currency represents a currency in the system
type Currency struct {
	ID           int         `json:"id"`
	Code         string      `json:"code"`         // USD, UAH, USDT, USDC, ETH
	Type         CurrencyType `json:"type"`        // fiat or crypto
	ExchangeRate float64     `json:"exchangeRate"` // USD per 1 unit of currency (1.0 for USD). Format: 1 currency = exchangeRate USD
	Name         string      `json:"name"`        // Full name: US Dollar, Ukrainian Hryvnia, etc.
}

