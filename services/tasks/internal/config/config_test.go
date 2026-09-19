package config

import (
	"strings"
	"testing"
)

func base(t *testing.T) {
	t.Helper()
	t.Setenv("TASKS_DATABASE_URL", "postgres://x")
	t.Setenv("TASKS_JWKS_URL", "http://auth/.well-known/jwks.json")
	t.Setenv("TASKS_FAMILY_ADDR", "http://family:9090/")
}

func TestLoadDefaults(t *testing.T) {
	base(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPPort != "8080" || cfg.DefaultTimezone != "Europe/Kyiv" || cfg.FamilyAddr != "http://family:9090" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.GoogleConfigured() {
		t.Fatal("google must not be configured without credentials")
	}
}

func TestLoadRequiresDatabaseJWKSAndFamily(t *testing.T) {
	for _, missing := range []string{"TASKS_DATABASE_URL", "TASKS_JWKS_URL", "TASKS_FAMILY_ADDR"} {
		t.Run(missing, func(t *testing.T) {
			base(t)
			t.Setenv(missing, "")
			if _, err := Load(); err == nil {
				t.Fatalf("expected an error without %s", missing)
			}
		})
	}
}

func TestLoadRejectsBadTimezone(t *testing.T) {
	base(t)
	t.Setenv("TASKS_DEFAULT_TIMEZONE", "Mars/Olympus_Mons")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error for an unknown timezone")
	}
}

const testKey = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

func google(t *testing.T) {
	t.Helper()
	t.Setenv("TASKS_GOOGLE_CLIENT_ID", "id")
	t.Setenv("TASKS_GOOGLE_CLIENT_SECRET", "secret")
	t.Setenv("TASKS_GOOGLE_REDIRECT_URL", "fmtasks:/oauthredirect")
	t.Setenv("TASKS_TOKEN_KEY", testKey)
}

func TestGoogleConfiguredWithAllFour(t *testing.T) {
	base(t)
	google(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.GoogleConfigured() {
		t.Fatal("google should be configured")
	}
}

func TestPartialGoogleConfigIsRejected(t *testing.T) {
	for _, missing := range []string{"TASKS_GOOGLE_CLIENT_ID", "TASKS_GOOGLE_CLIENT_SECRET", "TASKS_GOOGLE_REDIRECT_URL", "TASKS_TOKEN_KEY"} {
		t.Run(missing, func(t *testing.T) {
			base(t)
			google(t)
			t.Setenv(missing, "")
			if _, err := Load(); err == nil {
				t.Fatalf("expected an error with %s missing", missing)
			}
		})
	}
}

func TestTokenKeyMustBe32Bytes(t *testing.T) {
	base(t)
	google(t)
	t.Setenv("TASKS_TOKEN_KEY", "c2hvcnQ=")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error for a short key")
	}
}

func TestLogValueHidesSecrets(t *testing.T) {
	base(t)
	google(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	out := cfg.LogValue().String()
	if strings.Contains(out, "secret") || strings.Contains(out, testKey) {
		t.Fatalf("log value leaks a secret: %s", out)
	}
}
