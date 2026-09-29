package config

import "testing"

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestResolveDefaultsToProduction(t *testing.T) {
	c, err := Resolve("", env(nil))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if c.Env != "production" || c.Endpoints.Auth != "https://auth.nonamecat.pp.ua" || c.Endpoints.Notes != "https://notes.nonamecat.pp.ua" {
		t.Fatalf("config = %+v", c)
	}
}

func TestResolveLocalAndOverrides(t *testing.T) {
	c, err := Resolve("", env(map[string]string{"FM_ENV": "local", "FM_NOTES_URL": "http://notes.test/"}))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if c.Endpoints.Auth != "http://localhost:8081" || c.Endpoints.Notes != "http://notes.test" {
		t.Fatalf("config = %+v", c)
	}
}

func TestResolveExplicitEnvBeatsVariable(t *testing.T) {
	c, err := Resolve("local", env(map[string]string{"FM_ENV": "production"}))
	if err != nil || c.Env != "local" {
		t.Fatalf("config = %+v, err = %v", c, err)
	}
}

func TestResolveRejectsUnknownEnv(t *testing.T) {
	if _, err := Resolve("staging", env(nil)); err == nil {
		t.Fatal("Resolve(staging) succeeded")
	}
}
