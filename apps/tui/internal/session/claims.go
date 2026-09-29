package session

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrMalformedToken = errors.New("session: malformed access token")

type Claims struct {
	UserID    string
	FamilyID  string
	Email     string
	Locale    string
	ExpiresAt time.Time
}

func ParseClaims(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrMalformedToken
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return Claims{}, fmt.Errorf("%w: payload: %w", ErrMalformedToken, err)
	}
	var p struct {
		Sub      string `json:"sub"`
		FamilyID string `json:"family_id"`
		Email    string `json:"email"`
		Locale   string `json:"locale"`
		Exp      int64  `json:"exp"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return Claims{}, fmt.Errorf("%w: claims: %w", ErrMalformedToken, err)
	}
	if p.Sub == "" {
		return Claims{}, fmt.Errorf("%w: no subject", ErrMalformedToken)
	}
	c := Claims{UserID: p.Sub, FamilyID: p.FamilyID, Email: p.Email, Locale: p.Locale}
	if p.Exp > 0 {
		c.ExpiresAt = time.Unix(p.Exp, 0)
	}
	return c, nil
}
