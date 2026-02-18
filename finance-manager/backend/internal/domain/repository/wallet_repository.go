package repository

import "github.com/finance-manager/backend/internal/domain/entity"

// WalletRepository defines the interface for wallet data operations
type WalletRepository interface {
	Create(wallet *entity.Wallet) error
	GetByID(id int) (*entity.Wallet, error)
	GetByUserID(userID int) ([]*entity.Wallet, error)
	Update(wallet *entity.Wallet) error
	Delete(id int) error
	UpdateBalance(id int, balance float64) error
	UpdateCategory(id int, category *string) error
}
