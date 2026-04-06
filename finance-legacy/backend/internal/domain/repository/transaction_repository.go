package repository

import (
	"time"

	"github.com/finance-manager/backend/internal/domain/entity"
)

// TransactionRepository defines the interface for transaction data operations
type TransactionRepository interface {
	Create(transaction *entity.Transaction) error
	GetByID(id int) (*entity.Transaction, error)
	GetByWalletID(walletID int, from, to *time.Time) ([]*entity.Transaction, error)
	Update(transaction *entity.Transaction) error
	Delete(id int) error
}
