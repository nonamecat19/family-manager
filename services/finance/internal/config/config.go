// Package config loads the finance service's settings from the environment, FINANCE_-prefixed.
// Secrets have no defaults: a missing one crashes at boot rather than silently degrading.
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL string
	HTTPPort    string
	GRPCPort    string
	NATSURL     string

	JWKSURL  string
	Issuer   string
	Audience string

	LogLevel string
	LogJSON  bool

	// BaseCurrency is the currency family totals are reported in when accounts differ.
	BaseCurrency string
	// MaxPageSize caps ListTransactions regardless of what a client asks for.
	MaxPageSize int32
	// DefaultPageSize applies when a client asks for none.
	DefaultPageSize int32
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("FINANCE")
	v.AutomaticEnv()

	v.SetDefault("HTTP_PORT", "8080")
	v.SetDefault("GRPC_PORT", "9090")
	v.SetDefault("NATS_URL", "nats://localhost:4222")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_JSON", false)
	v.SetDefault("ISSUER", "family-manager")
	v.SetDefault("AUDIENCE", "family-manager")
	v.SetDefault("BASE_CURRENCY", "EUR")
	v.SetDefault("MAX_PAGE_SIZE", 200)
	v.SetDefault("DEFAULT_PAGE_SIZE", 50)

	cfg := &Config{
		DatabaseURL:     v.GetString("DATABASE_URL"),
		HTTPPort:        v.GetString("HTTP_PORT"),
		GRPCPort:        v.GetString("GRPC_PORT"),
		NATSURL:         v.GetString("NATS_URL"),
		JWKSURL:         v.GetString("JWKS_URL"),
		Issuer:          v.GetString("ISSUER"),
		Audience:        v.GetString("AUDIENCE"),
		LogLevel:        v.GetString("LOG_LEVEL"),
		LogJSON:         v.GetBool("LOG_JSON"),
		BaseCurrency:    v.GetString("BASE_CURRENCY"),
		MaxPageSize:     v.GetInt32("MAX_PAGE_SIZE"),
		DefaultPageSize: v.GetInt32("DEFAULT_PAGE_SIZE"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: FINANCE_DATABASE_URL is required")
	}
	if cfg.JWKSURL == "" {
		return nil, fmt.Errorf("config: FINANCE_JWKS_URL is required")
	}
	return cfg, nil
}
