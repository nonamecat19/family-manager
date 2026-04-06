package postgres

import (
	"context"

	"github.com/finance-manager/backend/internal/domain/entity"
	"github.com/finance-manager/backend/internal/domain/repository"
)

type investmentRepository struct {
	queries *Queries
}

func NewInvestmentRepository(queries *Queries) repository.InvestmentRepository {
	return &investmentRepository{queries: queries}
}

func (r *investmentRepository) Create(investment *entity.Investment) error {
	ctx := context.Background()
	
	var coinPtr *string
	if investment.Coin != "" {
		coinPtr = &investment.Coin
	}
	
	params := CreateInvestmentParams{
		UserID:      intToPgtypeInt4(investment.UserID),
		WalletID:    intToPgtypeInt4(investment.WalletID),
		Type:        string(investment.Type),
		Coin:        stringPtrToPgtypeText(coinPtr),
		Amount:      float64ToNumeric(investment.Amount),
		StartDate:   timeToPgtypeDate(investment.StartDate),
		EndDate:     timePtrToPgtypeDate(investment.EndDate),
		DailyReward: float64ToNumeric(investment.DailyReward),
		Status:      string(investment.Status),
	}
	
	dbInvestment, err := r.queries.CreateInvestment(ctx, params)
	if err != nil {
		return err
	}
	
	investment.ID = int(dbInvestment.ID)
	return nil
}

func (r *investmentRepository) GetByID(id int) (*entity.Investment, error) {
	ctx := context.Background()
	
	dbInvestment, err := r.queries.GetInvestmentByID(ctx, int32(id))
	if err != nil {
		return nil, err
	}
	
	return toDomainInvestment(dbInvestment), nil
}

func (r *investmentRepository) GetByUserID(userID int) ([]*entity.Investment, error) {
	ctx := context.Background()
	
	dbInvestments, err := r.queries.GetInvestmentsByUserID(ctx, intToPgtypeInt4(userID))
	if err != nil {
		return nil, err
	}
	
	investments := make([]*entity.Investment, len(dbInvestments))
	for i, inv := range dbInvestments {
		investments[i] = toDomainInvestment(inv)
	}
	
	return investments, nil
}

func (r *investmentRepository) Update(investment *entity.Investment) error {
	ctx := context.Background()
	
	var coinPtr *string
	if investment.Coin != "" {
		coinPtr = &investment.Coin
	}
	
	params := UpdateInvestmentParams{
		ID:          int32(investment.ID),
		WalletID:    intToPgtypeInt4(investment.WalletID),
		Type:        string(investment.Type),
		Coin:        stringPtrToPgtypeText(coinPtr),
		Amount:      float64ToNumeric(investment.Amount),
		StartDate:   timeToPgtypeDate(investment.StartDate),
		EndDate:     timePtrToPgtypeDate(investment.EndDate),
		DailyReward: float64ToNumeric(investment.DailyReward),
		Status:      string(investment.Status),
	}
	
	_, err := r.queries.UpdateInvestment(ctx, params)
	return err
}

func (r *investmentRepository) Delete(id int) error {
	ctx := context.Background()
	return r.queries.DeleteInvestment(ctx, int32(id))
}

