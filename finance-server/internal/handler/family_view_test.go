package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nnc/finance-tracker/server/internal/handler"
)

// mockFamilyViewDB implements handler.FamilyViewDB for testing.
type mockFamilyViewDB struct {
	expenses       []handler.FamilyExpense
	memberTotals   []handler.FamilyMemberTotal
	categoryTotals []handler.FamilyCategoryTotal
}

func (m *mockFamilyViewDB) GetFamilyExpenses(familyID string, limit, offset int) ([]handler.FamilyExpense, error) {
	if m.expenses == nil {
		return []handler.FamilyExpense{}, nil
	}
	return m.expenses, nil
}

func (m *mockFamilyViewDB) GetFamilyMemberTotals(familyID string, dateFrom, dateTo time.Time) ([]handler.FamilyMemberTotal, error) {
	if m.memberTotals == nil {
		return []handler.FamilyMemberTotal{}, nil
	}
	return m.memberTotals, nil
}

func (m *mockFamilyViewDB) GetFamilyCategoryTotals(familyID string, dateFrom, dateTo time.Time) ([]handler.FamilyCategoryTotal, error) {
	if m.categoryTotals == nil {
		return []handler.FamilyCategoryTotal{}, nil
	}
	return m.categoryTotals, nil
}

func setupFamilyViewRouter(familyDB handler.FamilyDB, viewDB handler.FamilyViewDB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewFamilyViewHandler(familyDB, viewDB)

	families := r.Group("/api/v1/families", func(c *gin.Context) {
		c.Set("user_id", c.GetHeader("X-User-ID"))
		c.Next()
	})
	{
		families.GET("/me/expenses", h.FamilyFeed)
		families.GET("/me/summary", h.FamilySummary)
	}
	return r
}

func TestFamilyFeed_Success(t *testing.T) {
	fdb := newMockFamilyDB()
	fdb.families["family-1"] = &handler.MockFamily{
		ID:          "family-1",
		Name:        "Smith Family",
		AdminUserID: "user-1",
	}
	fdb.userFamily["user-1"] = "family-1"

	viewDB := &mockFamilyViewDB{
		expenses: []handler.FamilyExpense{
			{
				ID:            "exp-1",
				UserID:        "user-1",
				UserEmail:     "user1@test.com",
				CategoryID:    "cat-1",
				CategoryName:  "Food",
				CategoryColor: "#FF7043",
				CategoryIcon:  "restaurant",
				AmountCents:   2500,
				Note:          "Lunch",
				ExpenseDate:   time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
				CreatedAt:     time.Now(),
			},
			{
				ID:            "exp-2",
				UserID:        "user-2",
				UserEmail:     "user2@test.com",
				CategoryID:    "cat-2",
				CategoryName:  "Transport",
				CategoryColor: "#42A5F5",
				CategoryIcon:  "directions_car",
				AmountCents:   1500,
				Note:          "Bus",
				ExpenseDate:   time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC),
				CreatedAt:     time.Now(),
			},
		},
	}

	r := setupFamilyViewRouter(fdb, viewDB)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/families/me/expenses", nil)
	req.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(resp) != 2 {
		t.Fatalf("expected 2 expenses, got %d", len(resp))
	}

	first := resp[0]
	if first["user_email"] != "user1@test.com" {
		t.Fatalf("expected user_email user1@test.com, got %v", first["user_email"])
	}
	if first["category_name"] != "Food" {
		t.Fatalf("expected category_name Food, got %v", first["category_name"])
	}
	if first["amount_cents"] != float64(2500) {
		t.Fatalf("expected amount_cents 2500, got %v", first["amount_cents"])
	}
}

func TestFamilyFeedNoFamily(t *testing.T) {
	fdb := newMockFamilyDB()
	viewDB := &mockFamilyViewDB{}

	r := setupFamilyViewRouter(fdb, viewDB)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/families/me/expenses", nil)
	req.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "no family" {
		t.Fatalf("expected error 'no family', got %v", resp["error"])
	}
}

func TestFamilyFeed_Empty(t *testing.T) {
	fdb := newMockFamilyDB()
	fdb.families["family-1"] = &handler.MockFamily{
		ID:          "family-1",
		Name:        "Smith Family",
		AdminUserID: "user-1",
	}
	fdb.userFamily["user-1"] = "family-1"

	viewDB := &mockFamilyViewDB{}

	r := setupFamilyViewRouter(fdb, viewDB)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/families/me/expenses", nil)
	req.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response as array: %v, body: %s", err, w.Body.String())
	}

	if len(resp) != 0 {
		t.Fatalf("expected empty array, got %d items", len(resp))
	}
}

func TestFamilySummary_Success(t *testing.T) {
	fdb := newMockFamilyDB()
	fdb.families["family-1"] = &handler.MockFamily{
		ID:          "family-1",
		Name:        "Smith Family",
		AdminUserID: "user-1",
	}
	fdb.userFamily["user-1"] = "family-1"

	viewDB := &mockFamilyViewDB{
		memberTotals: []handler.FamilyMemberTotal{
			{UserID: "user-1", UserEmail: "user1@test.com", TotalCents: 45000, Count: 12},
			{UserID: "user-2", UserEmail: "user2@test.com", TotalCents: 30000, Count: 8},
		},
		categoryTotals: []handler.FamilyCategoryTotal{
			{CategoryID: "cat-1", CategoryName: "Food", CategoryColor: "#FF7043", CategoryIcon: "restaurant", TotalCents: 50000, Count: 15},
			{CategoryID: "cat-2", CategoryName: "Transport", CategoryColor: "#42A5F5", CategoryIcon: "directions_car", TotalCents: 25000, Count: 5},
		},
	}

	r := setupFamilyViewRouter(fdb, viewDB)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/families/me/summary?month=2026-03", nil)
	req.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["total_cents"] != float64(75000) {
		t.Fatalf("expected total_cents 75000, got %v", resp["total_cents"])
	}

	byPerson, ok := resp["by_person"].([]any)
	if !ok {
		t.Fatal("by_person should be an array")
	}
	if len(byPerson) != 2 {
		t.Fatalf("expected 2 members, got %d", len(byPerson))
	}

	byCategory, ok := resp["by_category"].([]any)
	if !ok {
		t.Fatal("by_category should be an array")
	}
	if len(byCategory) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(byCategory))
	}
}

func TestFamilySummary_MissingMonth(t *testing.T) {
	fdb := newMockFamilyDB()
	fdb.families["family-1"] = &handler.MockFamily{
		ID:          "family-1",
		Name:        "Smith Family",
		AdminUserID: "user-1",
	}
	fdb.userFamily["user-1"] = "family-1"

	viewDB := &mockFamilyViewDB{}

	r := setupFamilyViewRouter(fdb, viewDB)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/families/me/summary", nil)
	req.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	errMsg, _ := resp["error"].(string)
	if errMsg == "" {
		t.Fatal("expected error message containing 'month'")
	}
}

func TestFamilySummary_NoFamily(t *testing.T) {
	fdb := newMockFamilyDB()
	viewDB := &mockFamilyViewDB{}

	r := setupFamilyViewRouter(fdb, viewDB)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/families/me/summary?month=2026-03", nil)
	req.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFamilySummary_Empty(t *testing.T) {
	fdb := newMockFamilyDB()
	fdb.families["family-1"] = &handler.MockFamily{
		ID:          "family-1",
		Name:        "Smith Family",
		AdminUserID: "user-1",
	}
	fdb.userFamily["user-1"] = "family-1"

	viewDB := &mockFamilyViewDB{}

	r := setupFamilyViewRouter(fdb, viewDB)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/families/me/summary?month=2026-03", nil)
	req.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["total_cents"] != float64(0) {
		t.Fatalf("expected total_cents 0, got %v", resp["total_cents"])
	}

	byPerson, ok := resp["by_person"].([]any)
	if !ok {
		t.Fatal("by_person should be an array (not null)")
	}
	if len(byPerson) != 0 {
		t.Fatalf("expected empty by_person, got %d", len(byPerson))
	}

	byCategory, ok := resp["by_category"].([]any)
	if !ok {
		t.Fatal("by_category should be an array (not null)")
	}
	if len(byCategory) != 0 {
		t.Fatalf("expected empty by_category, got %d", len(byCategory))
	}
}
