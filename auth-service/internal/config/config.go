package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL string
	HTTPPort    string
	GRPCPort    string
}

func Load() (*Config, error) {
	viper.SetDefault("HTTP_PORT", "8080")
	viper.SetDefault("GRPC_PORT", "50051")

	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	// Ignore error if .env not found — rely on env vars
	_ = viper.ReadInConfig()

	cfg := &Config{
		DatabaseURL: viper.GetString("DATABASE_URL"),
		HTTPPort:    viper.GetString("HTTP_PORT"),
		GRPCPort:    viper.GetString("GRPC_PORT"),
	}

	return cfg, nil
}
