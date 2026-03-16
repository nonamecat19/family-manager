package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nnc/finance-tracker/server/internal/handler"
)

// mockFamilyDB implements handler.FamilyDB for testing.
type mockFamilyDB struct {
	families    map[string]*handler.MockFamily
	members     map[string][]handler.MockFamilyMember // familyID -> members
	memberCount map[string]int64
	invitations map[string]*handler.MockInvitation // tokenHash -> invitation
	pending     map[string][]handler.MockPendingInvitation
	// Track user -> familyID mapping
	userFamily map[string]string
}

func newMockFamilyDB() *mockFamilyDB {
	return &mockFamilyDB{
		families:    make(map[string]*handler.MockFamily),
		members:     make(map[string][]handler.MockFamilyMember),
		memberCount: make(map[string]int64),
		invitations: make(map[string]*handler.MockInvitation),
		pending:     make(map[string][]handler.MockPendingInvitation),
		userFamily:  make(map[string]string),
	}
}

func (m *mockFamilyDB) CreateFamily(userID, name string) (handler.MockFamily, error) {
	f := handler.MockFamily{
		ID:          "family-1",
		Name:        name,
		AdminUserID: userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.families[f.ID] = &f
	return f, nil
}

func (m *mockFamilyDB) GetFamilyByUserID(userID string) (handler.MockFamily, error) {
	fid, ok := m.userFamily[userID]
	if !ok {
		return handler.MockFamily{}, handler.ErrFamilyNotFound
	}
	f, ok := m.families[fid]
	if !ok {
		return handler.MockFamily{}, handler.ErrFamilyNotFound
	}
	return *f, nil
}

func (m *mockFamilyDB) GetFamilyMembers(familyID string) ([]handler.MockFamilyMember, error) {
	return m.members[familyID], nil
}

func (m *mockFamilyDB) AddFamilyMember(familyID, userID, role string) error {
	m.members[familyID] = append(m.members[familyID], handler.MockFamilyMember{
		ID:       "member-" + userID,
		FamilyID: familyID,
		UserID:   userID,
		Email:    userID + "@test.com",
		Role:     role,
		JoinedAt: time.Now(),
	})
	m.userFamily[userID] = familyID
	m.memberCount[familyID]++
	return nil
}

func (m *mockFamilyDB) RemoveFamilyMember(familyID, userID string) (int64, error) {
	members := m.members[familyID]
	for i, mem := range members {
		if mem.UserID == userID && mem.Role != "admin" {
			m.members[familyID] = append(members[:i], members[i+1:]...)
			delete(m.userFamily, userID)
			m.memberCount[familyID]--
			return 1, nil
		}
	}
	return 0, nil
}

func (m *mockFamilyDB) DeleteFamily(familyID, adminUserID string) (int64, error) {
	f, ok := m.families[familyID]
	if !ok || f.AdminUserID != adminUserID {
		return 0, nil
	}
	// Clean up members
	for _, mem := range m.members[familyID] {
		delete(m.userFamily, mem.UserID)
	}
	delete(m.families, familyID)
	delete(m.members, familyID)
	delete(m.memberCount, familyID)
	return 1, nil
}

func (m *mockFamilyDB) GetFamilyMemberCount(familyID string) (int64, error) {
	return m.memberCount[familyID], nil
}

func (m *mockFamilyDB) CreateInvitation(familyID, inviterUserID, tokenHash string, expiresAt time.Time) (handler.MockInvitation, error) {
	inv := handler.MockInvitation{
		ID:            "inv-1",
		FamilyID:      familyID,
		InviterUserID: inviterUserID,
		TokenHash:     tokenHash,
		Status:        "pending",
		ExpiresAt:     expiresAt,
		CreatedAt:     time.Now(),
	}
	m.invitations[tokenHash] = &inv
	return inv, nil
}

func (m *mockFamilyDB) GetInvitationByTokenHash(tokenHash string) (handler.MockInvitation, error) {
	inv, ok := m.invitations[tokenHash]
	if !ok {
		return handler.MockInvitation{}, handler.ErrInvitationNotFound
	}
	return *inv, nil
}

func (m *mockFamilyDB) AcceptInvitation(invitationID string) (int64, error) {
	return 1, nil
}

func (m *mockFamilyDB) RevokeInvitation(invitationID, familyID string) (int64, error) {
	// Check if any invitation matches
	for hash, inv := range m.invitations {
		if inv.ID == invitationID && inv.FamilyID == familyID {
			delete(m.invitations, hash)
			return 1, nil
		}
	}
	return 0, nil
}

func (m *mockFamilyDB) GetPendingInvitations(familyID string) ([]handler.MockPendingInvitation, error) {
	return m.pending[familyID], nil
}

func setupFamilyRouter(db handler.FamilyDB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewFamilyHandler(db)

	// Simulate auth middleware by setting user_id
	families := r.Group("/api/v1/families", func(c *gin.Context) {
		c.Set("user_id", c.GetHeader("X-User-ID"))
		c.Next()
	})
	{
		families.POST("", h.CreateFamily)
		families.GET("/me", h.GetMyFamily)
		families.DELETE("/me", h.DeleteMyFamily)
		families.DELETE("/me/members/:userId", h.RemoveMember)
		families.POST("/me/leave", h.LeaveFamily)
		families.POST("/me/invitations", h.CreateInvitation)
		families.DELETE("/me/invitations/:id", h.RevokeInvitation)
	}
	invitations := r.Group("/api/v1/invitations", func(c *gin.Context) {
		c.Set("user_id", c.GetHeader("X-User-ID"))
		c.Next()
	})
	{
		invitations.GET("/:token", h.GetInvitationInfo)
		invitations.POST("/accept", h.AcceptInvitation)
	}
	return r
}

func TestCreateFamily(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := newMockFamilyDB()
		r := setupFamilyRouter(db)

		body, _ := json.Marshal(map[string]string{"name": "Smith Family"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["name"] != "Smith Family" {
			t.Fatalf("expected name 'Smith Family', got %v", resp["name"])
		}
	})

	t.Run("already in family", func(t *testing.T) {
		db := newMockFamilyDB()
		r := setupFamilyRouter(db)

		// Create first family
		body, _ := json.Marshal(map[string]string{"name": "Family 1"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Try to create second family
		body, _ = json.Marshal(map[string]string{"name": "Family 2"})
		req = httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-1")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestGetMyFamily(t *testing.T) {
	t.Run("success with members", func(t *testing.T) {
		db := newMockFamilyDB()
		r := setupFamilyRouter(db)

		// Create family first
		body, _ := json.Marshal(map[string]string{"name": "Smith Family"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Get family
		req = httptest.NewRequest(http.MethodGet, "/api/v1/families/me", nil)
		req.Header.Set("X-User-ID", "user-1")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp["family"] == nil {
			t.Fatal("response should include family")
		}
		if resp["members"] == nil {
			t.Fatal("response should include members")
		}
	})

	t.Run("no family", func(t *testing.T) {
		db := newMockFamilyDB()
		r := setupFamilyRouter(db)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/families/me", nil)
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestDeleteMyFamily(t *testing.T) {
	t.Run("admin success", func(t *testing.T) {
		db := newMockFamilyDB()
		r := setupFamilyRouter(db)

		// Create family
		body, _ := json.Marshal(map[string]string{"name": "Smith Family"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Delete family
		req = httptest.NewRequest(http.MethodDelete, "/api/v1/families/me", nil)
		req.Header.Set("X-User-ID", "user-1")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("non-admin rejected", func(t *testing.T) {
		db := newMockFamilyDB()
		r := setupFamilyRouter(db)

		// Create family as user-1
		body, _ := json.Marshal(map[string]string{"name": "Smith Family"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Add user-2 as member
		db.AddFamilyMember("family-1", "user-2", "member")

		// Try to delete as user-2
		req = httptest.NewRequest(http.MethodDelete, "/api/v1/families/me", nil)
		req.Header.Set("X-User-ID", "user-2")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestCreateInvitation(t *testing.T) {
	t.Run("success returns raw token", func(t *testing.T) {
		db := newMockFamilyDB()
		r := setupFamilyRouter(db)

		// Create family
		body, _ := json.Marshal(map[string]string{"name": "Smith Family"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Create invitation
		req = httptest.NewRequest(http.MethodPost, "/api/v1/families/me/invitations", nil)
		req.Header.Set("X-User-ID", "user-1")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		token, ok := resp["token"].(string)
		if !ok || token == "" {
			t.Fatal("response should include raw token")
		}
		// Token should be 64 hex chars (32 bytes)
		if len(token) != 64 {
			t.Fatalf("expected token length 64, got %d", len(token))
		}
	})

	t.Run("family full at 10 members", func(t *testing.T) {
		db := newMockFamilyDB()
		r := setupFamilyRouter(db)

		// Create family
		body, _ := json.Marshal(map[string]string{"name": "Smith Family"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Set member count to 10
		db.memberCount["family-1"] = 10

		// Try to create invitation
		req = httptest.NewRequest(http.MethodPost, "/api/v1/families/me/invitations", nil)
		req.Header.Set("X-User-ID", "user-1")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestAcceptInvitation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := newMockFamilyDB()
		r := setupFamilyRouter(db)

		// Create family as user-1
		body, _ := json.Marshal(map[string]string{"name": "Smith Family"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Create invitation
		req = httptest.NewRequest(http.MethodPost, "/api/v1/families/me/invitations", nil)
		req.Header.Set("X-User-ID", "user-1")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var invResp map[string]any
		json.Unmarshal(w.Body.Bytes(), &invResp)
		token := invResp["token"].(string)

		// Accept invitation as user-2
		body, _ = json.Marshal(map[string]string{"token": token})
		req = httptest.NewRequest(http.MethodPost, "/api/v1/invitations/accept", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-2")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("expired token", func(t *testing.T) {
		db := newMockFamilyDB()
		r := setupFamilyRouter(db)

		// Try to accept with unknown token
		body, _ := json.Marshal(map[string]string{"token": "nonexistent-token"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/invitations/accept", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-2")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("user already in family", func(t *testing.T) {
		db := newMockFamilyDB()
		r := setupFamilyRouter(db)

		// Create family as user-1
		body, _ := json.Marshal(map[string]string{"name": "Smith Family"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Create invitation
		req = httptest.NewRequest(http.MethodPost, "/api/v1/families/me/invitations", nil)
		req.Header.Set("X-User-ID", "user-1")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var invResp map[string]any
		json.Unmarshal(w.Body.Bytes(), &invResp)
		token := invResp["token"].(string)

		// Try to accept as user-1 (already in family)
		body, _ = json.Marshal(map[string]string{"token": token})
		req = httptest.NewRequest(http.MethodPost, "/api/v1/invitations/accept", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-1")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestLeaveFamily(t *testing.T) {
	t.Run("member success", func(t *testing.T) {
		db := newMockFamilyDB()
		r := setupFamilyRouter(db)

		// Create family as user-1
		body, _ := json.Marshal(map[string]string{"name": "Smith Family"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Add user-2 as member
		db.AddFamilyMember("family-1", "user-2", "member")

		// user-2 leaves
		req = httptest.NewRequest(http.MethodPost, "/api/v1/families/me/leave", nil)
		req.Header.Set("X-User-ID", "user-2")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("admin rejected", func(t *testing.T) {
		db := newMockFamilyDB()
		r := setupFamilyRouter(db)

		// Create family as user-1
		body, _ := json.Marshal(map[string]string{"name": "Smith Family"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Admin tries to leave
		req = httptest.NewRequest(http.MethodPost, "/api/v1/families/me/leave", nil)
		req.Header.Set("X-User-ID", "user-1")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestRevokeInvitation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := newMockFamilyDB()
		r := setupFamilyRouter(db)

		// Create family
		body, _ := json.Marshal(map[string]string{"name": "Smith Family"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Create invitation
		req = httptest.NewRequest(http.MethodPost, "/api/v1/families/me/invitations", nil)
		req.Header.Set("X-User-ID", "user-1")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Revoke invitation
		req = httptest.NewRequest(http.MethodDelete, "/api/v1/families/me/invitations/inv-1", nil)
		req.Header.Set("X-User-ID", "user-1")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found", func(t *testing.T) {
		db := newMockFamilyDB()
		r := setupFamilyRouter(db)

		// Create family
		body, _ := json.Marshal(map[string]string{"name": "Smith Family"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Revoke non-existent invitation
		req = httptest.NewRequest(http.MethodDelete, "/api/v1/families/me/invitations/nonexistent", nil)
		req.Header.Set("X-User-ID", "user-1")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestRemoveMember(t *testing.T) {
	t.Run("admin removes member", func(t *testing.T) {
		db := newMockFamilyDB()
		r := setupFamilyRouter(db)

		// Create family as user-1
		body, _ := json.Marshal(map[string]string{"name": "Smith Family"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Add user-2 as member
		db.AddFamilyMember("family-1", "user-2", "member")

		// Admin removes user-2
		req = httptest.NewRequest(http.MethodDelete, "/api/v1/families/me/members/user-2", nil)
		req.Header.Set("X-User-ID", "user-1")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("non-admin rejected", func(t *testing.T) {
		db := newMockFamilyDB()
		r := setupFamilyRouter(db)

		// Create family as user-1
		body, _ := json.Marshal(map[string]string{"name": "Smith Family"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/families", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-ID", "user-1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Add user-2 as member
		db.AddFamilyMember("family-1", "user-2", "member")
		// Add user-3 as member
		db.AddFamilyMember("family-1", "user-3", "member")

		// user-2 tries to remove user-3
		req = httptest.NewRequest(http.MethodDelete, "/api/v1/families/me/members/user-3", nil)
		req.Header.Set("X-User-ID", "user-2")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}
