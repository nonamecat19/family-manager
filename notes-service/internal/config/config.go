package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Port       string
	Postgres   PostgresConfig
	JWT        JWTConfig
	VercelBlob VercelBlobConfig
}

type PostgresConfig struct {
	Host       string
	Port       string
	User       string
	Password   string
	DB         string
	SSLMode    string
	SchemaName string
}

func (c PostgresConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&search_path=%s,public",
		c.User, c.Password, c.Host, c.Port, c.DB, c.SSLMode, c.SchemaName)
}

type JWTConfig struct {
	Secret          string
	AccessDuration  time.Duration
	RefreshDuration time.Duration
}

type VercelBlobConfig struct {
	Token string
}

func Load() Config {
	return Config{
		Port: getEnv("PORT", "8080"),
		Postgres: PostgresConfig{
			Host:       getEnv("POSTGRES_HOST", "localhost"),
			Port:       getEnv("POSTGRES_PORT", "5432"),
			User:       getEnv("POSTGRES_USER", "notes"),
			Password:   getEnv("POSTGRES_PASSWORD", "notes"),
			DB:         getEnv("POSTGRES_DB", "notes"),
			SSLMode:    getEnv("POSTGRES_SSL_MODE", "disable"),
			SchemaName: getEnv("POSTGRES_SCHEMA", "notes"),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "change-me-in-production"),
			AccessDuration:  parseDuration(getEnv("JWT_ACCESS_DURATION", "15m")),
			RefreshDuration: parseDuration(getEnv("JWT_REFRESH_DURATION", "168h")),
		},
		VercelBlob: VercelBlobConfig{
			Token: getEnv("BLOB_READ_WRITE_TOKEN", ""),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 15 * time.Minute
	}
	return d
}
