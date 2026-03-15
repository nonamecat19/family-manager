package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nnc/finance-tracker/server/internal/handler"
)

var errFKViolation = errors.New("violates foreign key constraint")

// mockExpenseDB implements handler.ExpenseDB for testing.
type mockExpenseDB struct {
	expenses []handler.MockExpense
	nextID   int
	// If set, CreateExpense returns this error.
	createErr error
}

func newMockExpenseDB() *mockExpenseDB {
	return &mockExpenseDB{
		expenses: make([]handler.MockExpense, 0),
		nextID:   1,
	}
}

func (m *mockExpenseDB) CreateExpense(userID, categoryID string, amountCents int64, note string, expenseDate time.Time) (handler.MockExpense, error) {
	if m.createErr != nil {
		return handler.MockExpense{}, m.createErr
	}
	exp := handler.MockExpense{
		ID:          expIDForIndex(m.nextID),
		UserID:      userID,
		CategoryID:  categoryID,
		AmountCents: amountCents,
		Note:        note,
		ExpenseDate: expenseDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.nextID++
	m.expenses = append(m.expenses, exp)
	return exp, nil
}

func (m *mockExpenseDB) GetExpensesByUser(userID string, limit, offset int) ([]handler.MockExpense, error) {
	var result []handler.MockExpense
	for _, exp := range m.expenses {
		if exp.UserID == userID {
			result = append(result, exp)
		}
	}
	// Apply offset and limit
	if offset >= len(result) {
		return []handler.MockExpense{}, nil
	}
	result = result[offset:]
	if limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

func expIDForIndex(i int) string {
	return "exp-" + string(rune('0'+i))
}

func setupExpenseRouter(db handler.ExpenseDB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewExpenseHandler(db)

	expenses := r.Group("/api/v1/expenses")
	expenses.Use(func(c *gin.Context) {
		c.Set("user_id", testUserID)
		c.Next()
	})
	{
		expenses.POST("", h.Create)
		expenses.GET("", h.List)
	}
	return r
}

func TestCreateExpense_Success(t *testing.T) {
	db := newMockExpenseDB()
	r := setupExpenseRouter(db)

	body, _ := json.Marshal(map[string]any{
		"category_id":  "550e8400-e29b-41d4-a716-446655440001",
		"amount_cents": 1500,
		"note":         "Lunch",
		"expense_date": "2026-03-15",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["id"] == nil || resp["id"] == "" {
		t.Fatal("response should include id")
	}
	if resp["category_id"] != "550e8400-e29b-41d4-a716-446655440001" {
		t.Fatalf("expected category_id, got %v", resp["category_id"])
	}
	// amount_cents must be integer, not float
	if resp["amount_cents"] != float64(1500) {
		t.Fatalf("expected amount_cents 1500, got %v", resp["amount_cents"])
	}
	if resp["expense_date"] != "2026-03-15" {
		t.Fatalf("expected expense_date 2026-03-15, got %v", resp["expense_date"])
	}
	if resp["note"] != "Lunch" {
		t.Fatalf("expected note Lunch, got %v", resp["note"])
	}
}

func TestCreateExpense_MissingCategoryID(t *testing.T) {
	db := newMockExpenseDB()
	r := setupExpenseRouter(db)

	body, _ := json.Marshal(map[string]any{
		"amount_cents": 1500,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateExpense_InvalidAmount_Zero(t *testing.T) {
	db := newMockExpenseDB()
	r := setupExpenseRouter(db)

	body, _ := json.Marshal(map[string]any{
		"category_id":  "550e8400-e29b-41d4-a716-446655440001",
		"amount_cents": 0,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateExpense_InvalidAmount_Negative(t *testing.T) {
	db := newMockExpenseDB()
	r := setupExpenseRouter(db)

	body, _ := json.Marshal(map[string]any{
		"category_id":  "550e8400-e29b-41d4-a716-446655440001",
		"amount_cents": -100,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateExpense_InvalidDate(t *testing.T) {
	db := newMockExpenseDB()
	r := setupExpenseRouter(db)

	body, _ := json.Marshal(map[string]any{
		"category_id":  "550e8400-e29b-41d4-a716-446655440001",
		"amount_cents": 1500,
		"expense_date": "not-a-date",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateExpense_DefaultDate(t *testing.T) {
	db := newMockExpenseDB()
	r := setupExpenseRouter(db)

	body, _ := json.Marshal(map[string]any{
		"category_id":  "550e8400-e29b-41d4-a716-446655440001",
		"amount_cents": 1500,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Should have today's date
	today := time.Now().Format("2006-01-02")
	if resp["expense_date"] != today {
		t.Fatalf("expected expense_date %s, got %v", today, resp["expense_date"])
	}
}

func TestCreateExpense_EmptyNote(t *testing.T) {
	db := newMockExpenseDB()
	r := setupExpenseRouter(db)

	body, _ := json.Marshal(map[string]any{
		"category_id":  "550e8400-e29b-41d4-a716-446655440001",
		"amount_cents": 1500,
		"expense_date": "2026-03-15",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateExpense_AmountCentsIsInteger(t *testing.T) {
	db := newMockExpenseDB()
	r := setupExpenseRouter(db)

	body, _ := json.Marshal(map[string]any{
		"category_id":  "550e8400-e29b-41d4-a716-446655440001",
		"amount_cents": 99999,
		"expense_date": "2026-03-15",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	// JSON numbers are float64 in Go, but amount_cents should be a whole number
	amountVal, ok := resp["amount_cents"].(float64)
	if !ok {
		t.Fatalf("amount_cents should be a number, got %T", resp["amount_cents"])
	}
	if amountVal != 99999 {
		t.Fatalf("expected 99999, got %v", amountVal)
	}
}

func TestListExpenses_Success(t *testing.T) {
	db := newMockExpenseDB()
	r := setupExpenseRouter(db)

	// Create two expenses
	for _, amt := range []int{1500, 2500} {
		body, _ := json.Marshal(map[string]any{
			"category_id":  "550e8400-e29b-41d4-a716-446655440001",
			"amount_cents": amt,
			"expense_date": "2026-03-15",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}

	// List
	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp) != 2 {
		t.Fatalf("expected 2 expenses, got %d", len(resp))
	}
}

func TestListExpenses_WithLimitOffset(t *testing.T) {
	db := newMockExpenseDB()
	r := setupExpenseRouter(db)

	// Create three expenses
	for _, amt := range []int{1000, 2000, 3000} {
		body, _ := json.Marshal(map[string]any{
			"category_id":  "550e8400-e29b-41d4-a716-446655440001",
			"amount_cents": amt,
			"expense_date": "2026-03-15",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}

	// List with limit=1&offset=1
	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses?limit=1&offset=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp) != 1 {
		t.Fatalf("expected 1 expense with limit=1, got %d", len(resp))
	}
}
