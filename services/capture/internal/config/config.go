package config

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const sealKeyBytes = 32

type Config struct {
	DatabaseURL string
	HTTPPort    string
	GRPCPort    string
	NATSURL     string

	JWKSURL  string
	Issuer   string
	Audience string

	SealKey []byte

	Retention     time.Duration
	PurgeInterval time.Duration

	LogLevel string
	LogJSON  bool
}

func (c *Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("http_port", c.HTTPPort),
		slog.String("grpc_port", c.GRPCPort),
		slog.Bool("seal_key_set", len(c.SealKey) == sealKeyBytes),
		slog.Duration("retention", c.Retention),
		slog.Duration("purge_interval", c.PurgeInterval),
	)
}

func parseSealKey(raw string) ([]byte, error) {
	if raw == "" {
		return nil, fmt.Errorf("config: CAPTURE_SEAL_KEY is required (openssl rand -base64 32)")
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(key) != sealKeyBytes {
		return nil, fmt.Errorf("config: CAPTURE_SEAL_KEY must be 32 bytes, base64 (openssl rand -base64 32)")
	}
	return key, nil
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("CAPTURE")
	v.AutomaticEnv()

	v.SetDefault("HTTP_PORT", "8080")
	v.SetDefault("GRPC_PORT", "9090")
	v.SetDefault("NATS_URL", "nats://localhost:4222")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_JSON", false)
	v.SetDefault("ISSUER", "family-manager")
	v.SetDefault("AUDIENCE", "family-manager")
	v.SetDefault("RETENTION", "2160h")
	v.SetDefault("PURGE_INTERVAL", "1h")

	cfg := &Config{
		DatabaseURL:   v.GetString("DATABASE_URL"),
		HTTPPort:      v.GetString("HTTP_PORT"),
		GRPCPort:      v.GetString("GRPC_PORT"),
		NATSURL:       v.GetString("NATS_URL"),
		JWKSURL:       v.GetString("JWKS_URL"),
		Issuer:        v.GetString("ISSUER"),
		Audience:      v.GetString("AUDIENCE"),
		Retention:     v.GetDuration("RETENTION"),
		PurgeInterval: v.GetDuration("PURGE_INTERVAL"),
		LogLevel:      v.GetString("LOG_LEVEL"),
		LogJSON:       v.GetBool("LOG_JSON"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: CAPTURE_DATABASE_URL is required")
	}
	if cfg.JWKSURL == "" {
		return nil, fmt.Errorf("config: CAPTURE_JWKS_URL is required")
	}
	key, err := parseSealKey(strings.TrimSpace(v.GetString("SEAL_KEY")))
	if err != nil {
		return nil, err
	}
	cfg.SealKey = key
	if cfg.Retention <= 0 || cfg.PurgeInterval <= 0 {
		return nil, fmt.Errorf("config: CAPTURE_RETENTION and CAPTURE_PURGE_INTERVAL must be positive")
	}
	return cfg, nil
}
