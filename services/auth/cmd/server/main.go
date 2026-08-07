package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"github.com/nnc/auth-service/internal/config"
	grpcserver "github.com/nnc/auth-service/internal/grpc"
	"github.com/nnc/auth-service/internal/handler"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Database
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()
	_ = pool // used by repositories once implemented

	// HTTP server
	r := gin.Default()
	h := handler.New()
	h.RegisterRoutes(r)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler: r,
	}

	go func() {
		log.Printf("HTTP server listening on :%s", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server: %v", err)
		}
	}()

	// gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("gRPC listen: %v", err)
	}

	grpcSrv := grpc.NewServer()
	grpcserver.Register(grpcSrv, grpcserver.New())

	go func() {
		log.Printf("gRPC server listening on :%s", cfg.GRPCPort)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatalf("gRPC server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	grpcSrv.GracefulStop()
	if err := httpServer.Shutdown(context.Background()); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
}
