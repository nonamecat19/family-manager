package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nnc/finance-tracker/server/internal/handler"
	"github.com/nnc/finance-tracker/server/internal/service"
)

const testJWTSecret = "test-secret-key-for-unit-tests-32b"

// mockDB implements the handler.AuthDB interface for testing without a real database.
type mockDB struct {
	users         map[string]*handler.MockUser
	refreshTokens map[string]*handler.MockRefreshToken
}

func newMockDB() *mockDB {
	return &mockDB{
		users:         make(map[string]*handler.MockUser),
		refreshTokens: make(map[string]*handler.MockRefreshToken),
	}
}

func (m *mockDB) CreateUser(email, passwordHash string) (handler.MockUser, error) {
	if _, exists := m.users[email]; exists {
		return handler.MockUser{}, handler.ErrDuplicateEmail
	}
	u := handler.MockUser{
		ID:           "test-user-id",
		Email:        email,
		PasswordHash: passwordHash,
	}
	m.users[email] = &u
	return u, nil
}

func (m *mockDB) GetUserByEmail(email string) (handler.MockUser, error) {
	u, exists := m.users[email]
	if !exists {
		return handler.MockUser{}, handler.ErrUserNotFound
	}
	return *u, nil
}

func (m *mockDB) StoreRefreshToken(userID, tokenHash string, expiresInDays int) error {
	m.refreshTokens[tokenHash] = &handler.MockRefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
	}
	return nil
}

func (m *mockDB) GetRefreshTokenByHash(tokenHash string) (handler.MockRefreshToken, error) {
	rt, exists := m.refreshTokens[tokenHash]
	if !exists {
		return handler.MockRefreshToken{}, handler.ErrTokenNotFound
	}
	return *rt, nil
}

func (m *mockDB) RevokeRefreshToken(tokenHash string) error {
	delete(m.refreshTokens, tokenHash)
	return nil
}

func setupRouter(db handler.AuthDB, authSvc *service.AuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewAuthHandler(db, authSvc)
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/signup", h.Signup)
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)
		auth.POST("/logout", h.Logout)
	}
	return r
}

func TestSignup_Success(t *testing.T) {
	db := newMockDB()
	authSvc := service.NewAuthService(testJWTSecret)
	r := setupRouter(db, authSvc)

	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["access_token"] == nil || resp["access_token"] == "" {
		t.Fatal("response should include access_token")
	}
	if resp["refresh_token"] == nil || resp["refresh_token"] == "" {
		t.Fatal("response should include refresh_token")
	}

	user, ok := resp["user"].(map[string]any)
	if !ok {
		t.Fatal("response should include user object")
	}
	if user["email"] != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %v", user["email"])
	}
}

func TestSignup_DuplicateEmail(t *testing.T) {
	db := newMockDB()
	authSvc := service.NewAuthService(testJWTSecret)
	r := setupRouter(db, authSvc)

	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})

	// First signup
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Second signup with same email
	body, _ = json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "An account with this email already exists" {
		t.Fatalf("expected specific error message, got: %v", resp["error"])
	}
}

func TestSignup_InvalidEmail(t *testing.T) {
	db := newMockDB()
	authSvc := service.NewAuthService(testJWTSecret)
	r := setupRouter(db, authSvc)

	body, _ := json.Marshal(map[string]string{
		"email":    "not-an-email",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSignup_ShortPassword(t *testing.T) {
	db := newMockDB()
	authSvc := service.NewAuthService(testJWTSecret)
	r := setupRouter(db, authSvc)

	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "short",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogin_Success(t *testing.T) {
	db := newMockDB()
	authSvc := service.NewAuthService(testJWTSecret)
	r := setupRouter(db, authSvc)

	// First signup
	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Login
	body, _ = json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["access_token"] == nil || resp["access_token"] == "" {
		t.Fatal("response should include access_token")
	}
	if resp["refresh_token"] == nil || resp["refresh_token"] == "" {
		t.Fatal("response should include refresh_token")
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	db := newMockDB()
	authSvc := service.NewAuthService(testJWTSecret)
	r := setupRouter(db, authSvc)

	body, _ := json.Marshal(map[string]string{
		"email":    "unknown@example.com",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "No account with this email" {
		t.Fatalf("expected specific error, got: %v", resp["error"])
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	db := newMockDB()
	authSvc := service.NewAuthService(testJWTSecret)
	r := setupRouter(db, authSvc)

	// Signup first
	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Login with wrong password
	body, _ = json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "wrongpassword",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "Wrong password" {
		t.Fatalf("expected specific error, got: %v", resp["error"])
	}
}
