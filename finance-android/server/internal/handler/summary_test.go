package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nnc/finance-tracker/server/internal/handler"
)

// mockSummaryDB implements handler.SummaryDB for testing.
type mockSummaryDB struct {
	categoryTotals []handler.CategoryTotal
	dailyTotals    []handler.DateTotal
	err            error
}

func (m *mockSummaryDB) GetCategoryTotals(userID string, dateFrom, dateTo time.Time) ([]handler.CategoryTotal, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.categoryTotals, nil
}

func (m *mockSummaryDB) GetDailyTotals(userID string, dateFrom, dateTo time.Time) ([]handler.DateTotal, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.dailyTotals, nil
}

func setupSummaryRouter(db handler.SummaryDB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewSummaryHandler(db)

	expenses := r.Group("/api/v1/expenses")
	expenses.Use(func(c *gin.Context) {
		c.Set("user_id", testUserID)
		c.Next()
	})
	{
		expenses.GET("/summary", h.Summary)
	}
	return r
}

func TestSummary_Success(t *testing.T) {
	db := &mockSummaryDB{
		categoryTotals: []handler.CategoryTotal{
			{
				CategoryID:    "550e8400-e29b-41d4-a716-446655440001",
				CategoryName:  "Food",
				CategoryColor: "#FF7043",
				CategoryIcon:  "restaurant",
				TotalCents:    45000,
				Count:         12,
			},
			{
				CategoryID:    "550e8400-e29b-41d4-a716-446655440002",
				CategoryName:  "Transport",
				CategoryColor: "#42A5F5",
				CategoryIcon:  "directions_car",
				TotalCents:    30000,
				Count:         8,
			},
		},
		dailyTotals: []handler.DateTotal{
			{Date: "2026-03-01", TotalCents: 5000},
			{Date: "2026-03-05", TotalCents: 15000},
			{Date: "2026-03-10", TotalCents: 55000},
		},
	}
	r := setupSummaryRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses/summary?month=2026-03", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["month"] != "2026-03" {
		t.Fatalf("expected month 2026-03, got %v", resp["month"])
	}

	// total_cents should be sum of category totals: 45000 + 30000 = 75000
	if resp["total_cents"] != float64(75000) {
		t.Fatalf("expected total_cents 75000, got %v", resp["total_cents"])
	}

	byCategory, ok := resp["by_category"].([]any)
	if !ok {
		t.Fatal("by_category should be an array")
	}
	if len(byCategory) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(byCategory))
	}

	first := byCategory[0].(map[string]any)
	if first["category_id"] != "550e8400-e29b-41d4-a716-446655440001" {
		t.Fatalf("expected category_id, got %v", first["category_id"])
	}
	if first["category_name"] != "Food" {
		t.Fatalf("expected category_name Food, got %v", first["category_name"])
	}
	if first["category_color"] != "#FF7043" {
		t.Fatalf("expected category_color #FF7043, got %v", first["category_color"])
	}
	if first["category_icon"] != "restaurant" {
		t.Fatalf("expected category_icon restaurant, got %v", first["category_icon"])
	}
	if first["total_cents"] != float64(45000) {
		t.Fatalf("expected total_cents 45000, got %v", first["total_cents"])
	}
	if first["count"] != float64(12) {
		t.Fatalf("expected count 12, got %v", first["count"])
	}

	byDate, ok := resp["by_date"].([]any)
	if !ok {
		t.Fatal("by_date should be an array")
	}
	if len(byDate) != 3 {
		t.Fatalf("expected 3 dates, got %d", len(byDate))
	}
}

func TestSummary_MissingMonth(t *testing.T) {
	db := &mockSummaryDB{}
	r := setupSummaryRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses/summary", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	errMsg, _ := resp["error"].(string)
	if errMsg == "" {
		t.Fatal("expected error message")
	}
}

func TestSummary_InvalidMonth(t *testing.T) {
	db := &mockSummaryDB{}
	r := setupSummaryRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses/summary?month=bad-format", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSummary_EmptyData(t *testing.T) {
	db := &mockSummaryDB{
		categoryTotals: []handler.CategoryTotal{},
		dailyTotals:    []handler.DateTotal{},
	}
	r := setupSummaryRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses/summary?month=2026-03", nil)
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

	byCategory, ok := resp["by_category"].([]any)
	if !ok {
		t.Fatal("by_category should be an array (not null)")
	}
	if len(byCategory) != 0 {
		t.Fatalf("expected empty by_category, got %d", len(byCategory))
	}

	byDate, ok := resp["by_date"].([]any)
	if !ok {
		t.Fatal("by_date should be an array (not null)")
	}
	if len(byDate) != 0 {
		t.Fatalf("expected empty by_date, got %d", len(byDate))
	}
}

func TestSummary_DBError(t *testing.T) {
	db := &mockSummaryDB{
		err: errors.New("database connection failed"),
	}
	r := setupSummaryRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses/summary?month=2026-03", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "Internal server error" {
		t.Fatalf("expected 'Internal server error', got %v", resp["error"])
	}
}
