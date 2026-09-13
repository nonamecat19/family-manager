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

	BaseCurrency string
	Timezone     string

	LogLevel string
	LogJSON  bool
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
	v.SetDefault("BASE_CURRENCY", "UAH")
	v.SetDefault("TIMEZONE", "Europe/Kyiv")

	cfg := &Config{
		DatabaseURL:  v.GetString("DATABASE_URL"),
		HTTPPort:     v.GetString("HTTP_PORT"),
		GRPCPort:     v.GetString("GRPC_PORT"),
		NATSURL:      v.GetString("NATS_URL"),
		JWKSURL:      v.GetString("JWKS_URL"),
		Issuer:       v.GetString("ISSUER"),
		Audience:     v.GetString("AUDIENCE"),
		BaseCurrency: v.GetString("BASE_CURRENCY"),
		Timezone:     v.GetString("TIMEZONE"),
		LogLevel:     v.GetString("LOG_LEVEL"),
		LogJSON:      v.GetBool("LOG_JSON"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: FINANCE_DATABASE_URL is required")
	}
	if cfg.JWKSURL == "" {
		return nil, fmt.Errorf("config: FINANCE_JWKS_URL is required")
	}
	return cfg, nil
}
