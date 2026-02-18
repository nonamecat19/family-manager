package repository

import "github.com/finance-manager/backend/internal/domain/entity"

// UserRepository defines the interface for user data operations
type UserRepository interface {
	Create(user *entity.User) error
	GetByID(id int) (*entity.User, error)
	GetByEmail(email string) (*entity.User, error)
	Update(user *entity.User) error
	Delete(id int) error
}
