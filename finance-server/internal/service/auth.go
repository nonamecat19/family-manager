package service

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// TokenPair holds an access and refresh JWT token.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// AuthService handles password hashing and JWT token operations.
type AuthService struct {
	jwtSecret []byte
}

// NewAuthService creates an AuthService with the given JWT signing secret.
func NewAuthService(jwtSecret string) *AuthService {
	return &AuthService{jwtSecret: []byte(jwtSecret)}
}

// JWTSecret returns the raw JWT secret bytes (for middleware initialization).
func (s *AuthService) JWTSecret() []byte {
	return s.jwtSecret
}

// HashPassword returns a bcrypt hash of the given password (cost 12).
func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compares a plaintext password against a bcrypt hash.
// Returns nil on match, error otherwise.
func (s *AuthService) CheckPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// ValidatePassword checks that a password meets length requirements.
func (s *AuthService) ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if len(password) > 72 {
		return errors.New("password must be at most 72 characters")
	}
	return nil
}

// GenerateTokenPair creates a signed access token (15 min) and refresh token (30 days).
func (s *AuthService) GenerateTokenPair(userID string) (*TokenPair, error) {
	now := time.Now()

	// Access token: 15 minutes
	accessClaims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(now),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	// Refresh token: 30 days with jti for revocation tracking
	refreshClaims := jwt.RegisteredClaims{
		ID:        uuid.New().String(),
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(now.Add(30 * 24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(now),
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("signing refresh token: %w", err)
	}

	return &TokenPair{AccessToken: accessStr, RefreshToken: refreshStr}, nil
}

// ValidateAccessToken parses and validates a JWT access token string.
// Returns the registered claims if valid.
func (s *AuthService) ValidateAccessToken(tokenStr string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// HashRefreshToken returns the SHA-256 hex digest of a refresh token.
// Used for secure DB storage (never store raw tokens).
func (s *AuthService) HashRefreshToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}
