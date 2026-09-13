package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Mode string

const (
	ModePolling Mode = "polling"
	ModeWebhook Mode = "webhook"
)

type Bot struct {
	Name  string
	Token string
}

type Config struct {
	DatabaseURL string
	HTTPPort    string

	Mode          Mode
	PublicURL     string
	WebhookSecret string

	APIBase string

	TokenKey string

	AuthAddr    string
	FamilyAddr  string
	FinanceAddr string
	NotesAddr   string
	RecipesAddr string
	TasksAddr   string

	Bots []Bot

	PollTimeout time.Duration
	TasksTick   time.Duration
	CallTimeout time.Duration

	LogLevel string
	LogJSON  bool
}

var botNames = []string{"finance", "recipes", "notes", "family", "tasks"}

func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("TELEGRAM")
	v.AutomaticEnv()

	v.SetDefault("HTTP_PORT", "8080")
	v.SetDefault("MODE", string(ModePolling))
	v.SetDefault("POLL_TIMEOUT", 30*time.Second)
	v.SetDefault("CALL_TIMEOUT", 10*time.Second)
	v.SetDefault("TASKS_TICK", time.Minute)
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_JSON", false)

	cfg := &Config{
		DatabaseURL:   v.GetString("DATABASE_URL"),
		HTTPPort:      v.GetString("HTTP_PORT"),
		Mode:          Mode(strings.ToLower(v.GetString("MODE"))),
		PublicURL:     strings.TrimSuffix(v.GetString("PUBLIC_URL"), "/"),
		WebhookSecret: v.GetString("WEBHOOK_SECRET"),
		APIBase:       strings.TrimSuffix(v.GetString("API_BASE"), "/"),
		TokenKey:      v.GetString("TOKEN_KEY"),
		AuthAddr:      v.GetString("AUTH_ADDR"),
		FamilyAddr:    v.GetString("FAMILY_ADDR"),
		FinanceAddr:   v.GetString("FINANCE_ADDR"),
		NotesAddr:     v.GetString("NOTES_ADDR"),
		RecipesAddr:   v.GetString("RECIPES_ADDR"),
		TasksAddr:     v.GetString("TASKS_ADDR"),
		TasksTick:     v.GetDuration("TASKS_TICK"),
		PollTimeout:   v.GetDuration("POLL_TIMEOUT"),
		CallTimeout:   v.GetDuration("CALL_TIMEOUT"),
		LogLevel:      v.GetString("LOG_LEVEL"),
		LogJSON:       v.GetBool("LOG_JSON"),
	}

	if path := v.GetString("TOKEN_KEY_FILE"); path != "" {
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("config: read TELEGRAM_TOKEN_KEY_FILE: %w", err)
		}
		cfg.TokenKey = strings.TrimSpace(string(body))
	}

	for _, name := range botNames {
		token := strings.TrimSpace(v.GetString(strings.ToUpper(name) + "_TOKEN"))
		if path := v.GetString(strings.ToUpper(name) + "_TOKEN_FILE"); path != "" {
			body, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("config: read TELEGRAM_%s_TOKEN_FILE: %w",
					strings.ToUpper(name), err)
			}
			token = strings.TrimSpace(string(body))
		}
		if token == "" {
			continue
		}
		cfg.Bots = append(cfg.Bots, Bot{Name: name, Token: token})
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("config: TELEGRAM_DATABASE_URL is required")
	}
	if c.TokenKey == "" {
		return fmt.Errorf("config: TELEGRAM_TOKEN_KEY or TELEGRAM_TOKEN_KEY_FILE is required " +
			"(generate: openssl rand -base64 32)")
	}
	if c.AuthAddr == "" {
		return fmt.Errorf("config: TELEGRAM_AUTH_ADDR is required")
	}
	if c.FamilyAddr == "" {
		return fmt.Errorf("config: TELEGRAM_FAMILY_ADDR is required; " +
			"every bot reads shared user settings from the family service")
	}
	if len(c.Bots) == 0 {
		return fmt.Errorf("config: no bot tokens set; expected at least one of %s",
			strings.Join(tokenVars(), ", "))
	}
	for _, b := range c.Bots {
		if b.Name == "tasks" && c.TasksTick <= 0 {
			return fmt.Errorf("config: TELEGRAM_TASKS_TICK must be a positive duration such as 1m")
		}
	}
	switch c.Mode {
	case ModePolling:
	case ModeWebhook:
		if c.PublicURL == "" {
			return fmt.Errorf("config: TELEGRAM_PUBLIC_URL is required in webhook mode")
		}
		if c.WebhookSecret == "" {
			return fmt.Errorf("config: TELEGRAM_WEBHOOK_SECRET is required in webhook mode")
		}
	default:
		return fmt.Errorf("config: TELEGRAM_MODE must be %q or %q, got %q",
			ModePolling, ModeWebhook, c.Mode)
	}
	return nil
}

func tokenVars() []string {
	vars := make([]string, 0, len(botNames))
	for _, name := range botNames {
		vars = append(vars, "TELEGRAM_"+strings.ToUpper(name)+"_TOKEN")
	}
	return vars
}

func (c *Config) AddrFor(bot string) string {
	switch bot {
	case "finance":
		return c.FinanceAddr
	case "recipes":
		return c.RecipesAddr
	case "notes":
		return c.NotesAddr
	case "family":
		return c.FamilyAddr
	case "tasks":
		return c.TasksAddr
	default:
		return ""
	}
}
