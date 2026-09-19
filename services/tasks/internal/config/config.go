package config

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"time"

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

	FamilyAddr string

	DefaultTimezone string

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	TokenKey           string

	ReminderTick time.Duration
	SyncTick     time.Duration

	LogLevel string
	LogJSON  bool
}

func (c *Config) GoogleConfigured() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != "" && c.GoogleRedirectURL != "" && c.TokenKey != ""
}

func (c *Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("http_port", c.HTTPPort),
		slog.String("grpc_port", c.GRPCPort),
		slog.String("family_addr", c.FamilyAddr),
		slog.String("default_timezone", c.DefaultTimezone),
		slog.Bool("google_configured", c.GoogleConfigured()),
		slog.Duration("reminder_tick", c.ReminderTick),
		slog.Duration("sync_tick", c.SyncTick),
	)
}

func (c *Config) validateGoogle() error {
	set := 0
	for _, v := range []string{c.GoogleClientID, c.GoogleClientSecret, c.GoogleRedirectURL, c.TokenKey} {
		if v != "" {
			set++
		}
	}
	if set == 0 {
		return nil
	}
	if set != 4 {
		return fmt.Errorf("config: set all of TASKS_GOOGLE_CLIENT_ID, TASKS_GOOGLE_CLIENT_SECRET, " +
			"TASKS_GOOGLE_REDIRECT_URL and TASKS_TOKEN_KEY, or none of them")
	}
	key, err := base64.StdEncoding.DecodeString(c.TokenKey)
	if err != nil || len(key) != 32 {
		return fmt.Errorf("config: TASKS_TOKEN_KEY must be 32 bytes, base64 (openssl rand -base64 32)")
	}
	return nil
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("TASKS")
	v.AutomaticEnv()

	v.SetDefault("HTTP_PORT", "8080")
	v.SetDefault("GRPC_PORT", "9090")
	v.SetDefault("NATS_URL", "nats://localhost:4222")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_JSON", false)
	v.SetDefault("ISSUER", "family-manager")
	v.SetDefault("AUDIENCE", "family-manager")
	v.SetDefault("DEFAULT_TIMEZONE", "Europe/Kyiv")
	v.SetDefault("REMINDER_TICK", "1m")
	v.SetDefault("SYNC_TICK", "5m")

	cfg := &Config{
		DatabaseURL:        v.GetString("DATABASE_URL"),
		HTTPPort:           v.GetString("HTTP_PORT"),
		GRPCPort:           v.GetString("GRPC_PORT"),
		NATSURL:            v.GetString("NATS_URL"),
		JWKSURL:            v.GetString("JWKS_URL"),
		Issuer:             v.GetString("ISSUER"),
		Audience:           v.GetString("AUDIENCE"),
		FamilyAddr:         strings.TrimSuffix(v.GetString("FAMILY_ADDR"), "/"),
		DefaultTimezone:    v.GetString("DEFAULT_TIMEZONE"),
		GoogleClientID:     strings.TrimSpace(v.GetString("GOOGLE_CLIENT_ID")),
		GoogleClientSecret: strings.TrimSpace(v.GetString("GOOGLE_CLIENT_SECRET")),
		GoogleRedirectURL:  strings.TrimSpace(v.GetString("GOOGLE_REDIRECT_URL")),
		TokenKey:           strings.TrimSpace(v.GetString("TOKEN_KEY")),
		ReminderTick:       v.GetDuration("REMINDER_TICK"),
		SyncTick:           v.GetDuration("SYNC_TICK"),
		LogLevel:           v.GetString("LOG_LEVEL"),
		LogJSON:            v.GetBool("LOG_JSON"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: TASKS_DATABASE_URL is required")
	}
	if cfg.JWKSURL == "" {
		return nil, fmt.Errorf("config: TASKS_JWKS_URL is required")
	}
	if cfg.FamilyAddr == "" {
		return nil, fmt.Errorf("config: TASKS_FAMILY_ADDR is required")
	}
	if _, err := time.LoadLocation(cfg.DefaultTimezone); err != nil {
		return nil, fmt.Errorf("config: TASKS_DEFAULT_TIMEZONE %q: %w", cfg.DefaultTimezone, err)
	}
	if cfg.ReminderTick <= 0 || cfg.SyncTick <= 0 {
		return nil, fmt.Errorf("config: TASKS_REMINDER_TICK and TASKS_SYNC_TICK must be positive")
	}
	if err := cfg.validateGoogle(); err != nil {
		return nil, err
	}
	return cfg, nil
}
