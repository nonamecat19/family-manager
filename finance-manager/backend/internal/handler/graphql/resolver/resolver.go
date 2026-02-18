package resolver

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

import (
	"github.com/finance-manager/backend/internal/usecase"
)

type Resolver struct {
	UserUseCase        *usecase.UserUseCase
	WalletUseCase      *usecase.WalletUseCase
	TransactionUseCase *usecase.TransactionUseCase
	RecurringUseCase   *usecase.RecurringUseCase
	InvestmentUseCase  *usecase.InvestmentUseCase
	CurrencyUseCase    *usecase.CurrencyUseCase
}
