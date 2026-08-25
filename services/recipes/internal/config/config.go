// Package config loads the recipes service's settings from the environment, RECIPES_-prefixed.
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

	// JWKSURL is services/auth's public key set, used to verify access tokens.
	JWKSURL  string
	Issuer   string
	Audience string

	// Storage backs recipe image uploads: MinIO locally, Cloudflare R2 in production.
	// Unlike DatabaseURL/JWKSURL these have defaults — running without image support (Images
	// stays nil in the handler) is a valid degraded state, not a boot failure, so a missing
	// object store in dev doesn't block everything else.
	StorageProvider  string
	StorageEndpoint  string
	StorageAccessKey string
	StorageSecretKey string
	StorageUseSSL    bool
	StorageBucket    string
	// StoragePublicURL is the base URL the app fetches images from. Empty means "same as
	// StorageEndpoint", which is fine only for local MinIO; R2 requires it explicitly.
	StoragePublicURL string

	LogLevel string
	LogJSON  bool
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("RECIPES")
	v.AutomaticEnv()

	v.SetDefault("HTTP_PORT", "8080")
	v.SetDefault("GRPC_PORT", "9090")
	v.SetDefault("NATS_URL", "nats://localhost:4222")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_JSON", false)
	v.SetDefault("ISSUER", "family-manager")
	v.SetDefault("AUDIENCE", "family-manager")
	v.SetDefault("STORAGE_PROVIDER", "minio")
	v.SetDefault("STORAGE_BUCKET", "recipes")
	v.SetDefault("STORAGE_USE_SSL", false)

	cfg := &Config{
		DatabaseURL:      v.GetString("DATABASE_URL"),
		HTTPPort:         v.GetString("HTTP_PORT"),
		GRPCPort:         v.GetString("GRPC_PORT"),
		NATSURL:          v.GetString("NATS_URL"),
		JWKSURL:          v.GetString("JWKS_URL"),
		Issuer:           v.GetString("ISSUER"),
		Audience:         v.GetString("AUDIENCE"),
		StorageProvider:  v.GetString("STORAGE_PROVIDER"),
		StorageEndpoint:  v.GetString("STORAGE_ENDPOINT"),
		StorageAccessKey: v.GetString("STORAGE_ACCESS_KEY"),
		StorageSecretKey: v.GetString("STORAGE_SECRET_KEY"),
		StorageUseSSL:    v.GetBool("STORAGE_USE_SSL"),
		StorageBucket:    v.GetString("STORAGE_BUCKET"),
		StoragePublicURL: v.GetString("STORAGE_PUBLIC_URL"),
		LogLevel:         v.GetString("LOG_LEVEL"),
		LogJSON:          v.GetBool("LOG_JSON"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: RECIPES_DATABASE_URL is required")
	}
	if cfg.JWKSURL == "" {
		return nil, fmt.Errorf("config: RECIPES_JWKS_URL is required")
	}
	return cfg, nil
}
