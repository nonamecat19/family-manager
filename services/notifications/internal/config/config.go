package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL string
	HTTPPort    string
	NATSURL     string
	FamilyAddr  string

	JWKSURL  string
	Issuer   string
	Audience string

	ExpoURL         string
	ExpoAccessToken string
	ReceiptInterval time.Duration
	ReceiptDelay    time.Duration

	LogLevel string
	LogJSON  bool
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("NOTIFICATIONS")
	v.AutomaticEnv()

	v.SetDefault("HTTP_PORT", "8080")
	v.SetDefault("NATS_URL", "nats://localhost:4222")
	v.SetDefault("FAMILY_ADDR", "http://localhost:9092")
	v.SetDefault("ISSUER", "family-manager")
	v.SetDefault("AUDIENCE", "family-manager")
	v.SetDefault("EXPO_URL", "https://exp.host/--/api/v2/push")
	v.SetDefault("RECEIPT_INTERVAL", 5*time.Minute)
	v.SetDefault("RECEIPT_DELAY", 15*time.Minute)
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_JSON", false)

	cfg := &Config{
		DatabaseURL:     v.GetString("DATABASE_URL"),
		HTTPPort:        v.GetString("HTTP_PORT"),
		NATSURL:         v.GetString("NATS_URL"),
		FamilyAddr:      v.GetString("FAMILY_ADDR"),
		JWKSURL:         v.GetString("JWKS_URL"),
		Issuer:          v.GetString("ISSUER"),
		Audience:        v.GetString("AUDIENCE"),
		ExpoURL:         v.GetString("EXPO_URL"),
		ExpoAccessToken: v.GetString("EXPO_ACCESS_TOKEN"),
		ReceiptInterval: v.GetDuration("RECEIPT_INTERVAL"),
		ReceiptDelay:    v.GetDuration("RECEIPT_DELAY"),
		LogLevel:        v.GetString("LOG_LEVEL"),
		LogJSON:         v.GetBool("LOG_JSON"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: NOTIFICATIONS_DATABASE_URL is required")
	}
	if cfg.JWKSURL == "" {
		return nil, fmt.Errorf("config: NOTIFICATIONS_JWKS_URL is required")
	}
	if cfg.ReceiptInterval <= 0 {
		return nil, fmt.Errorf("config: NOTIFICATIONS_RECEIPT_INTERVAL must be positive")
	}
	return cfg, nil
}
