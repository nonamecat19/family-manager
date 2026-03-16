package main

import (
	"context"
	"log"

	"github.com/nnc/finance-tracker/server/internal/config"
	"github.com/nnc/finance-tracker/server/internal/db"
	"github.com/nnc/finance-tracker/server/internal/db/sqlc"
	"github.com/nnc/finance-tracker/server/internal/handler"
	"github.com/nnc/finance-tracker/server/internal/router"
	"github.com/nnc/finance-tracker/server/internal/service"
)

func main() {
	cfg := config.Load()

	pool, err := db.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	queries := sqlc.New(pool)
	authDB := handler.NewPgAuthDB(queries)
	categoryDB := handler.NewPgCategoryDB(queries)
	expenseDB := handler.NewPgExpenseDB(queries)
	summaryDB := handler.NewPgSummaryDB(queries)
	familyDB := handler.NewPgFamilyDB(queries)
	familyViewDB := handler.NewPgFamilyViewDB(queries)
	authSvc := service.NewAuthService(cfg.JWTSecret)

	r := router.Setup(authDB, categoryDB, expenseDB, summaryDB, familyDB, familyViewDB, authSvc)

	log.Printf("Server starting on :%s", cfg.Port)
	log.Fatal(r.Run(":" + cfg.Port))
}
