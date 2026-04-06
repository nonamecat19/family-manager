package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck returns a simple 200 OK response to indicate the server is running.
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
