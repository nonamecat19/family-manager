package config

import (
	"strings"
	"testing"
)

func env(t *testing.T, pairs map[string]string) {
	t.Helper()
	base := map[string]string{
		"TELEGRAM_DATABASE_URL":  "postgres://localhost/telegram",
		"TELEGRAM_TOKEN_KEY":     "Zm9vYmFyZm9vYmFyZm9vYmFyZm9vYmFyMzI=",
		"TELEGRAM_AUTH_ADDR":     "http://auth:8080",
		"TELEGRAM_FAMILY_ADDR":   "http://family:8080",
		"TELEGRAM_FINANCE_ADDR":  "http://finance:8080",
		"TELEGRAM_FINANCE_TOKEN": "123:abc",
	}
	for k, v := range pairs {
		base[k] = v
	}
	for k, v := range base {
		if v == "" {
			continue
		}
		t.Setenv(k, v)
	}
}

func TestLoadDefaultsToPolling(t *testing.T) {
	env(t, nil)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Mode != ModePolling {
		t.Fatalf("mode = %q, want %q", cfg.Mode, ModePolling)
	}
	if len(cfg.Bots) != 1 || cfg.Bots[0].Name != "finance" {
		t.Fatalf("bots = %+v, want one finance bot", cfg.Bots)
	}
}

func TestWebhookModeNeedsAPublicURLAndSecret(t *testing.T) {
	env(t, map[string]string{"TELEGRAM_MODE": "webhook"})

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "TELEGRAM_PUBLIC_URL") {
		t.Fatalf("err = %v, want a complaint about TELEGRAM_PUBLIC_URL", err)
	}

	t.Setenv("TELEGRAM_PUBLIC_URL", "https://bots.example.test/")
	_, err = Load()
	if err == nil || !strings.Contains(err.Error(), "TELEGRAM_WEBHOOK_SECRET") {
		t.Fatalf("err = %v, want a complaint about TELEGRAM_WEBHOOK_SECRET", err)
	}

	t.Setenv("TELEGRAM_WEBHOOK_SECRET", "shhh")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PublicURL != "https://bots.example.test" {
		t.Fatalf("public url = %q, want the trailing slash gone", cfg.PublicURL)
	}
}

func TestLoadRejectsAnUnknownMode(t *testing.T) {
	env(t, map[string]string{"TELEGRAM_MODE": "carrier-pigeon"})

	if _, err := Load(); err == nil {
		t.Fatal("Load accepted an unknown mode")
	}
}

func TestLoadRequiresATokenKey(t *testing.T) {
	env(t, nil)
	t.Setenv("TELEGRAM_TOKEN_KEY", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load accepted a missing token key")
	}
}

func TestLoadRequiresAtLeastOneBot(t *testing.T) {
	env(t, nil)
	t.Setenv("TELEGRAM_FINANCE_TOKEN", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "no bot tokens set") {
		t.Fatalf("err = %v, want a complaint about missing bot tokens", err)
	}
}

func TestAddrForMapsEachBot(t *testing.T) {
	cfg := &Config{
		FinanceAddr: "finance",
		RecipesAddr: "recipes",
		NotesAddr:   "notes",
		FamilyAddr:  "family",
	}
	for _, name := range botNames {
		if cfg.AddrFor(name) != name {
			t.Fatalf("AddrFor(%q) = %q", name, cfg.AddrFor(name))
		}
	}
	if cfg.AddrFor("shopping") != "" {
		t.Fatal("AddrFor returned an address for an unknown bot")
	}
}

func TestLoadRequiresAFamilyAddress(t *testing.T) {
	env(t, nil)
	t.Setenv("TELEGRAM_FAMILY_ADDR", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "TELEGRAM_FAMILY_ADDR") {
		t.Fatalf("err = %v, want a complaint about TELEGRAM_FAMILY_ADDR", err)
	}
}
