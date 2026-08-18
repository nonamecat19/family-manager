// Package config loads the auth service's settings from the environment, AUTH_-prefixed.
// Secrets have no defaults: a missing one crashes at boot, because an auth service that
// silently invents a signing key is worse than one that does not start.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL string
	HTTPPort    string
	GRPCPort    string

	// SigningKeyPEM is the ES256 private key. Supplied inline (AUTH_SIGNING_KEY) or via a
	// file path (AUTH_SIGNING_KEY_FILE), which is how a Docker or Kubernetes secret arrives.
	SigningKeyPEM string
	Issuer        string
	Audience      string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration

	// FamilyAddr is services/family's internal listener, used to resolve the family_id claim
	// while minting a token. Empty disables the lookup: tokens are then minted without a
	// family_id and the apps show onboarding.
	FamilyAddr string

	// HashConcurrency caps how many argon2id hashes run at once. Each holds ~19 MiB for its
	// duration and both Register and Login are unauthenticated, so this is the setting that
	// decides whether a burst of sign-ins queues or exhausts the box. Zero means GOMAXPROCS.
	HashConcurrency int

	LogLevel string
	LogJSON  bool
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("AUTH")
	v.AutomaticEnv()

	v.SetDefault("HTTP_PORT", "8080")
	v.SetDefault("GRPC_PORT", "9090")
	v.SetDefault("ISSUER", "family-manager")
	v.SetDefault("AUDIENCE", "family-manager")
	v.SetDefault("ACCESS_TTL", 15*time.Minute)
	v.SetDefault("REFRESH_TTL", 30*24*time.Hour)
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_JSON", false)

	cfg := &Config{
		DatabaseURL:   v.GetString("DATABASE_URL"),
		HTTPPort:      v.GetString("HTTP_PORT"),
		GRPCPort:      v.GetString("GRPC_PORT"),
		SigningKeyPEM: v.GetString("SIGNING_KEY"),
		Issuer:        v.GetString("ISSUER"),
		Audience:      v.GetString("AUDIENCE"),
		AccessTTL:     v.GetDuration("ACCESS_TTL"),
		RefreshTTL:    v.GetDuration("REFRESH_TTL"),
		FamilyAddr:    v.GetString("FAMILY_ADDR"),

		HashConcurrency: v.GetInt("HASH_CONCURRENCY"),
		LogLevel:        v.GetString("LOG_LEVEL"),
		LogJSON:         v.GetBool("LOG_JSON"),
	}

	if path := v.GetString("SIGNING_KEY_FILE"); path != "" {
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("config: read AUTH_SIGNING_KEY_FILE: %w", err)
		}
		cfg.SigningKeyPEM = string(body)
	}
	// Env vars cannot hold real newlines conveniently, so accept the escaped form too.
	cfg.SigningKeyPEM = strings.ReplaceAll(cfg.SigningKeyPEM, `\n`, "\n")

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: AUTH_DATABASE_URL is required")
	}
	if strings.TrimSpace(cfg.SigningKeyPEM) == "" {
		return nil, fmt.Errorf(
			"config: AUTH_SIGNING_KEY or AUTH_SIGNING_KEY_FILE is required " +
				"(generate: openssl ecparam -name prime256v1 -genkey -noout)")
	}
	return cfg, nil
}
