package config

import (
	"fmt"
	"os"
	"strings"
)

type Endpoints struct {
	Auth  string
	Notes string
}

var presets = map[string]Endpoints{
	"production": {
		Auth:  "https://auth.nonamecat.pp.ua",
		Notes: "https://notes.nonamecat.pp.ua",
	},
	"local": {
		Auth:  "http://localhost:8081",
		Notes: "http://localhost:8085",
	},
}

type Config struct {
	Env       string
	Endpoints Endpoints
}

func Resolve(env string, getenv func(string) string) (Config, error) {
	if env == "" {
		env = getenv("FM_ENV")
	}
	if env == "" {
		env = "production"
	}
	ep, ok := presets[env]
	if !ok {
		return Config{}, fmt.Errorf("config: FM_ENV must be production or local, got %q", env)
	}
	if base := strings.TrimRight(getenv("FM_API_URL"), "/"); base != "" {
		ep = Endpoints{Auth: base, Notes: base}
	}
	if v := strings.TrimRight(getenv("FM_AUTH_URL"), "/"); v != "" {
		ep.Auth = v
	}
	if v := strings.TrimRight(getenv("FM_NOTES_URL"), "/"); v != "" {
		ep.Notes = v
	}
	return Config{Env: env, Endpoints: ep}, nil
}

func FromEnv(env string) (Config, error) {
	return Resolve(env, os.Getenv)
}
