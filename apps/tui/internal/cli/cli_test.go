package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nnc/family-manager/apps/tui/internal/credentials"
)

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	t.Setenv("FM_ENV", "local")
	t.Setenv("FM_API_URL", "")
	t.Setenv("FM_AUTH_URL", "http://127.0.0.1:1")
	t.Setenv("FM_NOTES_URL", "http://127.0.0.1:1")
	var out, errOut bytes.Buffer
	code := Run(context.Background(), args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestWhoAmIDecodesStoredToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.json")
	exp := time.Now().Add(10 * time.Minute)
	enc := base64.RawURLEncoding
	payload := fmt.Sprintf(`{"sub":"user-1","family_id":"family-1","email":"a@example.com","exp":%d}`, exp.Unix())
	tok := enc.EncodeToString([]byte(`{}`)) + "." + enc.EncodeToString([]byte(payload)) + ".sig"
	if err := (&credentials.Store{Path: path}).Save(credentials.Credentials{AccessToken: tok, RefreshToken: "r", ExpiresAt: exp}); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := run(t, "--credentials", path, "whoami")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	for _, want := range []string{"user:   user-1", "email:  a@example.com", "family: family-1", "server: http://127.0.0.1:1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestWhoAmILoggedOut(t *testing.T) {
	code, _, errOut := run(t, "--credentials", filepath.Join(t.TempDir(), "none.json"), "whoami")
	if code != 1 || !strings.Contains(errOut, "fm login") {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
}

func TestLogoutLoggedOut(t *testing.T) {
	code, out, _ := run(t, "--credentials", filepath.Join(t.TempDir(), "none.json"), "logout")
	if code != 0 || !strings.Contains(out, "Not signed in") {
		t.Fatalf("exit %d, stdout %q", code, out)
	}
}

func TestUnknownCommand(t *testing.T) {
	code, _, errOut := run(t, "--credentials", filepath.Join(t.TempDir(), "none.json"), "frobnicate")
	if code != 2 || !strings.Contains(errOut, "unknown command") {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
}

func TestFlagsOverrideEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.json")
	exp := time.Now().Add(10 * time.Minute)
	enc := base64.RawURLEncoding
	tok := enc.EncodeToString([]byte(`{}`)) + "." + enc.EncodeToString([]byte(`{"sub":"u"}`)) + ".sig"
	if err := (&credentials.Store{Path: path}).Save(credentials.Credentials{AccessToken: tok, RefreshToken: "r", ExpiresAt: exp}); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := run(t, "--credentials", path, "--auth-url", "https://auth.example.test/", "whoami")
	if code != 0 || !strings.Contains(out, "server: https://auth.example.test\n") {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errOut)
	}
}
