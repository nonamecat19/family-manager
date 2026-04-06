package handler

import (
	"errors"
	"net/http"
	"net/mail"

	"github.com/gin-gonic/gin"
	"github.com/nnc/finance-tracker/server/internal/service"
)

// Sentinel errors used by AuthDB implementations.
var (
	ErrDuplicateEmail = errors.New("duplicate email")
	ErrUserNotFound   = errors.New("user not found")
	ErrTokenNotFound  = errors.New("token not found")
)

// MockUser is the user representation used by the AuthDB interface.
type MockUser struct {
	ID           string
	Email        string
	PasswordHash string
}

// MockRefreshToken is the refresh token representation used by the AuthDB interface.
type MockRefreshToken struct {
	UserID    string
	TokenHash string
}

// AuthDB abstracts database operations for authentication.
// This allows testing with mock implementations.
type AuthDB interface {
	CreateUser(email, passwordHash string) (MockUser, error)
	GetUserByEmail(email string) (MockUser, error)
	StoreRefreshToken(userID, tokenHash string, expiresInDays int) error
	GetRefreshTokenByHash(tokenHash string) (MockRefreshToken, error)
	RevokeRefreshToken(tokenHash string) error
}

// AuthHandler handles authentication HTTP requests.
type AuthHandler struct {
	db      AuthDB
	authSvc *service.AuthService
}

// NewAuthHandler creates an AuthHandler with the given database and auth service.
func NewAuthHandler(db AuthDB, authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{db: db, authSvc: authSvc}
}

type signupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Signup creates a new user account and returns a token pair.
func (h *AuthHandler) Signup(c *gin.Context) {
	var req signupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate email
	if _, err := mail.ParseAddress(req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errors": gin.H{"email": "Invalid email address"}})
		return
	}

	// Validate password
	if err := h.authSvc.ValidatePassword(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errors": gin.H{"password": err.Error()}})
		return
	}

	// Hash password
	hash, err := h.authSvc.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Create user
	user, err := h.db.CreateUser(req.Email, hash)
	if err != nil {
		if errors.Is(err, ErrDuplicateEmail) {
			c.JSON(http.StatusConflict, gin.H{"error": "An account with this email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Generate tokens
	pair, err := h.authSvc.GenerateTokenPair(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Store refresh token hash
	tokenHash := h.authSvc.HashRefreshToken(pair.RefreshToken)
	if err := h.db.StoreRefreshToken(user.ID, tokenHash, 30); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
		},
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login validates credentials and returns a token pair.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Find user
	user, err := h.db.GetUserByEmail(req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No account with this email"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Check password
	if err := h.authSvc.CheckPassword(req.Password, user.PasswordHash); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Wrong password"})
		return
	}

	// Generate tokens
	pair, err := h.authSvc.GenerateTokenPair(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Store refresh token hash
	tokenHash := h.authSvc.HashRefreshToken(pair.RefreshToken)
	if err := h.db.StoreRefreshToken(user.ID, tokenHash, 30); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
		},
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh validates a refresh token, revokes it, and returns a new token pair.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate the refresh JWT
	claims, err := h.authSvc.ValidateAccessToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Look up token hash in DB
	tokenHash := h.authSvc.HashRefreshToken(req.RefreshToken)
	_, err = h.db.GetRefreshTokenByHash(tokenHash)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Revoke old token
	if err := h.db.RevokeRefreshToken(tokenHash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Generate new pair
	pair, err := h.authSvc.GenerateTokenPair(claims.Subject)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Store new refresh token hash
	newTokenHash := h.authSvc.HashRefreshToken(pair.RefreshToken)
	if err := h.db.StoreRefreshToken(claims.Subject, newTokenHash, 30); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
	})
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Logout revokes a refresh token.
func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	tokenHash := h.authSvc.HashRefreshToken(req.RefreshToken)
	_ = h.db.RevokeRefreshToken(tokenHash)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}
