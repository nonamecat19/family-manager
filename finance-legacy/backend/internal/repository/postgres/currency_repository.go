package postgres

import (
	"context"

	"github.com/finance-manager/backend/internal/domain/entity"
	"github.com/finance-manager/backend/internal/domain/repository"
)

type currencyRepository struct {
	queries *Queries
}

func NewCurrencyRepository(queries *Queries) repository.CurrencyRepository {
	return &currencyRepository{queries: queries}
}

func (r *currencyRepository) GetByCode(code string) (*entity.Currency, error) {
	ctx := context.Background()

	dbCurrency, err := r.queries.GetCurrencyByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	return toDomainCurrency(dbCurrency), nil
}

func (r *currencyRepository) GetByID(id int) (*entity.Currency, error) {
	ctx := context.Background()

	dbCurrency, err := r.queries.GetCurrencyByID(ctx, int32(id))
	if err != nil {
		return nil, err
	}

	return toDomainCurrency(dbCurrency), nil
}

func (r *currencyRepository) GetAll() ([]*entity.Currency, error) {
	ctx := context.Background()

	dbCurrencies, err := r.queries.GetAllCurrencies(ctx)
	if err != nil {
		return nil, err
	}

	currencies := make([]*entity.Currency, len(dbCurrencies))
	for i, c := range dbCurrencies {
		currencies[i] = toDomainCurrency(c)
	}

	return currencies, nil
}

func (r *currencyRepository) GetByType(currencyType entity.CurrencyType) ([]*entity.Currency, error) {
	ctx := context.Background()

	dbCurrencies, err := r.queries.GetCurrenciesByType(ctx, string(currencyType))
	if err != nil {
		return nil, err
	}

	currencies := make([]*entity.Currency, len(dbCurrencies))
	for i, c := range dbCurrencies {
		currencies[i] = toDomainCurrency(c)
	}

	return currencies, nil
}

func (r *currencyRepository) UpdateExchangeRate(code string, rate float64) error {
	ctx := context.Background()

	params := UpdateCurrencyExchangeRateParams{
		ExchangeRate: float64ToNumeric(rate),
		Code:         code,
	}

	return r.queries.UpdateCurrencyExchangeRate(ctx, params)
}



