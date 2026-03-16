package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Sentinel errors for family operations.
var (
	ErrFamilyNotFound     = errors.New("family not found")
	ErrInvitationNotFound = errors.New("invitation not found")
	ErrAlreadyInFamily    = errors.New("user already in a family")
	ErrFamilyFull         = errors.New("family is full")
)

// MockFamily is the family representation used by the FamilyDB interface.
type MockFamily struct {
	ID          string
	Name        string
	AdminUserID string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// MockFamilyMember is the member representation used by the FamilyDB interface.
type MockFamilyMember struct {
	ID       string
	FamilyID string
	UserID   string
	Email    string
	Role     string
	JoinedAt time.Time
}

// MockInvitation is the invitation representation used by the FamilyDB interface.
type MockInvitation struct {
	ID            string
	FamilyID      string
	InviterUserID string
	TokenHash     string
	Status        string
	FamilyName    string
	ExpiresAt     time.Time
	CreatedAt     time.Time
}

// MockPendingInvitation is the pending invitation representation.
type MockPendingInvitation struct {
	ID            string
	FamilyID      string
	InviterUserID string
	Status        string
	ExpiresAt     time.Time
	CreatedAt     time.Time
}

// FamilyDB abstracts database operations for families.
// This allows testing with mock implementations.
type FamilyDB interface {
	CreateFamily(userID, name string) (MockFamily, error)
	GetFamilyByUserID(userID string) (MockFamily, error)
	GetFamilyMembers(familyID string) ([]MockFamilyMember, error)
	AddFamilyMember(familyID, userID, role string) error
	RemoveFamilyMember(familyID, userID string) (int64, error)
	DeleteFamily(familyID, adminUserID string) (int64, error)
	GetFamilyMemberCount(familyID string) (int64, error)
	CreateInvitation(familyID, inviterUserID, tokenHash string, expiresAt time.Time) (MockInvitation, error)
	GetInvitationByTokenHash(tokenHash string) (MockInvitation, error)
	AcceptInvitation(invitationID string) (int64, error)
	RevokeInvitation(invitationID, familyID string) (int64, error)
	GetPendingInvitations(familyID string) ([]MockPendingInvitation, error)
}

// FamilyHandler handles family HTTP requests.
type FamilyHandler struct {
	db FamilyDB
}

// NewFamilyHandler creates a FamilyHandler with the given database.
func NewFamilyHandler(db FamilyDB) *FamilyHandler {
	return &FamilyHandler{db: db}
}

type createFamilyRequest struct {
	Name string `json:"name"`
}

// CreateFamily handles POST /api/v1/families.
func (h *FamilyHandler) CreateFamily(c *gin.Context) {
	var req createFamilyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	userID := c.GetString("user_id")

	// Check user not already in a family
	_, err := h.db.GetFamilyByUserID(userID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "You are already in a family"})
		return
	}
	if !errors.Is(err, ErrFamilyNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Create family
	family, err := h.db.CreateFamily(userID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Add creator as admin member
	if err := h.db.AddFamilyMember(family.ID, userID, "admin"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":            family.ID,
		"name":          family.Name,
		"admin_user_id": family.AdminUserID,
	})
}

// GetMyFamily handles GET /api/v1/families/me.
func (h *FamilyHandler) GetMyFamily(c *gin.Context) {
	userID := c.GetString("user_id")

	family, err := h.db.GetFamilyByUserID(userID)
	if err != nil {
		if errors.Is(err, ErrFamilyNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no family"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	members, err := h.db.GetFamilyMembers(family.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	memberList := make([]gin.H, len(members))
	for i, m := range members {
		memberList[i] = gin.H{
			"id":        m.ID,
			"user_id":   m.UserID,
			"email":     m.Email,
			"role":      m.Role,
			"joined_at": m.JoinedAt,
		}
	}

	result := gin.H{
		"family": gin.H{
			"id":            family.ID,
			"name":          family.Name,
			"admin_user_id": family.AdminUserID,
		},
		"members": memberList,
	}

	// Include invitations for admin only
	if family.AdminUserID == userID {
		invitations, err := h.db.GetPendingInvitations(family.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
		invList := make([]gin.H, len(invitations))
		for i, inv := range invitations {
			invList[i] = gin.H{
				"id":         inv.ID,
				"status":     inv.Status,
				"expires_at": inv.ExpiresAt,
				"created_at": inv.CreatedAt,
			}
		}
		result["invitations"] = invList
	}

	c.JSON(http.StatusOK, result)
}

// DeleteMyFamily handles DELETE /api/v1/families/me.
func (h *FamilyHandler) DeleteMyFamily(c *gin.Context) {
	userID := c.GetString("user_id")

	family, err := h.db.GetFamilyByUserID(userID)
	if err != nil {
		if errors.Is(err, ErrFamilyNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no family"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if family.AdminUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admin can delete the family"})
		return
	}

	rows, err := h.db.DeleteFamily(family.ID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Family not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Family deleted"})
}

// RemoveMember handles DELETE /api/v1/families/me/members/:userId.
func (h *FamilyHandler) RemoveMember(c *gin.Context) {
	userID := c.GetString("user_id")
	targetUserID := c.Param("userId")

	family, err := h.db.GetFamilyByUserID(userID)
	if err != nil {
		if errors.Is(err, ErrFamilyNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no family"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if family.AdminUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admin can remove members"})
		return
	}

	rows, err := h.db.RemoveFamilyMember(family.ID, targetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Member not found or is admin"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed"})
}

// LeaveFamily handles POST /api/v1/families/me/leave.
func (h *FamilyHandler) LeaveFamily(c *gin.Context) {
	userID := c.GetString("user_id")

	family, err := h.db.GetFamilyByUserID(userID)
	if err != nil {
		if errors.Is(err, ErrFamilyNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no family"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if family.AdminUserID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Admin cannot leave. Delete the family instead."})
		return
	}

	_, err = h.db.RemoveFamilyMember(family.ID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Left family"})
}

// CreateInvitation handles POST /api/v1/families/me/invitations.
func (h *FamilyHandler) CreateInvitation(c *gin.Context) {
	userID := c.GetString("user_id")

	family, err := h.db.GetFamilyByUserID(userID)
	if err != nil {
		if errors.Is(err, ErrFamilyNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no family"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if family.AdminUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admin can create invitations"})
		return
	}

	// Check member count
	count, err := h.db.GetFamilyMemberCount(family.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	if count >= 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Family is full (max 10 members)"})
		return
	}

	// Generate token: 32 random bytes -> hex
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	rawToken := hex.EncodeToString(tokenBytes)

	// SHA-256 hash for storage
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	inv, err := h.db.CreateInvitation(family.ID, userID, tokenHash, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token":      rawToken,
		"expires_at": inv.ExpiresAt,
	})
}

// RevokeInvitation handles DELETE /api/v1/families/me/invitations/:id.
func (h *FamilyHandler) RevokeInvitation(c *gin.Context) {
	userID := c.GetString("user_id")
	invitationID := c.Param("id")

	family, err := h.db.GetFamilyByUserID(userID)
	if err != nil {
		if errors.Is(err, ErrFamilyNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no family"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if family.AdminUserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admin can revoke invitations"})
		return
	}

	rows, err := h.db.RevokeInvitation(invitationID, family.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invitation not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation revoked"})
}

// GetInvitationInfo handles GET /api/v1/invitations/:token.
func (h *FamilyHandler) GetInvitationInfo(c *gin.Context) {
	rawToken := c.Param("token")

	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	inv, err := h.db.GetInvitationByTokenHash(tokenHash)
	if err != nil {
		if errors.Is(err, ErrInvitationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Invitation not found or expired"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"family_name": inv.FamilyName,
		"expires_at":  inv.ExpiresAt,
	})
}

type acceptInvitationRequest struct {
	Token string `json:"token"`
}

// AcceptInvitation handles POST /api/v1/invitations/accept.
func (h *FamilyHandler) AcceptInvitation(c *gin.Context) {
	var req acceptInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	userID := c.GetString("user_id")

	// Check user not already in a family
	_, err := h.db.GetFamilyByUserID(userID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "You are already in a family"})
		return
	}
	if !errors.Is(err, ErrFamilyNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Hash token and look up
	hash := sha256.Sum256([]byte(req.Token))
	tokenHash := hex.EncodeToString(hash[:])

	inv, err := h.db.GetInvitationByTokenHash(tokenHash)
	if err != nil {
		if errors.Is(err, ErrInvitationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Invitation not found or expired"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Check member count
	count, err := h.db.GetFamilyMemberCount(inv.FamilyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	if count >= 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Family is full (max 10 members)"})
		return
	}

	// Add member
	if err := h.db.AddFamilyMember(inv.FamilyID, userID, "member"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Mark invitation accepted
	_, err = h.db.AcceptInvitation(inv.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"family_id":   inv.FamilyID,
		"family_name": inv.FamilyName,
		"message":     "Joined family",
	})
}
