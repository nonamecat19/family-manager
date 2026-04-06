package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// FamilyExpense represents a single expense in the family feed.
type FamilyExpense struct {
	ID            string
	UserID        string
	UserEmail     string
	CategoryID    string
	CategoryName  string
	CategoryColor string
	CategoryIcon  string
	AmountCents   int64
	Note          string
	ExpenseDate   time.Time
	CreatedAt     time.Time
}

// FamilyMemberTotal represents per-user expense totals.
type FamilyMemberTotal struct {
	UserID     string
	UserEmail  string
	TotalCents int64
	Count      int
}

// FamilyCategoryTotal represents per-category expense totals for a family.
type FamilyCategoryTotal struct {
	CategoryID    string
	CategoryName  string
	CategoryColor string
	CategoryIcon  string
	TotalCents    int64
	Count         int
}

// FamilyViewDB abstracts database operations for family expense views.
type FamilyViewDB interface {
	GetFamilyExpenses(familyID string, limit, offset int) ([]FamilyExpense, error)
	GetFamilyMemberTotals(familyID string, dateFrom, dateTo time.Time) ([]FamilyMemberTotal, error)
	GetFamilyCategoryTotals(familyID string, dateFrom, dateTo time.Time) ([]FamilyCategoryTotal, error)
}

// FamilyViewHandler handles family view HTTP requests.
type FamilyViewHandler struct {
	familyDB FamilyDB
	viewDB   FamilyViewDB
}

// NewFamilyViewHandler creates a FamilyViewHandler with the given databases.
func NewFamilyViewHandler(familyDB FamilyDB, viewDB FamilyViewDB) *FamilyViewHandler {
	return &FamilyViewHandler{familyDB: familyDB, viewDB: viewDB}
}

// FamilyFeed handles GET /api/v1/families/me/expenses.
func (h *FamilyViewHandler) FamilyFeed(c *gin.Context) {
	userID := c.GetString("user_id")

	family, err := h.familyDB.GetFamilyByUserID(userID)
	if err != nil {
		if errors.Is(err, ErrFamilyNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no family"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	limit, offset := parsePagination(c)

	expenses, err := h.viewDB.GetFamilyExpenses(family.ID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	result := make([]gin.H, len(expenses))
	for i, e := range expenses {
		result[i] = gin.H{
			"id":             e.ID,
			"user_id":        e.UserID,
			"user_email":     e.UserEmail,
			"category_id":    e.CategoryID,
			"category_name":  e.CategoryName,
			"category_color": e.CategoryColor,
			"category_icon":  e.CategoryIcon,
			"amount_cents":   e.AmountCents,
			"note":           e.Note,
			"expense_date":   e.ExpenseDate.Format("2006-01-02"),
		}
	}

	c.JSON(http.StatusOK, result)
}

// FamilySummary handles GET /api/v1/families/me/summary.
func (h *FamilyViewHandler) FamilySummary(c *gin.Context) {
	userID := c.GetString("user_id")

	family, err := h.familyDB.GetFamilyByUserID(userID)
	if err != nil {
		if errors.Is(err, ErrFamilyNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no family"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	month := c.Query("month")
	if month == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "month query parameter is required (format: YYYY-MM)"})
		return
	}

	parsed, err := time.Parse("2006-01", month)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "month query parameter is required (format: YYYY-MM)"})
		return
	}

	dateFrom := parsed
	dateTo := time.Date(parsed.Year(), parsed.Month()+1, 0, 0, 0, 0, 0, time.UTC)

	memberTotals, err := h.viewDB.GetFamilyMemberTotals(family.ID, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	categoryTotals, err := h.viewDB.GetFamilyCategoryTotals(family.ID, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	var totalCents int64
	for _, mt := range memberTotals {
		totalCents += mt.TotalCents
	}

	byPerson := make([]gin.H, len(memberTotals))
	for i, mt := range memberTotals {
		byPerson[i] = gin.H{
			"user_id":       mt.UserID,
			"user_email":    mt.UserEmail,
			"total_cents":   mt.TotalCents,
			"expense_count": mt.Count,
		}
	}

	byCategory := make([]gin.H, len(categoryTotals))
	for i, ct := range categoryTotals {
		byCategory[i] = gin.H{
			"category_id":    ct.CategoryID,
			"category_name":  ct.CategoryName,
			"category_color": ct.CategoryColor,
			"category_icon":  ct.CategoryIcon,
			"total_cents":    ct.TotalCents,
			"expense_count":  ct.Count,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_cents": totalCents,
		"by_person":   byPerson,
		"by_category": byCategory,
	})
}

// parsePagination extracts limit and offset from query params with defaults.
func parsePagination(c *gin.Context) (limit, offset int) {
	limit = 50
	offset = 0

	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}

	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			offset = v
		}
	}
	if offset < 0 {
		offset = 0
	}

	return limit, offset
}
