package repository

import (
	"time"

	"github.com/finance-manager/backend/internal/domain/entity"
)

// RecurringRepository defines the interface for recurring payment data operations
type RecurringRepository interface {
	Create(recurring *entity.Recurring) error
	GetByID(id int) (*entity.Recurring, error)
	GetByUserID(userID int) ([]*entity.Recurring, error)
	GetDueRecurring() ([]*entity.Recurring, error)
	Update(recurring *entity.Recurring) error
	UpdateNextExecution(id int, nextExecution *time.Time) error
	Delete(id int) error
}
