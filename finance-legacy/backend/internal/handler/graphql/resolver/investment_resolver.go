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

// CreateInvestment is the resolver for the createInvestment field.
func (r *mutationResolver) CreateInvestment(ctx context.Context, userID string, walletID string, typeArg string, coin *string, amount float64, startDate string, endDate *string, dailyReward *float64) (*generated.Investment, error) {
	log.Printf("CreateInvestment called with userID=%s, walletID=%s, type=%s", userID, walletID, typeArg)

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

	// Convert string type to InvestmentType
	investmentType := entity.InvestmentType(typeArg)

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

	// Extract coin string
	var coinStr string
	if coin != nil {
		coinStr = *coin
	}

	// Extract dailyReward
	var dailyRewardVal float64
	if dailyReward != nil {
		dailyRewardVal = *dailyReward
	}

	// Create investment using use case
	investment, err := r.Resolver.InvestmentUseCase.CreateInvestment(
		uid,
		wid,
		investmentType,
		coinStr,
		amount,
		startDateParsed,
		endDateParsed,
		dailyRewardVal,
	)
	if err != nil {
		log.Printf("Error creating investment: %v", err)
		return nil, fmt.Errorf("failed to create investment: %w", err)
	}

	log.Printf("Investment created successfully with ID=%d", investment.ID)

	// Convert domain entity to GraphQL model
	var coinPtr *string
	if investment.Coin != "" {
		coinPtr = &investment.Coin
	}

	var amountPtr *float64
	amountPtr = &investment.Amount

	var endDateStr *string
	if investment.EndDate != nil {
		formatted := investment.EndDate.Format(time.RFC3339)
		endDateStr = &formatted
	}

	var dailyRewardPtr *float64
	if investment.DailyReward != 0 {
		dailyRewardPtr = &investment.DailyReward
	}

	return &generated.Investment{
		ID:          fmt.Sprintf("%d", investment.ID),
		UserID:      fmt.Sprintf("%d", investment.UserID),
		WalletID:    fmt.Sprintf("%d", investment.WalletID),
		Type:        string(investment.Type),
		Coin:        coinPtr,
		Amount:      amountPtr,
		StartDate:   investment.StartDate.Format(time.RFC3339),
		EndDate:     endDateStr,
		DailyReward: dailyRewardPtr,
		Status:      string(investment.Status),
	}, nil
}

// Investments is the resolver for the investments field.
func (r *queryResolver) Investments(ctx context.Context, userID string) ([]*generated.Investment, error) {
	// Convert string ID to int
	uid, err := strconv.Atoi(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get investments from use case
	investments, err := r.Resolver.InvestmentUseCase.GetInvestmentsByUserID(uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get investments: %w", err)
	}

	// Convert domain entities to GraphQL models
	result := make([]*generated.Investment, len(investments))
	for i, investment := range investments {
		var coin *string
		if investment.Coin != "" {
			coin = &investment.Coin
		}

		var amount *float64
		amount = &investment.Amount

		var endDate *string
		if investment.EndDate != nil {
			formatted := investment.EndDate.Format(time.RFC3339)
			endDate = &formatted
		}

		var dailyReward *float64
		if investment.DailyReward != 0 {
			dailyReward = &investment.DailyReward
		}

		result[i] = &generated.Investment{
			ID:          fmt.Sprintf("%d", investment.ID),
			UserID:      fmt.Sprintf("%d", investment.UserID),
			WalletID:    fmt.Sprintf("%d", investment.WalletID),
			Type:        string(investment.Type),
			Coin:        coin,
			Amount:      amount,
			StartDate:   investment.StartDate.Format(time.RFC3339),
			EndDate:     endDate,
			DailyReward: dailyReward,
			Status:      string(investment.Status),
		}
	}

	return result, nil
}



