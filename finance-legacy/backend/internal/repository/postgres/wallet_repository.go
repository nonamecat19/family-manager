package postgres

import (
	"context"

	"github.com/finance-manager/backend/internal/domain/entity"
	"github.com/finance-manager/backend/internal/domain/repository"
)

type walletRepository struct {
	queries *Queries
}

func NewWalletRepository(queries *Queries) repository.WalletRepository {
	return &walletRepository{queries: queries}
}

func (r *walletRepository) Create(wallet *entity.Wallet) error {
	ctx := context.Background()
	
	params := CreateWalletParams{
		UserID:   intToPgtypeInt4(wallet.UserID),
		Name:     wallet.Name,
		Type:     string(wallet.Type),
		Currency: wallet.Currency,
		Balance:  float64ToNumeric(wallet.Balance),
		Category: stringPtrToPgtypeText(wallet.Category),
	}
	
	dbWallet, err := r.queries.CreateWallet(ctx, params)
	if err != nil {
		return err
	}
	
	wallet.ID = int(dbWallet.ID)
	wallet.CreatedAt = pgtypeToTime(dbWallet.CreatedAt)
	wallet.Category = pgtypeTextToStringPtr(dbWallet.Category)
	return nil
}

func (r *walletRepository) GetByID(id int) (*entity.Wallet, error) {
	ctx := context.Background()
	
	dbWallet, err := r.queries.GetWalletByID(ctx, int32(id))
	if err != nil {
		return nil, err
	}
	
	return getWalletByIDRowToDomain(dbWallet), nil
}

func (r *walletRepository) GetByUserID(userID int) ([]*entity.Wallet, error) {
	ctx := context.Background()
	
	dbWallets, err := r.queries.GetWalletsByUserID(ctx, intToPgtypeInt4(userID))
	if err != nil {
		return nil, err
	}
	
	wallets := make([]*entity.Wallet, len(dbWallets))
	for i, w := range dbWallets {
		wallets[i] = getWalletsByUserIDRowToDomain(w)
	}
	
	return wallets, nil
}

func (r *walletRepository) Update(wallet *entity.Wallet) error {
	ctx := context.Background()
	
	params := UpdateWalletParams{
		ID:       int32(wallet.ID),
		Name:     wallet.Name,
		Type:     string(wallet.Type),
		Currency: wallet.Currency,
	}
	
	_, err := r.queries.UpdateWallet(ctx, params)
	return err
}

func (r *walletRepository) UpdateCategory(id int, category *string) error {
	ctx := context.Background()
	
	params := UpdateWalletCategoryParams{
		ID:       int32(id),
		Category: stringPtrToPgtypeText(category),
	}
	
	_, err := r.queries.UpdateWalletCategory(ctx, params)
	return err
}

func (r *walletRepository) Delete(id int) error {
	ctx := context.Background()
	return r.queries.DeleteWallet(ctx, int32(id))
}

func (r *walletRepository) UpdateBalance(id int, balance float64) error {
	ctx := context.Background()
	
	params := UpdateWalletBalanceParams{
		ID:      int32(id),
		Balance: float64ToNumeric(balance),
	}
	
	_, err := r.queries.UpdateWalletBalance(ctx, params)
	return err
}

