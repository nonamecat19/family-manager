package usecase

import (
	"time"

	"github.com/finance-manager/backend/internal/domain/entity"
	"github.com/finance-manager/backend/internal/domain/repository"
)

type InvestmentUseCase struct {
	investmentRepo repository.InvestmentRepository
	walletRepo     repository.WalletRepository
}

func NewInvestmentUseCase(
	investmentRepo repository.InvestmentRepository,
	walletRepo repository.WalletRepository,
) *InvestmentUseCase {
	return &InvestmentUseCase{
		investmentRepo: investmentRepo,
		walletRepo:     walletRepo,
	}
}

func (uc *InvestmentUseCase) CreateInvestment(
	userID, walletID int,
	investmentType entity.InvestmentType,
	coin string,
	amount float64,
	startDate time.Time,
	endDate *time.Time,
	dailyReward float64,
) (*entity.Investment, error) {
	// Verify wallet exists
	_, err := uc.walletRepo.GetByID(walletID)
	if err != nil {
		return nil, err
	}

	investment := &entity.Investment{
		UserID:      userID,
		WalletID:    walletID,
		Type:        investmentType,
		Coin:        coin,
		Amount:      amount,
		StartDate:   startDate,
		EndDate:     endDate,
		DailyReward: dailyReward,
		Status:      entity.InvestmentStatusActive,
	}

	if err := uc.investmentRepo.Create(investment); err != nil {
		return nil, err
	}

	return investment, nil
}

func (uc *InvestmentUseCase) GetInvestmentsByUserID(userID int) ([]*entity.Investment, error) {
	return uc.investmentRepo.GetByUserID(userID)
}

func (uc *InvestmentUseCase) GetInvestmentByID(id int) (*entity.Investment, error) {
	return uc.investmentRepo.GetByID(id)
}

func (uc *InvestmentUseCase) UpdateInvestment(investment *entity.Investment) error {
	return uc.investmentRepo.Update(investment)
}

func (uc *InvestmentUseCase) DeleteInvestment(id int) error {
	return uc.investmentRepo.Delete(id)
}
