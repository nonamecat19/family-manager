package service

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testJWTSecret = "test-secret-key-for-unit-tests-32b"

func TestHashPassword(t *testing.T) {
	svc := NewAuthService(testJWTSecret)

	hash, err := svc.HashPassword("mypassword")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "mypassword" {
		t.Fatal("Hash should differ from input")
	}
	if hash == "" {
		t.Fatal("Hash should not be empty")
	}
}

func TestCheckPassword_Correct(t *testing.T) {
	svc := NewAuthService(testJWTSecret)

	hash, err := svc.HashPassword("mypassword")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	err = svc.CheckPassword("mypassword", hash)
	if err != nil {
		t.Fatalf("CheckPassword should return nil for correct password, got: %v", err)
	}
}

func TestCheckPassword_Wrong(t *testing.T) {
	svc := NewAuthService(testJWTSecret)

	hash, err := svc.HashPassword("mypassword")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	err = svc.CheckPassword("wrong", hash)
	if err == nil {
		t.Fatal("CheckPassword should return error for wrong password")
	}
}

func TestGenerateTokenPair(t *testing.T) {
	svc := NewAuthService(testJWTSecret)

	pair, err := svc.GenerateTokenPair("user-uuid-123")
	if err != nil {
		t.Fatalf("GenerateTokenPair returned error: %v", err)
	}
	if pair.AccessToken == "" {
		t.Fatal("AccessToken should not be empty")
	}
	if pair.RefreshToken == "" {
		t.Fatal("RefreshToken should not be empty")
	}
}

func TestAccessTokenClaims(t *testing.T) {
	svc := NewAuthService(testJWTSecret)

	pair, err := svc.GenerateTokenPair("user-uuid-123")
	if err != nil {
		t.Fatalf("GenerateTokenPair returned error: %v", err)
	}

	// Parse and validate access token
	claims, err := svc.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken returned error: %v", err)
	}

	if claims.Subject != "user-uuid-123" {
		t.Fatalf("Expected sub=user-uuid-123, got %s", claims.Subject)
	}

	// Expiry should be ~15 minutes from now
	expiry := claims.ExpiresAt.Time
	expectedExpiry := time.Now().Add(15 * time.Minute)
	diff := expiry.Sub(expectedExpiry)
	if diff < -1*time.Minute || diff > 1*time.Minute {
		t.Fatalf("Access token expiry should be ~15min from now, got diff: %v", diff)
	}
}

func TestRefreshTokenClaims(t *testing.T) {
	svc := NewAuthService(testJWTSecret)

	pair, err := svc.GenerateTokenPair("user-uuid-123")
	if err != nil {
		t.Fatalf("GenerateTokenPair returned error: %v", err)
	}

	// Parse refresh token to check claims
	token, err := jwt.ParseWithClaims(pair.RefreshToken, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		return []byte(testJWTSecret), nil
	})
	if err != nil {
		t.Fatalf("Failed to parse refresh token: %v", err)
	}

	claims := token.Claims.(*jwt.RegisteredClaims)

	if claims.Subject != "user-uuid-123" {
		t.Fatalf("Expected sub=user-uuid-123, got %s", claims.Subject)
	}

	// JTI should be set
	if claims.ID == "" {
		t.Fatal("Refresh token should have jti set")
	}

	// Expiry should be ~30 days from now
	expiry := claims.ExpiresAt.Time
	expectedExpiry := time.Now().Add(30 * 24 * time.Hour)
	diff := expiry.Sub(expectedExpiry)
	if diff < -1*time.Minute || diff > 1*time.Minute {
		t.Fatalf("Refresh token expiry should be ~30 days from now, got diff: %v", diff)
	}
}

func TestHashRefreshToken(t *testing.T) {
	svc := NewAuthService(testJWTSecret)

	hash1 := svc.HashRefreshToken("some-token-string")
	hash2 := svc.HashRefreshToken("some-token-string")

	if hash1 == "" {
		t.Fatal("Hash should not be empty")
	}
	if hash1 != hash2 {
		t.Fatal("HashRefreshToken should return consistent results")
	}
	if hash1 == "some-token-string" {
		t.Fatal("Hash should differ from input")
	}
}

func TestValidatePassword(t *testing.T) {
	svc := NewAuthService(testJWTSecret)

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid 8 chars", "12345678", false},
		{"valid long", "a-very-good-password-indeed", false},
		{"too short", "1234567", true},
		{"empty", "", true},
		{"exactly 72", string(make([]byte, 72)), false},
		{"too long 73", string(make([]byte, 73)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
		})
	}
}
