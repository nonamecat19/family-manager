package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Sentinel errors for category operations.
var (
	ErrCategoryNotFound = errors.New("category not found")
)

// MockCategory is the category representation used by the CategoryDB interface.
type MockCategory struct {
	ID        string
	UserID    string
	Name      string
	Icon      string
	Color     string
	SortOrder int
}

// CategoryDB abstracts database operations for categories.
// This allows testing with mock implementations.
type CategoryDB interface {
	CreateCategory(userID, name, icon, color string) (MockCategory, error)
	GetCategoriesByUser(userID string) ([]MockCategory, error)
	GetCategoryByID(id, userID string) (MockCategory, error)
	UpdateCategory(id, userID, name, icon, color string) error
	DeleteCategory(id, userID string) error
	UpdateCategorySortOrder(id, userID string, sortOrder int) error
}

// CategoryHandler handles category HTTP requests.
type CategoryHandler struct {
	db CategoryDB
}

// NewCategoryHandler creates a CategoryHandler with the given database.
func NewCategoryHandler(db CategoryDB) *CategoryHandler {
	return &CategoryHandler{db: db}
}

type createCategoryRequest struct {
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

// Create handles POST /api/v1/categories.
func (h *CategoryHandler) Create(c *gin.Context) {
	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Name == "" || req.Icon == "" || req.Color == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, icon, and color are required"})
		return
	}

	userID := c.GetString("user_id")
	cat, err := h.db.CreateCategory(userID, req.Name, req.Icon, req.Color)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         cat.ID,
		"user_id":    cat.UserID,
		"name":       cat.Name,
		"icon":       cat.Icon,
		"color":      cat.Color,
		"sort_order": cat.SortOrder,
	})
}

// List handles GET /api/v1/categories.
func (h *CategoryHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	cats, err := h.db.GetCategoriesByUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	result := make([]gin.H, len(cats))
	for i, cat := range cats {
		result[i] = gin.H{
			"id":         cat.ID,
			"user_id":    cat.UserID,
			"name":       cat.Name,
			"icon":       cat.Icon,
			"color":      cat.Color,
			"sort_order": cat.SortOrder,
		}
	}

	c.JSON(http.StatusOK, result)
}

type updateCategoryRequest struct {
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

// Update handles PUT /api/v1/categories/:id.
func (h *CategoryHandler) Update(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	var req updateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Name == "" || req.Icon == "" || req.Color == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, icon, and color are required"})
		return
	}

	err := h.db.UpdateCategory(id, userID, req.Name, req.Icon, req.Color)
	if err != nil {
		if errors.Is(err, ErrCategoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category updated"})
}

// Delete handles DELETE /api/v1/categories/:id.
func (h *CategoryHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")
	// Accept optional reassign_to param (no-op in Phase 3, used in Phase 4 when expenses exist)
	_ = c.Query("reassign_to")

	err := h.db.DeleteCategory(id, userID)
	if err != nil {
		if errors.Is(err, ErrCategoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}

type reorderItem struct {
	ID        string `json:"id"`
	SortOrder int    `json:"sort_order"`
}

// Reorder handles PUT /api/v1/categories/reorder.
func (h *CategoryHandler) Reorder(c *gin.Context) {
	var items []reorderItem
	if err := c.ShouldBindJSON(&items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	userID := c.GetString("user_id")
	for _, item := range items {
		if err := h.db.UpdateCategorySortOrder(item.ID, userID, item.SortOrder); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Categories reordered"})
}

type bulkCreateRequest struct {
	Categories []createCategoryRequest `json:"categories"`
}

// BulkCreate handles POST /api/v1/categories/bulk.
func (h *CategoryHandler) BulkCreate(c *gin.Context) {
	var req bulkCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(req.Categories) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "categories array is required"})
		return
	}

	userID := c.GetString("user_id")
	result := make([]gin.H, 0, len(req.Categories))

	for _, catReq := range req.Categories {
		if catReq.Name == "" || catReq.Icon == "" || catReq.Color == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "each category requires name, icon, and color"})
			return
		}

		cat, err := h.db.CreateCategory(userID, catReq.Name, catReq.Icon, catReq.Color)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		result = append(result, gin.H{
			"id":         cat.ID,
			"user_id":    cat.UserID,
			"name":       cat.Name,
			"icon":       cat.Icon,
			"color":      cat.Color,
			"sort_order": cat.SortOrder,
		})
	}

	c.JSON(http.StatusCreated, result)
}
