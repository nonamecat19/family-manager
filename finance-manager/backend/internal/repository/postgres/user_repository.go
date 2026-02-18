package postgres

import (
	"context"

	"github.com/finance-manager/backend/internal/domain/entity"
	"github.com/finance-manager/backend/internal/domain/repository"
)

type userRepository struct {
	queries *Queries
}

func NewUserRepository(queries *Queries) repository.UserRepository {
	return &userRepository{queries: queries}
}

func (r *userRepository) Create(user *entity.User) error {
	ctx := context.Background()
	
	params := CreateUserParams{
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
	}
	
	dbUser, err := r.queries.CreateUser(ctx, params)
	if err != nil {
		return err
	}
	
	user.ID = int(dbUser.ID)
	user.CreatedAt = pgtypeToTime(dbUser.CreatedAt)
	return nil
}

func (r *userRepository) GetByID(id int) (*entity.User, error) {
	ctx := context.Background()
	
	dbUser, err := r.queries.GetUserByID(ctx, int32(id))
	if err != nil {
		return nil, err
	}
	
	return toDomainUser(dbUser), nil
}

func (r *userRepository) GetByEmail(email string) (*entity.User, error) {
	ctx := context.Background()
	
	dbUser, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	
	return toDomainUser(dbUser), nil
}

func (r *userRepository) Update(user *entity.User) error {
	ctx := context.Background()
	
	params := UpdateUserParams{
		ID:           int32(user.ID),
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
	}
	
	_, err := r.queries.UpdateUser(ctx, params)
	return err
}

func (r *userRepository) Delete(id int) error {
	ctx := context.Background()
	return r.queries.DeleteUser(ctx, int32(id))
}

