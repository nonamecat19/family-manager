// Package config loads the family service's settings from the environment, FAMILY_-prefixed.
// Secrets have no defaults: a missing one crashes at boot rather than silently degrading.
package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL string
	HTTPPort    string
	GRPCPort    string
	NATSURL     string

	// JWKSURL is services/auth's public key set, used to verify access tokens.
	JWKSURL  string
	Issuer   string
	Audience string

	LogLevel string
	LogJSON  bool

	// InvitationTTL bounds how long an invite link stays usable.
	InvitationTTL time.Duration
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("FAMILY")
	v.AutomaticEnv()

	v.SetDefault("HTTP_PORT", "8080")
	v.SetDefault("GRPC_PORT", "9090")
	v.SetDefault("NATS_URL", "nats://localhost:4222")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_JSON", false)
	v.SetDefault("ISSUER", "family-manager")
	v.SetDefault("AUDIENCE", "family-manager")
	v.SetDefault("INVITATION_TTL", 7*24*time.Hour)

	cfg := &Config{
		DatabaseURL:   v.GetString("DATABASE_URL"),
		HTTPPort:      v.GetString("HTTP_PORT"),
		GRPCPort:      v.GetString("GRPC_PORT"),
		NATSURL:       v.GetString("NATS_URL"),
		JWKSURL:       v.GetString("JWKS_URL"),
		Issuer:        v.GetString("ISSUER"),
		Audience:      v.GetString("AUDIENCE"),
		LogLevel:      v.GetString("LOG_LEVEL"),
		LogJSON:       v.GetBool("LOG_JSON"),
		InvitationTTL: v.GetDuration("INVITATION_TTL"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: FAMILY_DATABASE_URL is required")
	}
	if cfg.JWKSURL == "" {
		return nil, fmt.Errorf("config: FAMILY_JWKS_URL is required")
	}
	return cfg, nil
}
