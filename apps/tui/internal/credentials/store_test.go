package credentials

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	s := &Store{Path: filepath.Join(t.TempDir(), "family-manager", "credentials.json")}
	want := Credentials{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC),
	}
	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.AccessToken != want.AccessToken || got.RefreshToken != want.RefreshToken || !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Fatalf("Load = %+v, want %+v", got, want)
	}
}

func TestSaveIsOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions")
	}
	s := &Store{Path: filepath.Join(t.TempDir(), "fm", "credentials.json")}
	if err := s.Save(Credentials{AccessToken: "a", RefreshToken: "r"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	fi, err := os.Stat(s.Path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Fatalf("file mode = %o, want 600", perm)
	}
	di, err := os.Stat(filepath.Dir(s.Path))
	if err != nil {
		t.Fatalf("Stat dir: %v", err)
	}
	if perm := di.Mode().Perm(); perm != 0o700 {
		t.Fatalf("dir mode = %o, want 700", perm)
	}
}

func TestSaveOverwritesAndLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	s := &Store{Path: filepath.Join(dir, "credentials.json")}
	if err := s.Save(Credentials{AccessToken: "a1", RefreshToken: "r1"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Save(Credentials{AccessToken: "a2", RefreshToken: "r2"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.RefreshToken != "r2" {
		t.Fatalf("RefreshToken = %q, want r2", got.RefreshToken)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("dir has %d entries, want only credentials.json", len(entries))
	}
}

func TestLoadMissingIsNotFound(t *testing.T) {
	s := &Store{Path: filepath.Join(t.TempDir(), "credentials.json")}
	if _, err := s.Load(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load err = %v, want ErrNotFound", err)
	}
}

func TestLoadWithoutRefreshTokenIsNotFound(t *testing.T) {
	p := filepath.Join(t.TempDir(), "credentials.json")
	if err := os.WriteFile(p, []byte(`{"access_token":"a"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Store{Path: p}
	if _, err := s.Load(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load err = %v, want ErrNotFound", err)
	}
}

func TestLoadCorruptFails(t *testing.T) {
	p := filepath.Join(t.TempDir(), "credentials.json")
	if err := os.WriteFile(p, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Store{Path: p}
	_, err := s.Load()
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("Load err = %v, want a parse error", err)
	}
}

func TestDeleteIsIdempotent(t *testing.T) {
	s := &Store{Path: filepath.Join(t.TempDir(), "credentials.json")}
	if err := s.Save(Credentials{AccessToken: "a", RefreshToken: "r"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Delete(); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := s.Delete(); err != nil {
		t.Fatalf("second Delete: %v", err)
	}
	if _, err := s.Load(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load after Delete err = %v, want ErrNotFound", err)
	}
}
