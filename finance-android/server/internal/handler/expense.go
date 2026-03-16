package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Sentinel errors for expense operations.
var (
	ErrExpenseNotFound = errors.New("expense not found")
)

// MockExpense is the expense representation used by the ExpenseDB interface.
type MockExpense struct {
	ID          string
	UserID      string
	CategoryID  string
	AmountCents int64
	Note        string
	ExpenseDate time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ExpenseDB abstracts database operations for expenses.
// This allows testing with mock implementations.
type ExpenseDB interface {
	CreateExpense(userID, categoryID string, amountCents int64, note string, expenseDate time.Time) (MockExpense, error)
	GetExpensesByUser(userID string, limit, offset int) ([]MockExpense, error)
	GetExpensesByUserFiltered(userID string, limit, offset int, dateFrom, dateTo *time.Time, categoryID string) ([]MockExpense, error)
	UpdateExpense(id, userID, categoryID string, amountCents int64, note string, expenseDate time.Time) (MockExpense, error)
	DeleteExpense(id, userID string) error
}

// ExpenseHandler handles expense HTTP requests.
type ExpenseHandler struct {
	db ExpenseDB
}

// NewExpenseHandler creates an ExpenseHandler with the given database.
func NewExpenseHandler(db ExpenseDB) *ExpenseHandler {
	return &ExpenseHandler{db: db}
}

type createExpenseRequest struct {
	CategoryID  string `json:"category_id"`
	AmountCents int64  `json:"amount_cents"`
	Note        string `json:"note"`
	ExpenseDate string `json:"expense_date"`
}

// Create handles POST /api/v1/expenses.
func (h *ExpenseHandler) Create(c *gin.Context) {
	var req createExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.CategoryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id is required"})
		return
	}

	if req.AmountCents <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount_cents must be greater than 0"})
		return
	}

	var expenseDate time.Time
	if req.ExpenseDate == "" {
		expenseDate = time.Now()
	} else {
		var err error
		expenseDate, err = time.Parse("2006-01-02", req.ExpenseDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "expense_date must be in YYYY-MM-DD format"})
			return
		}
	}

	userID := c.GetString("user_id")
	exp, err := h.db.CreateExpense(userID, req.CategoryID, req.AmountCents, req.Note, expenseDate)
	if err != nil {
		if strings.Contains(err.Error(), "foreign key") || strings.Contains(err.Error(), "violates") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           exp.ID,
		"user_id":      exp.UserID,
		"category_id":  exp.CategoryID,
		"amount_cents": exp.AmountCents,
		"note":         exp.Note,
		"expense_date": exp.ExpenseDate.Format("2006-01-02"),
		"created_at":   exp.CreatedAt,
	})
}

type updateExpenseRequest struct {
	CategoryID  string `json:"category_id"`
	AmountCents int64  `json:"amount_cents"`
	Note        string `json:"note"`
	ExpenseDate string `json:"expense_date"`
}

// Update handles PUT /api/v1/expenses/:id.
func (h *ExpenseHandler) Update(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	var req updateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.CategoryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id is required"})
		return
	}

	if req.AmountCents <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount_cents must be greater than 0"})
		return
	}

	var expenseDate time.Time
	if req.ExpenseDate == "" {
		expenseDate = time.Now()
	} else {
		var err error
		expenseDate, err = time.Parse("2006-01-02", req.ExpenseDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "expense_date must be in YYYY-MM-DD format"})
			return
		}
	}

	exp, err := h.db.UpdateExpense(id, userID, req.CategoryID, req.AmountCents, req.Note, expenseDate)
	if err != nil {
		if errors.Is(err, ErrExpenseNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found"})
			return
		}
		if strings.Contains(err.Error(), "foreign key") || strings.Contains(err.Error(), "violates") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           exp.ID,
		"user_id":      exp.UserID,
		"category_id":  exp.CategoryID,
		"amount_cents": exp.AmountCents,
		"note":         exp.Note,
		"expense_date": exp.ExpenseDate.Format("2006-01-02"),
		"created_at":   exp.CreatedAt,
		"updated_at":   exp.UpdatedAt,
	})
}

// Delete handles DELETE /api/v1/expenses/:id.
func (h *ExpenseHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")

	err := h.db.DeleteExpense(id, userID)
	if err != nil {
		if errors.Is(err, ErrExpenseNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}

// List handles GET /api/v1/expenses.
func (h *ExpenseHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")

	limit := 50
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	var dateFrom, dateTo *time.Time
	if df := c.Query("date_from"); df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			dateFrom = &t
		}
	}
	if dt := c.Query("date_to"); dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			dateTo = &t
		}
	}
	categoryID := c.Query("category_id")

	expenses, err := h.db.GetExpensesByUserFiltered(userID, limit, offset, dateFrom, dateTo, categoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	result := make([]gin.H, len(expenses))
	for i, exp := range expenses {
		result[i] = gin.H{
			"id":           exp.ID,
			"user_id":      exp.UserID,
			"category_id":  exp.CategoryID,
			"amount_cents": exp.AmountCents,
			"note":         exp.Note,
			"expense_date": exp.ExpenseDate.Format("2006-01-02"),
			"created_at":   exp.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, result)
}
