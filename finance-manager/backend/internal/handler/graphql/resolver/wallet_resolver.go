package resolver

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/finance-manager/backend/internal/domain/entity"
	"github.com/finance-manager/backend/internal/handler/graphql/generated"
)

// CreateWallet is the resolver for the createWallet field.
func (r *mutationResolver) CreateWallet(ctx context.Context, userID string, name string, typeArg string, currency string, balance *float64, category *string) (*generated.Wallet, error) {
	log.Printf("CreateWallet called with userID=%s, name=%s, type=%s, currency=%s, balance=%v, category=%v", userID, name, typeArg, currency, balance, category)

	// Convert string ID to int
	uid, err := strconv.Atoi(userID)
	if err != nil {
		log.Printf("Error converting userID to int: %v", err)
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Convert string type to WalletType
	walletType := entity.WalletType(typeArg)

	// Set default balance if not provided
	initialBalance := 0.0
	if balance != nil {
		initialBalance = *balance
	}

	// Create wallet using use case
	wallet, err := r.Resolver.WalletUseCase.CreateWallet(uid, name, walletType, currency, initialBalance, category)
	if err != nil {
		log.Printf("Error creating wallet: %v", err)
		return nil, err
	}

	log.Printf("Wallet created successfully with ID=%d", wallet.ID)

	// Convert domain entity to GraphQL model
	var categoryPtr *string
	if wallet.Category != nil {
		categoryPtr = wallet.Category
	}
	return &generated.Wallet{
		ID:        fmt.Sprintf("%d", wallet.ID),
		UserID:    fmt.Sprintf("%d", wallet.UserID),
		Name:      wallet.Name,
		Type:      string(wallet.Type),
		Currency:  wallet.Currency,
		Balance:   wallet.Balance,
		Category:  categoryPtr,
		CreatedAt: wallet.CreatedAt.Format(time.RFC3339),
	}, nil
}

// UpdateWallet is the resolver for the updateWallet field.
func (r *mutationResolver) UpdateWallet(ctx context.Context, id string, name *string, typeArg *string, currency *string, balance *float64, category *string) (*generated.Wallet, error) {
	log.Printf("UpdateWallet called with id=%s, category=%v", id, category)

	// Convert string ID to int
	wid, err := strconv.Atoi(id)
	if err != nil {
		log.Printf("Error converting wallet ID to int: %v", err)
		return nil, fmt.Errorf("invalid wallet ID: %w", err)
	}

	// Check if category was explicitly provided in the request
	// We need to check the field context to see if category was in the original arguments
	fc := graphql.GetFieldContext(ctx)
	categoryProvided := false
	if fc != nil && fc.Args != nil {
		_, categoryProvided = fc.Args["category"]
	}

	// Update wallet using use case
	// Pass category only if it was explicitly provided (even if nil, to allow clearing)
	var categoryToUpdate *string
	if categoryProvided {
		categoryToUpdate = category
	}

	wallet, err := r.Resolver.WalletUseCase.UpdateWallet(wid, name, typeArg, currency, balance, categoryToUpdate, categoryProvided)
	if err != nil {
		log.Printf("Error updating wallet: %v", err)
		return nil, fmt.Errorf("failed to update wallet: %w", err)
	}

	log.Printf("Wallet updated successfully with ID=%d", wallet.ID)

	// Convert domain entity to GraphQL model
	var categoryPtr *string
	if wallet.Category != nil {
		categoryPtr = wallet.Category
	}
	return &generated.Wallet{
		ID:        fmt.Sprintf("%d", wallet.ID),
		UserID:    fmt.Sprintf("%d", wallet.UserID),
		Name:      wallet.Name,
		Type:      string(wallet.Type),
		Currency:  wallet.Currency,
		Balance:   wallet.Balance,
		Category:  categoryPtr,
		CreatedAt: wallet.CreatedAt.Format(time.RFC3339),
	}, nil
}

// DeleteWallet is the resolver for the deleteWallet field.
func (r *mutationResolver) DeleteWallet(ctx context.Context, id string) (bool, error) {
	log.Printf("DeleteWallet called with id=%s", id)

	// Convert string ID to int
	wid, err := strconv.Atoi(id)
	if err != nil {
		log.Printf("Error converting wallet ID to int: %v", err)
		return false, fmt.Errorf("invalid wallet ID: %w", err)
	}

	// Delete wallet using use case
	err = r.Resolver.WalletUseCase.DeleteWallet(wid)
	if err != nil {
		log.Printf("Error deleting wallet: %v", err)
		return false, fmt.Errorf("failed to delete wallet: %w", err)
	}

	log.Printf("Wallet deleted successfully with ID=%d", wid)
	return true, nil
}

// Wallets is the resolver for the wallets field.
func (r *queryResolver) Wallets(ctx context.Context, userID string) ([]*generated.Wallet, error) {
	// Convert string ID to int
	uid, err := strconv.Atoi(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get wallets from use case
	wallets, err := r.Resolver.WalletUseCase.GetWalletsByUserID(uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallets: %w", err)
	}

	// Convert domain entities to GraphQL models
	result := make([]*generated.Wallet, len(wallets))
	for i, wallet := range wallets {
		var categoryPtr *string
		if wallet.Category != nil {
			categoryPtr = wallet.Category
		}
		result[i] = &generated.Wallet{
			ID:        fmt.Sprintf("%d", wallet.ID),
			UserID:    fmt.Sprintf("%d", wallet.UserID),
			Name:      wallet.Name,
			Type:      string(wallet.Type),
			Currency:  wallet.Currency,
			Balance:   wallet.Balance,
			Category:  categoryPtr,
			CreatedAt: wallet.CreatedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}

