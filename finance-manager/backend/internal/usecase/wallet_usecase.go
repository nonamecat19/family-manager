package usecase

import (
	"fmt"

	"github.com/finance-manager/backend/internal/domain/entity"
	"github.com/finance-manager/backend/internal/domain/repository"
)

type WalletUseCase struct {
	walletRepo repository.WalletRepository
	userRepo   repository.UserRepository
}

func NewWalletUseCase(walletRepo repository.WalletRepository, userRepo repository.UserRepository) *WalletUseCase {
	return &WalletUseCase{
		walletRepo: walletRepo,
		userRepo:   userRepo,
	}
}

func (uc *WalletUseCase) CreateWallet(userID int, name string, walletType entity.WalletType, currency string, balance float64, category *string) (*entity.Wallet, error) {
	// Verify user exists
	_, err := uc.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user with ID %d does not exist: %w", userID, err)
	}

	wallet := &entity.Wallet{
		UserID:   userID,
		Name:     name,
		Type:     walletType,
		Currency: currency,
		Balance:  balance,
		Category: category,
	}

	if err := uc.walletRepo.Create(wallet); err != nil {
		return nil, err
	}

	return wallet, nil
}

func (uc *WalletUseCase) GetWalletsByUserID(userID int) ([]*entity.Wallet, error) {
	return uc.walletRepo.GetByUserID(userID)
}

func (uc *WalletUseCase) GetWalletByID(id int) (*entity.Wallet, error) {
	return uc.walletRepo.GetByID(id)
}

func (uc *WalletUseCase) UpdateWallet(id int, name, walletType, currency *string, balance *float64, category *string, categoryProvided bool) (*entity.Wallet, error) {
	wallet, err := uc.walletRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Track if we need to update name, type, currency, or category
	needsUpdate := false

	if name != nil {
		wallet.Name = *name
		needsUpdate = true
	}
	if walletType != nil {
		wallet.Type = entity.WalletType(*walletType)
		needsUpdate = true
	}
	if currency != nil {
		wallet.Currency = *currency
		needsUpdate = true
	}
	if balance != nil {
		wallet.Balance = *balance
	}

	// Update name, type, and currency if any of them changed
	if needsUpdate {
		if err := uc.walletRepo.Update(wallet); err != nil {
			return nil, err
		}
	}

	// Update category separately if it was explicitly provided (allows setting to nil to clear)
	if categoryProvided {
		if err := uc.walletRepo.UpdateCategory(id, category); err != nil {
			return nil, err
		}
		wallet.Category = category
	}

	// Update balance separately if it was provided
	if balance != nil {
		if err := uc.walletRepo.UpdateBalance(id, *balance); err != nil {
			return nil, err
		}
	}

	return wallet, nil
}

func (uc *WalletUseCase) DeleteWallet(id int) error {
	return uc.walletRepo.Delete(id)
}
