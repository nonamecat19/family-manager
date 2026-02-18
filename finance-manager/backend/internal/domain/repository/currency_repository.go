package repository

import "github.com/finance-manager/backend/internal/domain/entity"

type CurrencyRepository interface {
	GetByCode(code string) (*entity.Currency, error)
	GetByID(id int) (*entity.Currency, error)
	GetAll() ([]*entity.Currency, error)
	GetByType(currencyType entity.CurrencyType) ([]*entity.Currency, error)
	UpdateExchangeRate(code string, rate float64) error
}



