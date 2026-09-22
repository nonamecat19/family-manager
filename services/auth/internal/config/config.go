package config

import (
	"fmt"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	DatabaseURL string
	HTTPPort    string
	GRPCPort    string

	SigningKeyPEM string
	Issuer        string
	Audience      string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration

	FamilyAddr string

	HashConcurrency int

	LoginFailureThreshold int
	LoginLockoutBase      time.Duration
	LoginLockoutMax       time.Duration

	DeviceLoginPerIP     int
	DeviceLoginWindow    time.Duration
	DeviceLoginMaxActive int64

	DeviceLoginPerNetwork int

	TrustedProxies []netip.Prefix
	TrustPeerXFF   bool

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
	v.SetDefault("DEVICE_LOGIN_PER_IP", 10)
	v.SetDefault("DEVICE_LOGIN_WINDOW", 10*time.Minute)
	v.SetDefault("DEVICE_LOGIN_MAX_ACTIVE", 100000)
	v.SetDefault("DEVICE_LOGIN_PER_NETWORK", 60)
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

		LoginFailureThreshold: v.GetInt("LOGIN_FAILURE_THRESHOLD"),
		LoginLockoutBase:      v.GetDuration("LOGIN_LOCKOUT_BASE"),
		LoginLockoutMax:       v.GetDuration("LOGIN_LOCKOUT_MAX"),

		DeviceLoginPerIP:     v.GetInt("DEVICE_LOGIN_PER_IP"),
		DeviceLoginWindow:    v.GetDuration("DEVICE_LOGIN_WINDOW"),
		DeviceLoginMaxActive: v.GetInt64("DEVICE_LOGIN_MAX_ACTIVE"),

		LogLevel: v.GetString("LOG_LEVEL"),
		LogJSON:  v.GetBool("LOG_JSON"),
	}

	if path := v.GetString("SIGNING_KEY_FILE"); path != "" {
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("config: read AUTH_SIGNING_KEY_FILE: %w", err)
		}
		cfg.SigningKeyPEM = string(body)
	}
	cfg.SigningKeyPEM = strings.ReplaceAll(cfg.SigningKeyPEM, `\n`, "\n")

	proxies, err := ParseProxies(v.GetString("TRUSTED_PROXIES"))
	if err != nil {
		return nil, err
	}
	cfg.TrustedProxies = proxies
	cfg.TrustPeerXFF = v.GetBool("TRUST_PEER_XFF")
	cfg.DeviceLoginPerNetwork = v.GetInt("DEVICE_LOGIN_PER_NETWORK")

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

func ParseProxies(list string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, raw := range strings.Split(list, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		p, err := netip.ParsePrefix(raw)
		if err != nil {
			return nil, fmt.Errorf("config: AUTH_TRUSTED_PROXIES: %w", err)
		}
		out = append(out, p.Masked())
	}
	return out, nil
}
