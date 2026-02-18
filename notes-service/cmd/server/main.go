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
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/nnc/notes-manager-backend/internal/auth"
	"github.com/nnc/notes-manager-backend/internal/config"
	"github.com/nnc/notes-manager-backend/internal/database"
	"github.com/nnc/notes-manager-backend/internal/database/sqlc"
	"github.com/nnc/notes-manager-backend/internal/graph"
	"github.com/nnc/notes-manager-backend/internal/storage"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	// Database
	pool, err := database.NewPool(ctx, cfg.Postgres.DSN())
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	// Run migrations
	runMigrations(cfg.Postgres.DSN())

	queries := sqlc.New(pool)

	// Auth
	jwtMgr := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.AccessDuration, cfg.JWT.RefreshDuration)
	authSvc := auth.NewService(queries, jwtMgr)

	// Storage
	store, err := storage.NewMinIOStorage(ctx, cfg.MinIO)
	if err != nil {
		log.Fatalf("minio: %v", err)
	}

	// GraphQL
	resolver := &graph.Resolver{
		Queries: queries,
		Auth:    authSvc,
		JWT:     jwtMgr,
		Storage: store,
	}

	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{MaxUploadSize: 10 << 20}) // 10 MB
	srv.Use(extension.Introspection{})

	// Routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("GET /playground", playground.Handler("Notes Manager", "/graphql"))
	mux.Handle("POST /graphql", auth.Middleware(jwtMgr)(srv))

	httpSrv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	go func() {
		log.Printf("listening on :%s", cfg.Port)
		log.Printf("playground at http://localhost:%s/", cfg.Port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	shutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	httpSrv.Shutdown(shutCtx)
}

func runMigrations(dsn string) {
	// golang-migrate pgx/v5 driver uses "pgx5://" scheme
	migrateDSN := strings.Replace(dsn, "postgres://", "pgx5://", 1)
	m, err := migrate.New("file://internal/database/migrations", migrateDSN)
	if err != nil {
		log.Fatalf("migrations init: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migrations up: %v", err)
	}
	log.Println("migrations applied")
}
