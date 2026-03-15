package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nnc/finance-tracker/server/internal/handler"
	"github.com/nnc/finance-tracker/server/internal/middleware"
	"github.com/nnc/finance-tracker/server/internal/service"
)

// Setup creates and configures the Gin router with CORS middleware and routes.
func Setup(db handler.AuthDB, categoryDB handler.CategoryDB, expenseDB handler.ExpenseDB, authSvc *service.AuthService) *gin.Engine {
	r := gin.Default()

	r.Use(corsMiddleware())

	api := r.Group("/api/v1")
	{
		api.GET("/health", handler.HealthCheck)

		// Auth routes (public)
		authHandler := handler.NewAuthHandler(db, authSvc)
		auth := api.Group("/auth")
		{
			auth.POST("/signup", authHandler.Signup)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)
		}

		// Protected routes
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware(authSvc.JWTSecret()))
		{
			categoryHandler := handler.NewCategoryHandler(categoryDB)
			categories := protected.Group("categories")
			{
				categories.POST("", categoryHandler.Create)
				categories.GET("", categoryHandler.List)
				categories.PUT("/reorder", categoryHandler.Reorder)
				categories.POST("/bulk", categoryHandler.BulkCreate)
				categories.PUT("/:id", categoryHandler.Update)
				categories.DELETE("/:id", categoryHandler.Delete)
			}

			expenseHandler := handler.NewExpenseHandler(expenseDB)
			expenses := protected.Group("expenses")
			{
				expenses.POST("", expenseHandler.Create)
				expenses.GET("", expenseHandler.List)
			}
		}
	}

	return r
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
