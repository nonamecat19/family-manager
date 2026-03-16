package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// CategoryTotal represents a category's aggregated expense data.
type CategoryTotal struct {
	CategoryID    string
	CategoryName  string
	CategoryColor string
	CategoryIcon  string
	TotalCents    int64
	Count         int
}

// DateTotal represents a single date's aggregated expense total.
type DateTotal struct {
	Date       string // "2006-01-02" format
	TotalCents int64
}

// SummaryDB abstracts database operations for expense summaries.
type SummaryDB interface {
	GetCategoryTotals(userID string, dateFrom, dateTo time.Time) ([]CategoryTotal, error)
	GetDailyTotals(userID string, dateFrom, dateTo time.Time) ([]DateTotal, error)
}

// SummaryHandler handles summary HTTP requests.
type SummaryHandler struct {
	db SummaryDB
}

// NewSummaryHandler creates a SummaryHandler with the given database.
func NewSummaryHandler(db SummaryDB) *SummaryHandler {
	return &SummaryHandler{db: db}
}

// Summary handles GET /api/v1/expenses/summary.
func (h *SummaryHandler) Summary(c *gin.Context) {
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

	userID := c.GetString("user_id")

	categoryTotals, err := h.db.GetCategoryTotals(userID, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	dailyTotals, err := h.db.GetDailyTotals(userID, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	var totalCents int64
	for _, ct := range categoryTotals {
		totalCents += ct.TotalCents
	}

	byCategory := make([]gin.H, len(categoryTotals))
	for i, ct := range categoryTotals {
		byCategory[i] = gin.H{
			"category_id":    ct.CategoryID,
			"category_name":  ct.CategoryName,
			"category_color": ct.CategoryColor,
			"category_icon":  ct.CategoryIcon,
			"total_cents":    ct.TotalCents,
			"count":          ct.Count,
		}
	}

	byDate := make([]gin.H, len(dailyTotals))
	for i, dt := range dailyTotals {
		byDate[i] = gin.H{
			"date":        dt.Date,
			"total_cents": dt.TotalCents,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"month":       month,
		"total_cents": totalCents,
		"by_category": byCategory,
		"by_date":     byDate,
	})
}
