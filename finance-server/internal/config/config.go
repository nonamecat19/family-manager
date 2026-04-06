package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	DatabaseURL string
	Port        string
	GinMode     string
	JWTSecret   string
}

// Load reads environment variables and returns a Config.
// It attempts to load a .env file first but does not fail if missing.
func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://finance:finance_dev@localhost:5432/finance_tracker?sslmode=disable"),
		Port:        getEnv("PORT", "8080"),
		GinMode:     getEnv("GIN_MODE", "debug"),
		JWTSecret:   getEnv("JWT_SECRET", "dev-jwt-secret-change-in-production"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
