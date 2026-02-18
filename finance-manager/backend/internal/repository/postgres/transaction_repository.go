package postgres

import (
	"context"
	"time"

	"github.com/finance-manager/backend/internal/domain/entity"
	"github.com/finance-manager/backend/internal/domain/repository"
)

type transactionRepository struct {
	queries *Queries
}

func NewTransactionRepository(queries *Queries) repository.TransactionRepository {
	return &transactionRepository{queries: queries}
}

func (r *transactionRepository) Create(transaction *entity.Transaction) error {
	ctx := context.Background()

	var descPtr, catPtr *string
	if transaction.Description != "" {
		descPtr = &transaction.Description
	}
	if transaction.Category != "" {
		catPtr = &transaction.Category
	}

	params := CreateTransactionParams{
		WalletID:    intToPgtypeInt4(transaction.WalletID),
		Type:        string(transaction.Type),
		Amount:      float64ToNumeric(transaction.Amount),
		Currency:    transaction.Currency,
		Description: stringPtrToPgtypeText(descPtr),
		Category:    stringPtrToPgtypeText(catPtr),
		ExecutedAt:  timePtrToPgtype(transaction.ExecutedAt),
	}

	dbTransaction, err := r.queries.CreateTransaction(ctx, params)
	if err != nil {
		return err
	}

	transaction.ID = int(dbTransaction.ID)
	transaction.CreatedAt = pgtypeToTime(dbTransaction.CreatedAt)
	return nil
}

func (r *transactionRepository) GetByID(id int) (*entity.Transaction, error) {
	ctx := context.Background()

	dbTransaction, err := r.queries.GetTransactionByID(ctx, int32(id))
	if err != nil {
		return nil, err
	}

	return toDomainTransaction(dbTransaction), nil
}

func (r *transactionRepository) GetByWalletID(walletID int, from, to *time.Time) ([]*entity.Transaction, error) {
	ctx := context.Background()

	params := GetTransactionsByWalletIDParams{
		WalletID: intToPgtypeInt4(walletID),
		Column2:  timePtrToPgtype(from),
		Column3:  timePtrToPgtype(to),
	}

	dbTransactions, err := r.queries.GetTransactionsByWalletID(ctx, params)
	if err != nil {
		return nil, err
	}

	transactions := make([]*entity.Transaction, len(dbTransactions))
	for i, t := range dbTransactions {
		transactions[i] = toDomainTransaction(t)
	}

	return transactions, nil
}

func (r *transactionRepository) Update(transaction *entity.Transaction) error {
	ctx := context.Background()

	var descPtr, catPtr *string
	if transaction.Description != "" {
		descPtr = &transaction.Description
	}
	if transaction.Category != "" {
		catPtr = &transaction.Category
	}

	params := UpdateTransactionParams{
		ID:          int32(transaction.ID),
		Type:        string(transaction.Type),
		Amount:      float64ToNumeric(transaction.Amount),
		Currency:    transaction.Currency,
		Description: stringPtrToPgtypeText(descPtr),
		Category:    stringPtrToPgtypeText(catPtr),
		ExecutedAt:  timePtrToPgtype(transaction.ExecutedAt),
	}

	_, err := r.queries.UpdateTransaction(ctx, params)
	return err
}

func (r *transactionRepository) Delete(id int) error {
	ctx := context.Background()
	return r.queries.DeleteTransaction(ctx, int32(id))
}
