package main

import (
	"fmt"
	"log"
	"os"

	"github.com/finance-manager/backend/internal/infrastructure/config"
	"github.com/finance-manager/backend/internal/infrastructure/database"
	"github.com/finance-manager/backend/internal/repository/postgres"
	"github.com/finance-manager/backend/internal/usecase"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	queries := postgres.New(db.Pool)
	userRepo := postgres.NewUserRepository(queries)

	// Initialize use case
	userUC := usecase.NewUserUseCase(userRepo)

	// Get email and password from command line args or use defaults
	email := "test@example.com"
	password := "password123"

	if len(os.Args) > 1 {
		email = os.Args[1]
	}
	if len(os.Args) > 2 {
		password = os.Args[2]
	}

	// Create user
	user, err := userUC.CreateUser(email, password)
	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	fmt.Printf("✅ User created successfully!\n")
	fmt.Printf("   ID: %d\n", user.ID)
	fmt.Printf("   Email: %s\n", user.Email)
	fmt.Printf("   Created At: %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("\nYou can now use user ID %d to create wallets.\n", user.ID)
}
