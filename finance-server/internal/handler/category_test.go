package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/nnc/finance-tracker/server/internal/handler"
)

const testUserID = "550e8400-e29b-41d4-a716-446655440000"

// mockCategoryDB implements handler.CategoryDB for testing.
type mockCategoryDB struct {
	categories []handler.MockCategory
	nextID     int
}

func newMockCategoryDB() *mockCategoryDB {
	return &mockCategoryDB{
		categories: make([]handler.MockCategory, 0),
		nextID:     1,
	}
}

func (m *mockCategoryDB) CreateCategory(userID, name, icon, color string) (handler.MockCategory, error) {
	cat := handler.MockCategory{
		ID:        idForIndex(m.nextID),
		UserID:    userID,
		Name:      name,
		Icon:      icon,
		Color:     color,
		SortOrder: len(m.categories),
	}
	m.nextID++
	m.categories = append(m.categories, cat)
	return cat, nil
}

func (m *mockCategoryDB) GetCategoriesByUser(userID string) ([]handler.MockCategory, error) {
	var result []handler.MockCategory
	for _, cat := range m.categories {
		if cat.UserID == userID {
			result = append(result, cat)
		}
	}
	return result, nil
}

func (m *mockCategoryDB) GetCategoryByID(id, userID string) (handler.MockCategory, error) {
	for _, cat := range m.categories {
		if cat.ID == id && cat.UserID == userID {
			return cat, nil
		}
	}
	return handler.MockCategory{}, handler.ErrCategoryNotFound
}

func (m *mockCategoryDB) UpdateCategory(id, userID, name, icon, color string) error {
	for i, cat := range m.categories {
		if cat.ID == id && cat.UserID == userID {
			m.categories[i].Name = name
			m.categories[i].Icon = icon
			m.categories[i].Color = color
			return nil
		}
	}
	return handler.ErrCategoryNotFound
}

func (m *mockCategoryDB) DeleteCategory(id, userID string) error {
	for i, cat := range m.categories {
		if cat.ID == id && cat.UserID == userID {
			m.categories = append(m.categories[:i], m.categories[i+1:]...)
			return nil
		}
	}
	return handler.ErrCategoryNotFound
}

func (m *mockCategoryDB) UpdateCategorySortOrder(id, userID string, sortOrder int) error {
	for i, cat := range m.categories {
		if cat.ID == id && cat.UserID == userID {
			m.categories[i].SortOrder = sortOrder
			return nil
		}
	}
	return nil
}

func idForIndex(i int) string {
	return "cat-" + string(rune('0'+i))
}

func setupCategoryRouter(db handler.CategoryDB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewCategoryHandler(db)

	cats := r.Group("/api/v1/categories")
	cats.Use(func(c *gin.Context) {
		c.Set("user_id", testUserID)
		c.Next()
	})
	{
		cats.POST("", h.Create)
		cats.GET("", h.List)
		cats.PUT("/reorder", h.Reorder)
		cats.POST("/bulk", h.BulkCreate)
		cats.PUT("/:id", h.Update)
		cats.DELETE("/:id", h.Delete)
	}
	return r
}

func TestCreateCategory_Success(t *testing.T) {
	db := newMockCategoryDB()
	r := setupCategoryRouter(db)

	body, _ := json.Marshal(map[string]string{
		"name":  "Food",
		"icon":  "restaurant",
		"color": "#FF7043",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["name"] != "Food" {
		t.Fatalf("expected name Food, got %v", resp["name"])
	}
	if resp["icon"] != "restaurant" {
		t.Fatalf("expected icon restaurant, got %v", resp["icon"])
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Fatal("response should include id")
	}
	if resp["sort_order"] == nil {
		t.Fatal("response should include sort_order")
	}
}

func TestCreateCategory_MissingFields(t *testing.T) {
	db := newMockCategoryDB()
	r := setupCategoryRouter(db)

	body, _ := json.Marshal(map[string]string{
		"name": "Food",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListCategories_Success(t *testing.T) {
	db := newMockCategoryDB()
	r := setupCategoryRouter(db)

	// Create two categories first
	for _, name := range []string{"Food", "Transport"} {
		body, _ := json.Marshal(map[string]string{
			"name": name, "icon": "icon", "color": "#000",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}

	// List
	req := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(resp))
	}
}

func TestUpdateCategory_Success(t *testing.T) {
	db := newMockCategoryDB()
	r := setupCategoryRouter(db)

	// Create a category
	body, _ := json.Marshal(map[string]string{
		"name": "Food", "icon": "restaurant", "color": "#FF7043",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created map[string]any
	json.Unmarshal(w.Body.Bytes(), &created)
	catID := created["id"].(string)

	// Update it
	body, _ = json.Marshal(map[string]string{
		"name": "Groceries", "icon": "local_grocery_store", "color": "#66BB6A",
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/categories/"+catID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateCategory_NotFound(t *testing.T) {
	db := newMockCategoryDB()
	r := setupCategoryRouter(db)

	body, _ := json.Marshal(map[string]string{
		"name": "Groceries", "icon": "local_grocery_store", "color": "#66BB6A",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/categories/nonexistent-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCategory_Success(t *testing.T) {
	db := newMockCategoryDB()
	r := setupCategoryRouter(db)

	// Create a category
	body, _ := json.Marshal(map[string]string{
		"name": "Food", "icon": "restaurant", "color": "#FF7043",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var created map[string]any
	json.Unmarshal(w.Body.Bytes(), &created)
	catID := created["id"].(string)

	// Delete it
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/categories/"+catID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCategory_NotFound(t *testing.T) {
	db := newMockCategoryDB()
	r := setupCategoryRouter(db)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/categories/nonexistent-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReorderCategories_Success(t *testing.T) {
	db := newMockCategoryDB()
	r := setupCategoryRouter(db)

	// Create two categories
	var ids []string
	for _, name := range []string{"Food", "Transport"} {
		body, _ := json.Marshal(map[string]string{
			"name": name, "icon": "icon", "color": "#000",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/categories", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var created map[string]any
		json.Unmarshal(w.Body.Bytes(), &created)
		ids = append(ids, created["id"].(string))
	}

	// Reorder (swap)
	reorderBody, _ := json.Marshal([]map[string]any{
		{"id": ids[0], "sort_order": 1},
		{"id": ids[1], "sort_order": 0},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/categories/reorder", bytes.NewReader(reorderBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBulkCreateCategories_Success(t *testing.T) {
	db := newMockCategoryDB()
	r := setupCategoryRouter(db)

	body, _ := json.Marshal(map[string]any{
		"categories": []map[string]string{
			{"name": "Food", "icon": "restaurant", "color": "#FF7043"},
			{"name": "Transport", "icon": "directions_car", "color": "#42A5F5"},
			{"name": "Shopping", "icon": "shopping_cart", "color": "#AB47BC"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/categories/bulk", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp []map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp) != 3 {
		t.Fatalf("expected 3 categories, got %d", len(resp))
	}
}
