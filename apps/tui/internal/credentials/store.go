package credentials

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

var ErrNotFound = errors.New("credentials: not logged in")

type Credentials struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type Store struct {
	Path string
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("credentials: locate config dir: %w", err)
	}
	return filepath.Join(dir, "family-manager", "credentials.json"), nil
}

func Default() (*Store, error) {
	p, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return &Store{Path: p}, nil
}

func (s *Store) Load() (Credentials, error) {
	raw, err := os.ReadFile(s.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return Credentials{}, ErrNotFound
	}
	if err != nil {
		return Credentials{}, fmt.Errorf("credentials: read %s: %w", s.Path, err)
	}
	var c Credentials
	if err := json.Unmarshal(raw, &c); err != nil {
		return Credentials{}, fmt.Errorf("credentials: parse %s: %w", s.Path, err)
	}
	if c.RefreshToken == "" {
		return Credentials{}, ErrNotFound
	}
	return c, nil
}

func (s *Store) Save(c Credentials) error {
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("credentials: create %s: %w", dir, err)
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("credentials: encode: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".credentials-*.json")
	if err != nil {
		return fmt.Errorf("credentials: create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("credentials: chmod temp file: %w", err)
	}
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return fmt.Errorf("credentials: write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("credentials: close temp file: %w", err)
	}
	if err := os.Rename(tmpName, s.Path); err != nil {
		return fmt.Errorf("credentials: replace %s: %w", s.Path, err)
	}
	return nil
}

func (s *Store) Delete() error {
	err := os.Remove(s.Path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("credentials: remove %s: %w", s.Path, err)
	}
	return nil
}
