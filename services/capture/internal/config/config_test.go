package config

import (
	"strings"
	"testing"
	"time"
)

const testKey = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

func base(t *testing.T) {
	t.Helper()
	t.Setenv("CAPTURE_DATABASE_URL", "postgres://x")
	t.Setenv("CAPTURE_JWKS_URL", "http://auth/.well-known/jwks.json")
	t.Setenv("CAPTURE_SEAL_KEY", testKey)
}

func TestLoadDefaults(t *testing.T) {
	base(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPPort != "8080" || cfg.GRPCPort != "9090" {
		t.Fatalf("unexpected ports: %s %s", cfg.HTTPPort, cfg.GRPCPort)
	}
	if cfg.Retention != 90*24*time.Hour {
		t.Fatalf("retention = %s, want 90 days", cfg.Retention)
	}
	if cfg.PurgeInterval <= 0 {
		t.Fatalf("purge interval = %s, want positive", cfg.PurgeInterval)
	}
	if len(cfg.SealKey) != 32 {
		t.Fatalf("seal key length = %d, want 32", len(cfg.SealKey))
	}
	if cfg.Issuer != "family-manager" || cfg.Audience != "family-manager" {
		t.Fatalf("unexpected jwt defaults: %s %s", cfg.Issuer, cfg.Audience)
	}
}

func TestLoadRequiresDatabaseJWKSAndSealKey(t *testing.T) {
	for _, missing := range []string{"CAPTURE_DATABASE_URL", "CAPTURE_JWKS_URL", "CAPTURE_SEAL_KEY"} {
		t.Run(missing, func(t *testing.T) {
			base(t)
			t.Setenv(missing, "")
			if _, err := Load(); err == nil {
				t.Fatalf("expected an error without %s", missing)
			}
		})
	}
}

func TestSealKeyMustBe32BytesBase64(t *testing.T) {
	for name, key := range map[string]string{
		"short":      "c2hvcnQ=",
		"not base64": "not-base64-at-all!!",
		"too long":   "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWYwMTIz",
	} {
		t.Run(name, func(t *testing.T) {
			base(t)
			t.Setenv("CAPTURE_SEAL_KEY", key)
			if _, err := Load(); err == nil {
				t.Fatalf("expected an error for seal key %q", key)
			}
		})
	}
}

func TestDurationsMustBePositive(t *testing.T) {
	for _, name := range []string{"CAPTURE_RETENTION", "CAPTURE_PURGE_INTERVAL"} {
		t.Run(name, func(t *testing.T) {
			base(t)
			t.Setenv(name, "0s")
			if _, err := Load(); err == nil {
				t.Fatalf("expected an error for zero %s", name)
			}
		})
	}
}

func TestDurationsOverride(t *testing.T) {
	base(t)
	t.Setenv("CAPTURE_RETENTION", "24h")
	t.Setenv("CAPTURE_PURGE_INTERVAL", "5m")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Retention != 24*time.Hour || cfg.PurgeInterval != 5*time.Minute {
		t.Fatalf("unexpected durations: %s %s", cfg.Retention, cfg.PurgeInterval)
	}
}

func TestLogValueHidesSealKey(t *testing.T) {
	base(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	out := cfg.LogValue().String()
	if strings.Contains(out, testKey) || strings.Contains(out, string(cfg.SealKey)) || strings.Contains(out, "postgres://") {
		t.Fatalf("log value leaks a secret: %s", out)
	}
}
