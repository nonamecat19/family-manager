package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/finance-manager/backend/internal/handler/graphql/generated"
	"github.com/finance-manager/backend/internal/handler/graphql/resolver"
	"github.com/finance-manager/backend/internal/infrastructure/config"
	"github.com/finance-manager/backend/internal/infrastructure/database"
	"github.com/finance-manager/backend/internal/infrastructure/scheduler"
	"github.com/finance-manager/backend/internal/repository/postgres"
	"github.com/finance-manager/backend/internal/usecase"
	"github.com/rs/cors"
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
	walletRepo := postgres.NewWalletRepository(queries)
	transactionRepo := postgres.NewTransactionRepository(queries)
	recurringRepo := postgres.NewRecurringRepository(queries)
	investmentRepo := postgres.NewInvestmentRepository(queries)
	currencyRepo := postgres.NewCurrencyRepository(queries)

	// Initialize use cases
	userUC := usecase.NewUserUseCase(userRepo)
	walletUC := usecase.NewWalletUseCase(walletRepo, userRepo)
	transactionUC := usecase.NewTransactionUseCase(transactionRepo, walletRepo)
	recurringUC := usecase.NewRecurringUseCase(recurringRepo, walletRepo)
	investmentUC := usecase.NewInvestmentUseCase(investmentRepo, walletRepo)
	currencyUC := usecase.NewCurrencyUseCase(currencyRepo)

	// Initialize scheduler
	sched := scheduler.NewScheduler(recurringRepo, transactionRepo, walletRepo)
	sched.Start()
	defer sched.Stop()

	// Initialize GraphQL resolver
	graphqlResolver := &resolver.Resolver{
		UserUseCase:        userUC,
		WalletUseCase:      walletUC,
		TransactionUseCase: transactionUC,
		RecurringUseCase:   recurringUC,
		InvestmentUseCase:  investmentUC,
		CurrencyUseCase:    currencyUC,
	}

	// Create GraphQL executable schema
	graphqlConfig := generated.Config{Resolvers: graphqlResolver}
	executableSchema := generated.NewExecutableSchema(graphqlConfig)

	// Create GraphQL handler
	graphqlHandler := handler.NewDefaultServer(executableSchema)

	// Configure CORS
	var corsOptions cors.Options

	if cfg.Environment == "development" {
		// In development, allow localhost on any port
		corsOptions = cors.Options{
			AllowOriginFunc: func(origin string) bool {
				return strings.HasPrefix(origin, "http://localhost:") ||
					strings.HasPrefix(origin, "http://127.0.0.1:")
			},
			AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
			AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With"},
			AllowCredentials: true,
			Debug:            true,
		}
	} else {
		// In production, use specific origins
		allowedOrigins := []string{
			"http://localhost:5173", // Frontend dev server
			"http://localhost:3000", // Fallback for other setups
		}

		// Allow additional origins from environment variable
		if corsOrigins := os.Getenv("CORS_ORIGINS"); corsOrigins != "" {
			allowedOrigins = append(allowedOrigins, strings.Split(corsOrigins, ",")...)
		}

		corsOptions = cors.Options{
			AllowedOrigins:   allowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
			AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With"},
			AllowCredentials: true,
			Debug:            false,
		}
	}

	c := cors.New(corsOptions)

	// Create a new mux
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// GraphQL endpoint with CORS applied directly
	mux.Handle("/graphql", c.Handler(graphqlHandler))

	// GraphQL playground (for development) - also with CORS
	mux.Handle("/playground", c.Handler(playground.Handler("GraphQL Playground", "/graphql")))

	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: mux,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
