package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Port     string
	Postgres PostgresConfig
	JWT      JWTConfig
	MinIO    MinIOConfig
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DB       string
}

func (c PostgresConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=notes,public",
		c.User, c.Password, c.Host, c.Port, c.DB)
}

type JWTConfig struct {
	Secret          string
	AccessDuration  time.Duration
	RefreshDuration time.Duration
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

func Load() Config {
	return Config{
		Port: getEnv("PORT", "8080"),
		Postgres: PostgresConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "notes"),
			Password: getEnv("POSTGRES_PASSWORD", "notes"),
			DB:       getEnv("POSTGRES_DB", "notes"),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "change-me-in-production"),
			AccessDuration:  parseDuration(getEnv("JWT_ACCESS_DURATION", "15m")),
			RefreshDuration: parseDuration(getEnv("JWT_REFRESH_DURATION", "168h")),
		},
		MinIO: MinIOConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:    getEnv("MINIO_BUCKET", "notes-images"),
			UseSSL:    getEnv("MINIO_USE_SSL", "false") == "true",
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
