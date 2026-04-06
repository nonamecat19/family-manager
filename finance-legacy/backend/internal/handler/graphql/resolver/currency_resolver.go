package resolver

import (
	"context"
	"fmt"

	"github.com/finance-manager/backend/internal/domain/entity"
	"github.com/finance-manager/backend/internal/handler/graphql/generated"
)

// Currencies is the resolver for the currencies field.
func (r *queryResolver) Currencies(ctx context.Context, typeArg *string) ([]*generated.Currency, error) {
	var currencies []*entity.Currency
	var err error

	if typeArg != nil {
		currencyType := entity.CurrencyType(*typeArg)
		currencies, err = r.Resolver.CurrencyUseCase.GetCurrenciesByType(currencyType)
	} else {
		currencies, err = r.Resolver.CurrencyUseCase.GetAllCurrencies()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get currencies: %w", err)
	}

	// Convert domain entities to GraphQL models
	result := make([]*generated.Currency, len(currencies))
	for i, currency := range currencies {
		result[i] = &generated.Currency{
			ID:           fmt.Sprintf("%d", currency.ID),
			Code:         currency.Code,
			Type:         string(currency.Type),
			ExchangeRate: currency.ExchangeRate,
			Name:         currency.Name,
		}
	}

	return result, nil
}

// Currency is the resolver for the currency field.
func (r *queryResolver) Currency(ctx context.Context, code string) (*generated.Currency, error) {
	currency, err := r.Resolver.CurrencyUseCase.GetCurrencyByCode(code)
	if err != nil {
		return nil, fmt.Errorf("failed to get currency: %w", err)
	}

	return &generated.Currency{
		ID:           fmt.Sprintf("%d", currency.ID),
		Code:         currency.Code,
		Type:         string(currency.Type),
		ExchangeRate: currency.ExchangeRate,
		Name:         currency.Name,
	}, nil
}

// CurrencyInfo is the resolver for the currencyInfo field on Wallet.
func (r *walletResolver) CurrencyInfo(ctx context.Context, obj *generated.Wallet) (*generated.Currency, error) {
	if obj.Currency == "" {
		return nil, nil
	}

	currency, err := r.Resolver.CurrencyUseCase.GetCurrencyByCode(obj.Currency)
	if err != nil {
		// Return nil if currency not found instead of error
		return nil, nil
	}

	return &generated.Currency{
		ID:           fmt.Sprintf("%d", currency.ID),
		Code:         currency.Code,
		Type:         string(currency.Type),
		ExchangeRate: currency.ExchangeRate,
		Name:         currency.Name,
	}, nil
}

// CurrencyInfo is the resolver for the currencyInfo field on Transaction.
func (r *transactionResolver) CurrencyInfo(ctx context.Context, obj *generated.Transaction) (*generated.Currency, error) {
	if obj.Currency == "" {
		return nil, nil
	}

	currency, err := r.Resolver.CurrencyUseCase.GetCurrencyByCode(obj.Currency)
	if err != nil {
		// Return nil if currency not found instead of error
		return nil, nil
	}

	return &generated.Currency{
		ID:           fmt.Sprintf("%d", currency.ID),
		Code:         currency.Code,
		Type:         string(currency.Type),
		ExchangeRate: currency.ExchangeRate,
		Name:         currency.Name,
	}, nil
}

// CurrencyInfo is the resolver for the currencyInfo field on Recurring.
func (r *recurringResolver) CurrencyInfo(ctx context.Context, obj *generated.Recurring) (*generated.Currency, error) {
	if obj.Currency == "" {
		return nil, nil
	}

	currency, err := r.Resolver.CurrencyUseCase.GetCurrencyByCode(obj.Currency)
	if err != nil {
		// Return nil if currency not found instead of error
		return nil, nil
	}

	return &generated.Currency{
		ID:           fmt.Sprintf("%d", currency.ID),
		Code:         currency.Code,
		Type:         string(currency.Type),
		ExchangeRate: currency.ExchangeRate,
		Name:         currency.Name,
	}, nil
}

