package usecase

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/finance-manager/backend/internal/domain/entity"
	"github.com/finance-manager/backend/internal/domain/repository"
)

type CurrencyUseCase struct {
	currencyRepo repository.CurrencyRepository
}

func NewCurrencyUseCase(currencyRepo repository.CurrencyRepository) *CurrencyUseCase {
	return &CurrencyUseCase{
		currencyRepo: currencyRepo,
	}
}

func (uc *CurrencyUseCase) GetCurrencyByCode(code string) (*entity.Currency, error) {
	return uc.currencyRepo.GetByCode(code)
}

func (uc *CurrencyUseCase) GetCurrencyByID(id int) (*entity.Currency, error) {
	return uc.currencyRepo.GetByID(id)
}

func (uc *CurrencyUseCase) GetAllCurrencies() ([]*entity.Currency, error) {
	return uc.currencyRepo.GetAll()
}

func (uc *CurrencyUseCase) GetCurrenciesByType(currencyType entity.CurrencyType) ([]*entity.Currency, error) {
	return uc.currencyRepo.GetByType(currencyType)
}

func (uc *CurrencyUseCase) UpdateExchangeRate(code string, rate float64) error {
	return uc.currencyRepo.UpdateExchangeRate(code, rate)
}

// UpdateExchangeRatesFromAPIs fetches exchange rates from external APIs and updates the database
func (uc *CurrencyUseCase) UpdateExchangeRatesFromAPIs() error {
	// Fetch crypto rates
	if err := uc.updateCryptoRates(); err != nil {
		return fmt.Errorf("failed to update crypto rates: %w", err)
	}

	// Fetch fiat rates
	if err := uc.updateFiatRates(); err != nil {
		return fmt.Errorf("failed to update fiat rates: %w", err)
	}

	return nil
}

// updateCryptoRates fetches crypto rates from CoinGecko
func (uc *CurrencyUseCase) updateCryptoRates() error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get("https://api.coingecko.com/api/v3/simple/price?ids=bitcoin,ethereum&vs_currencies=usd")
	if err != nil {
		return fmt.Errorf("failed to fetch crypto rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("crypto API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read crypto API response: %w", err)
	}

	var cryptoRates map[string]map[string]float64
	if err := json.Unmarshal(body, &cryptoRates); err != nil {
		return fmt.Errorf("failed to parse crypto API response: %w", err)
	}

	// Update BTC
	if btcData, ok := cryptoRates["bitcoin"]; ok {
		if usdRate, ok := btcData["usd"]; ok {
			// CoinGecko returns: 1 BTC = usdRate USD
			// Standard format: 1 currency = exchangeRate USD
			// So we store: exchangeRate = usdRate (USD per 1 BTC)
			// Example: if 1 BTC = 91000 USD, store 91000
			if err := uc.currencyRepo.UpdateExchangeRate("BTC", usdRate); err != nil {
				return fmt.Errorf("failed to update BTC rate: %w", err)
			}
		}
	}

	// Update ETH
	if ethData, ok := cryptoRates["ethereum"]; ok {
		if usdRate, ok := ethData["usd"]; ok {
			// CoinGecko returns: 1 ETH = usdRate USD
			// Standard format: 1 currency = exchangeRate USD
			// So we store: exchangeRate = usdRate (USD per 1 ETH)
			// Example: if 1 ETH = 3157 USD, store 3157
			if err := uc.currencyRepo.UpdateExchangeRate("ETH", usdRate); err != nil {
				return fmt.Errorf("failed to update ETH rate: %w", err)
			}
		}
	}

	return nil
}

// updateFiatRates fetches fiat rates from Monobank
func (uc *CurrencyUseCase) updateFiatRates() error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get("https://api.monobank.ua/bank/currency")
	if err != nil {
		return fmt.Errorf("failed to fetch fiat rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fiat API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read fiat API response: %w", err)
	}

	var fiatRates []struct {
		CurrencyCodeA int      `json:"currencyCodeA"` // From currency (840 = USD)
		CurrencyCodeB int      `json:"currencyCodeB"` // To currency (980 = UAH)
		RateCross     *float64 `json:"rateCross"`     // Mid rate (may be missing)
		RateBuy       float64  `json:"rateBuy"`
		RateSell      float64  `json:"rateSell"`
	}

	if err := json.Unmarshal(body, &fiatRates); err != nil {
		return fmt.Errorf("failed to parse fiat API response: %w", err)
	}

	// Find USD to UAH rate (840 = USD, 980 = UAH)
	for _, rate := range fiatRates {
		if rate.CurrencyCodeA == 840 && rate.CurrencyCodeB == 980 {
			// Monobank: 1 USD = X UAH (where X is rateBuy, rateSell, or rateCross)
			// Standard format: 1 currency = exchangeRate USD
			// So we store: exchangeRate = 1 / X (USD per 1 UAH)
			// Example: if 1 USD = 42 UAH, then 1 UAH = 1/42 USD = 0.0238, store 0.0238
			var uahPerUsd float64
			if rate.RateCross != nil && *rate.RateCross > 0 {
				uahPerUsd = *rate.RateCross
			} else if rate.RateBuy > 0 && rate.RateSell > 0 {
				// Calculate mid rate from buy and sell
				uahPerUsd = (rate.RateBuy + rate.RateSell) / 2.0
			} else if rate.RateBuy > 0 {
				uahPerUsd = rate.RateBuy
			} else {
				return fmt.Errorf("no valid rate found for UAH")
			}

			// Convert: 1 USD = uahPerUsd UAH, so 1 UAH = 1/uahPerUsd USD
			usdPerUah := 1.0 / uahPerUsd
			if err := uc.currencyRepo.UpdateExchangeRate("UAH", usdPerUah); err != nil {
				return fmt.Errorf("failed to update UAH rate: %w", err)
			}
			break
		}
	}

	return nil
}
