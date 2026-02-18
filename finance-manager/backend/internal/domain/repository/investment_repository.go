package repository

import "github.com/finance-manager/backend/internal/domain/entity"

// InvestmentRepository defines the interface for investment data operations
type InvestmentRepository interface {
	Create(investment *entity.Investment) error
	GetByID(id int) (*entity.Investment, error)
	GetByUserID(userID int) ([]*entity.Investment, error)
	Update(investment *entity.Investment) error
	Delete(id int) error
}
