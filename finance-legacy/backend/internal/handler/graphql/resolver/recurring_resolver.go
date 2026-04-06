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

// CreateRecurring is the resolver for the createRecurring field.
func (r *mutationResolver) CreateRecurring(ctx context.Context, userID string, walletID string, typeArg string, amount float64, currency string, description *string, recurrence string, startDate string, endDate *string) (*generated.Recurring, error) {
	log.Printf("CreateRecurring called with userID=%s, walletID=%s, type=%s", userID, walletID, typeArg)

	// Convert string IDs to int
	uid, err := strconv.Atoi(userID)
	if err != nil {
		log.Printf("Error converting user ID to int: %v", err)
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	wid, err := strconv.Atoi(walletID)
	if err != nil {
		log.Printf("Error converting wallet ID to int: %v", err)
		return nil, fmt.Errorf("invalid wallet ID: %w", err)
	}

	// Convert string type to RecurringType
	recurringType := entity.RecurringType(typeArg)

	// Convert string recurrence to RecurrencePattern
	recurrencePattern := entity.RecurrencePattern(recurrence)

	// Parse startDate
	startDateParsed, err := time.Parse(time.RFC3339, startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid startDate format: %w", err)
	}

	// Parse optional endDate
	var endDateParsed *time.Time
	if endDate != nil {
		parsed, err := time.Parse(time.RFC3339, *endDate)
		if err != nil {
			return nil, fmt.Errorf("invalid endDate format: %w", err)
		}
		endDateParsed = &parsed
	}

	// Extract description string
	var descStr string
	if description != nil {
		descStr = *description
	}

	// Create recurring using use case
	recurring, err := r.Resolver.RecurringUseCase.CreateRecurring(
		uid,
		wid,
		recurringType,
		amount,
		currency,
		descStr,
		recurrencePattern,
		startDateParsed,
		endDateParsed,
	)
	if err != nil {
		log.Printf("Error creating recurring: %v", err)
		return nil, fmt.Errorf("failed to create recurring: %w", err)
	}

	log.Printf("Recurring created successfully with ID=%d", recurring.ID)

	// Convert domain entity to GraphQL model
	var descPtr *string
	if recurring.Description != "" {
		descPtr = &recurring.Description
	}

	var endDateStr *string
	if recurring.EndDate != nil {
		formatted := recurring.EndDate.Format(time.RFC3339)
		endDateStr = &formatted
	}

	var nextExecution *string
	if recurring.NextExecution != nil {
		formatted := recurring.NextExecution.Format(time.RFC3339)
		nextExecution = &formatted
	}

	return &generated.Recurring{
		ID:            fmt.Sprintf("%d", recurring.ID),
		UserID:        fmt.Sprintf("%d", recurring.UserID),
		WalletID:      fmt.Sprintf("%d", recurring.WalletID),
		Type:          string(recurring.Type),
		Amount:        recurring.Amount,
		Currency:      recurring.Currency,
		Description:   descPtr,
		Recurrence:    string(recurring.Recurrence),
		StartDate:     recurring.StartDate.Format(time.RFC3339),
		EndDate:       endDateStr,
		NextExecution: nextExecution,
	}, nil
}

// RecurringPayments is the resolver for the recurringPayments field.
func (r *queryResolver) RecurringPayments(ctx context.Context, userID string) ([]*generated.Recurring, error) {
	// Convert string ID to int
	uid, err := strconv.Atoi(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get recurring payments from use case
	recurringPayments, err := r.Resolver.RecurringUseCase.GetRecurringByUserID(uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get recurring payments: %w", err)
	}

	// Convert domain entities to GraphQL models
	result := make([]*generated.Recurring, len(recurringPayments))
	for i, recurring := range recurringPayments {
		var description *string
		if recurring.Description != "" {
			description = &recurring.Description
		}

		var endDate *string
		if recurring.EndDate != nil {
			formatted := recurring.EndDate.Format(time.RFC3339)
			endDate = &formatted
		}

		var nextExecution *string
		if recurring.NextExecution != nil {
			formatted := recurring.NextExecution.Format(time.RFC3339)
			nextExecution = &formatted
		}

		result[i] = &generated.Recurring{
			ID:            fmt.Sprintf("%d", recurring.ID),
			UserID:        fmt.Sprintf("%d", recurring.UserID),
			WalletID:      fmt.Sprintf("%d", recurring.WalletID),
			Type:          string(recurring.Type),
			Amount:        recurring.Amount,
			Currency:      recurring.Currency,
			Description:   description,
			Recurrence:    string(recurring.Recurrence),
			StartDate:     recurring.StartDate.Format(time.RFC3339),
			EndDate:       endDate,
			NextExecution: nextExecution,
		}
	}

	return result, nil
}



