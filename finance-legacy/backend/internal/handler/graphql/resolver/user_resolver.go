package resolver

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/finance-manager/backend/internal/handler/graphql/generated"
)

// CreateUser is the resolver for the createUser field.
func (r *mutationResolver) CreateUser(ctx context.Context, email string, password string) (*generated.User, error) {
	log.Printf("CreateUser called with email=%s", email)

	// Create user using use case
	user, err := r.Resolver.UserUseCase.CreateUser(email, password)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		return nil, err
	}

	log.Printf("User created successfully with ID=%d", user.ID)

	// Convert domain entity to GraphQL model
	return &generated.User{
		ID:        fmt.Sprintf("%d", user.ID),
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}, nil
}


