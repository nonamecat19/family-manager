package storage

import (
	"strings"
	"testing"
)

func validMinIO() Config {
	return Config{Endpoint: "minio:9000", AccessKey: "key", SecretKey: "secret"}
}

func TestNewRequiresEndpointAndCredentials(t *testing.T) {
	cases := map[string]Config{
		"no endpoint":   {AccessKey: "key", SecretKey: "secret"},
		"no access key": {Endpoint: "minio:9000", SecretKey: "secret"},
		"no secret key": {Endpoint: "minio:9000", AccessKey: "key"},
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			// A client built with a blank credential fails on first upload instead of at
			// boot, which turns a misconfigured deploy into a broken feature nobody notices
			// until someone tries to add a photo.
			if _, err := New(cfg); err == nil {
				t.Fatal("New() = nil error, want a refusal")
			}
		})
	}
}

func TestNewDefaultsToMinIO(t *testing.T) {
	c, err := New(validMinIO())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.provider != ProviderMinIO {
		t.Fatalf("provider = %q, want %q", c.provider, ProviderMinIO)
	}
}

func TestNewRejectsAnUnknownProvider(t *testing.T) {
	cfg := validMinIO()
	cfg.Provider = "s3"
	if _, err := New(cfg); err == nil {
		t.Fatal("New() accepted an unknown provider")
	}
}

// R2's S3 endpoint serves no public reads at all, so a missing PublicURL there produces image
// URLs that are guaranteed to 404 — the failure appears in the app, far from the config.
func TestR2RequiresAPublicURL(t *testing.T) {
	cfg := validMinIO()
	cfg.Provider = ProviderR2
	if _, err := New(cfg); err == nil {
		t.Fatal("New() accepted an R2 config with no PublicURL")
	}
}

func TestMinIOFallsBackToTheEndpointForPublicURL(t *testing.T) {
	cfg := validMinIO()
	c, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.publicURL != "http://minio:9000" {
		t.Fatalf("publicURL = %q, want the endpoint with a scheme", c.publicURL)
	}

	cfg.UseSSL = true
	c, err = New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if !strings.HasPrefix(c.publicURL, "https://") {
		t.Fatalf("publicURL = %q, want https when UseSSL is set", c.publicURL)
	}
}

func TestPublicURLLosesItsTrailingSlash(t *testing.T) {
	cfg := validMinIO()
	cfg.PublicURL = "https://images.example.test/"
	c, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := c.objectURL("recipes", "fam/rec.jpg"); got != "https://images.example.test/recipes/fam/rec.jpg" {
		t.Fatalf("objectURL = %q, want no doubled slash", got)
	}
}

// The two providers disagree about the bucket, and getting it wrong produces a 404 for every
// image rather than an error anything logs.
func TestObjectURLShapePerProvider(t *testing.T) {
	minio, err := New(Config{
		Endpoint: "minio:9000", AccessKey: "k", SecretKey: "s",
		PublicURL: "http://localhost:9000",
	})
	if err != nil {
		t.Fatalf("New(minio): %v", err)
	}
	if got, want := minio.objectURL("recipes", "fam/rec.jpg"),
		"http://localhost:9000/recipes/fam/rec.jpg"; got != want {
		t.Errorf("minio objectURL = %q, want %q", got, want)
	}

	r2, err := New(Config{
		Provider: ProviderR2, Endpoint: "acct.r2.cloudflarestorage.com",
		AccessKey: "k", SecretKey: "s", PublicURL: "https://images.example.test",
	})
	if err != nil {
		t.Fatalf("New(r2): %v", err)
	}
	// An R2 public domain is bound to one bucket, so repeating the bucket is a 404.
	if got, want := r2.objectURL("recipes", "fam/rec.jpg"),
		"https://images.example.test/fam/rec.jpg"; got != want {
		t.Errorf("r2 objectURL = %q, want %q", got, want)
	}
}
