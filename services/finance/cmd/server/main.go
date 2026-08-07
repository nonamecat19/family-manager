// Command server runs the finance service: Connect/JSON on :8080 for apps, gRPC on the same
// port via h2c for sibling services.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database"
	"github.com/nnc/family-manager/libs/go/events"
	"github.com/nnc/family-manager/libs/go/logger"
	"github.com/nnc/family-manager/sdk/go/finance/v1/financev1connect"
	"github.com/nnc/family-manager/services/finance/db"
	"github.com/nnc/family-manager/services/finance/internal/config"
	dbfs "github.com/nnc/family-manager/services/finance/internal/db"
	"github.com/nnc/family-manager/services/finance/internal/handler"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logger.New(logger.Options{Service: "finance", Level: cfg.LogLevel, JSON: cfg.LogJSON})
	logger.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, database.Config{URL: cfg.DatabaseURL})
	if err != nil {
		return err
	}
	defer pool.Close()

	applied, err := database.Migrate(ctx, pool, dbfs.Migrations, dbfs.MigrationsDir)
	if err != nil {
		return err
	}
	if len(applied) > 0 {
		log.Info("migrations applied", slog.Any("versions", applied))
	}

	var bus handler.EventBus
	if connected, err := events.Connect(events.Config{URL: cfg.NATSURL, Name: "finance"}); err != nil {
		// Events are notifications, not the source of truth: a broker outage must not stop
		// the ledger accepting writes.
		log.Warn("events disabled", slog.String("error", err.Error()))
	} else {
		defer connected.Close()
		if err := connected.EnsureStream(ctx, "finance"); err != nil {
			log.Warn("ensure stream failed", slog.String("error", err.Error()))
		}
		bus = connected
	}

	verifier, err := fmauth.NewVerifier(fmauth.VerifierConfig{
		JWKSURL:  cfg.JWKSURL,
		Issuer:   cfg.Issuer,
		Audience: cfg.Audience,
	})
	if err != nil {
		return err
	}

	h := handler.New(handler.Options{
		Queries:         db.New(pool),
		Bus:             bus,
		Log:             log,
		BaseCurrency:    cfg.BaseCurrency,
		DefaultPageSize: cfg.DefaultPageSize,
		MaxPageSize:     cfg.MaxPageSize,
	})

	mux := http.NewServeMux()
	// No public procedure on this service: every ledger call needs an identity.
	path, svc := financev1connect.NewFinanceServiceHandler(
		h, connect.WithInterceptors(fmauth.Interceptor(verifier)),
	)
	mux.Handle(path, svc)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		if err := pool.Ping(context.Background()); err != nil {
			http.Error(w, "db unavailable", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})

	// h2c so gRPC (sibling services) and Connect/JSON (apps) share one port.
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		log.Info("listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
