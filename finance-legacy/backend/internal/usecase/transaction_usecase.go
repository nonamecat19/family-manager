package usecase

import (
	"time"

	"github.com/finance-manager/backend/internal/domain/entity"
	"github.com/finance-manager/backend/internal/domain/repository"
)

type TransactionUseCase struct {
	transactionRepo repository.TransactionRepository
	walletRepo      repository.WalletRepository
}

func NewTransactionUseCase(
	transactionRepo repository.TransactionRepository,
	walletRepo repository.WalletRepository,
) *TransactionUseCase {
	return &TransactionUseCase{
		transactionRepo: transactionRepo,
		walletRepo:      walletRepo,
	}
}

func (uc *TransactionUseCase) CreateTransaction(
	walletID int,
	transactionType entity.TransactionType,
	amount float64,
	currency string,
	description, category string,
	executedAt *time.Time,
) (*entity.Transaction, error) {
	// Get wallet to verify it exists
	wallet, err := uc.walletRepo.GetByID(walletID)
	if err != nil {
		return nil, err
	}

	transaction := &entity.Transaction{
		WalletID:    walletID,
		Type:        transactionType,
		Amount:      amount,
		Currency:    currency,
		Description: description,
		Category:    category,
		ExecutedAt:  executedAt,
	}

	if err := uc.transactionRepo.Create(transaction); err != nil {
		return nil, err
	}

	// Update wallet balance
	var newBalance float64
	switch transactionType {
	case entity.TransactionTypeIncome:
		newBalance = wallet.Balance + amount
	case entity.TransactionTypeExpense:
		newBalance = wallet.Balance - amount
	case entity.TransactionTypeTransfer:
		// Transfer doesn't change balance (handled separately)
		newBalance = wallet.Balance
	}

	if transactionType != entity.TransactionTypeTransfer {
		if err := uc.walletRepo.UpdateBalance(walletID, newBalance); err != nil {
			return nil, err
		}
	}

	return transaction, nil
}

func (uc *TransactionUseCase) GetTransactionsByWalletID(walletID int, from, to *time.Time) ([]*entity.Transaction, error) {
	return uc.transactionRepo.GetByWalletID(walletID, from, to)
}

func (uc *TransactionUseCase) GetTransactionByID(id int) (*entity.Transaction, error) {
	return uc.transactionRepo.GetByID(id)
}

func (uc *TransactionUseCase) DeleteTransaction(id int) error {
	return uc.transactionRepo.Delete(id)
}
