package resolver

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/finance-manager/backend/internal/domain/entity"
	"github.com/finance-manager/backend/internal/handler/graphql/generated"
)

// CreateTransaction is the resolver for the createTransaction field.
func (r *mutationResolver) CreateTransaction(ctx context.Context, walletID string, typeArg string, amount float64, currency string, description *string, category *string, executedAt *string) (*generated.Transaction, error) {
	log.Printf("CreateTransaction called with walletID=%s, type=%s, amount=%f", walletID, typeArg, amount)

	// Convert string ID to int
	wid, err := strconv.Atoi(walletID)
	if err != nil {
		log.Printf("Error converting wallet ID to int: %v", err)
		return nil, fmt.Errorf("invalid wallet ID: %w", err)
	}

	// Convert string type to TransactionType
	transactionType := entity.TransactionType(typeArg)

	// Parse optional executedAt date
	var executedAtTime *time.Time
	if executedAt != nil {
		parsed, err := time.Parse(time.RFC3339, *executedAt)
		if err != nil {
			return nil, fmt.Errorf("invalid executedAt date format: %w", err)
		}
		executedAtTime = &parsed
	}

	// Extract description and category strings
	var descStr, catStr string
	if description != nil {
		descStr = *description
	}
	if category != nil {
		catStr = *category
	}

	// Create transaction using use case
	transaction, err := r.Resolver.TransactionUseCase.CreateTransaction(
		wid,
		transactionType,
		amount,
		currency,
		descStr,
		catStr,
		executedAtTime,
	)
	if err != nil {
		log.Printf("Error creating transaction: %v", err)
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	log.Printf("Transaction created successfully with ID=%d", transaction.ID)

	// Convert domain entity to GraphQL model
	var descPtr, catPtr *string
	if transaction.Description != "" {
		descPtr = &transaction.Description
	}
	if transaction.Category != "" {
		catPtr = &transaction.Category
	}

	var executedAtStr *string
	if transaction.ExecutedAt != nil {
		formatted := transaction.ExecutedAt.Format(time.RFC3339)
		executedAtStr = &formatted
	}

	return &generated.Transaction{
		ID:          fmt.Sprintf("%d", transaction.ID),
		WalletID:    fmt.Sprintf("%d", transaction.WalletID),
		Type:        string(transaction.Type),
		Amount:      transaction.Amount,
		Currency:    transaction.Currency,
		Description: descPtr,
		Category:    catPtr,
		CreatedAt:   transaction.CreatedAt.Format(time.RFC3339),
		ExecutedAt:  executedAtStr,
	}, nil
}

// Transactions is the resolver for the transactions field.
func (r *queryResolver) Transactions(ctx context.Context, walletID string, from *string, to *string) ([]*generated.Transaction, error) {
	// Convert string ID to int
	wid, err := strconv.Atoi(walletID)
	if err != nil {
		return nil, fmt.Errorf("invalid wallet ID: %w", err)
	}

	// Parse optional date strings to time.Time
	var fromTime, toTime *time.Time
	if from != nil {
		parsed, err := time.Parse(time.RFC3339, *from)
		if err != nil {
			return nil, fmt.Errorf("invalid from date format: %w", err)
		}
		fromTime = &parsed
	}
	if to != nil {
		parsed, err := time.Parse(time.RFC3339, *to)
		if err != nil {
			return nil, fmt.Errorf("invalid to date format: %w", err)
		}
		toTime = &parsed
	}

	// Get transactions from use case
	transactions, err := r.Resolver.TransactionUseCase.GetTransactionsByWalletID(wid, fromTime, toTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	// Convert domain entities to GraphQL models
	result := make([]*generated.Transaction, len(transactions))
	for i, transaction := range transactions {
		var description *string
		if transaction.Description != "" {
			description = &transaction.Description
		}

		var category *string
		if transaction.Category != "" {
			category = &transaction.Category
		}

		var executedAt *string
		if transaction.ExecutedAt != nil {
			formatted := transaction.ExecutedAt.Format(time.RFC3339)
			executedAt = &formatted
		}

		result[i] = &generated.Transaction{
			ID:          fmt.Sprintf("%d", transaction.ID),
			WalletID:    fmt.Sprintf("%d", transaction.WalletID),
			Type:        string(transaction.Type),
			Amount:      transaction.Amount,
			Currency:    transaction.Currency,
			Description: description,
			Category:    category,
			CreatedAt:   transaction.CreatedAt.Format(time.RFC3339),
			ExecutedAt:  executedAt,
		}
	}

	return result, nil
}



