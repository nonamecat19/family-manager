// Command server runs the auth service: auth.v1.AuthService over Connect, plus the JWKS
// endpoint every other service fetches to verify the tokens minted here.
package main

import (
	"context"
	"encoding/json"
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

	"github.com/nnc/family-manager/libs/go/database"
	"github.com/nnc/family-manager/libs/go/logger"
	"github.com/nnc/family-manager/libs/go/rpc"
	"github.com/nnc/family-manager/sdk/go/auth/v1/authv1connect"
	"github.com/nnc/family-manager/services/auth/db"
	"github.com/nnc/family-manager/services/auth/internal/config"
	dbfs "github.com/nnc/family-manager/services/auth/internal/db"
	"github.com/nnc/family-manager/services/auth/internal/family"
	"github.com/nnc/family-manager/services/auth/internal/handler"
	"github.com/nnc/family-manager/services/auth/internal/token"
)

// maxRequestBytes bounds a decoded request body. Nothing this service accepts is large — the
// biggest message is a household with its members — so the cap is small enough that an
// oversize body is refused during the read rather than after it is buffered.
const maxRequestBytes = 1 << 20 // 1 MiB

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

	log := logger.New(logger.Options{Service: "auth", Level: cfg.LogLevel, JSON: cfg.LogJSON})
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

	signer, err := token.NewSigner(token.Config{
		PrivateKeyPEM: cfg.SigningKeyPEM,
		Issuer:        cfg.Issuer,
		Audience:      cfg.Audience,
		TTL:           cfg.AccessTTL,
	})
	if err != nil {
		return err
	}

	var lookup handler.FamilyLookup
	if cfg.FamilyAddr != "" {
		lookup = family.New(cfg.FamilyAddr, 2*time.Second)
	} else {
		log.Warn("AUTH_FAMILY_ADDR is unset; tokens will carry no family_id")
	}

	h := handler.New(handler.Options{
		Queries:    db.New(pool),
		Signer:     signer,
		Family:     lookup,
		Log:        log,
		RefreshTTL: cfg.RefreshTTL,
	})

	// Expired refresh tokens are rows nobody will ever read again; sweeping them keeps the
	// unique index on token_hash from growing without bound.
	go sweepExpiredTokens(ctx, db.New(pool), log)

	srv := &http.Server{
		Addr: fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler: h2c.NewHandler(newMux(h, signer, pool, log), &http2.Server{
			// Without this an HTTP/2 connection with no open streams is kept forever; the
			// http.Server IdleTimeout above governs HTTP/1 only.
			IdleTimeout:          120 * time.Second,
			MaxConcurrentStreams: 250,
		}),
		ReadHeaderTimeout: 10 * time.Second,
		// No ReadTimeout or WriteTimeout on purpose. Both would have to be sized for the
		// slowest legitimate request — an 8 MiB recipe photo from a phone on a bad
		// connection — which makes them useless as a defence. ReadHeaderTimeout stops the
		// slowloris that matters, connect.WithReadMaxBytes bounds the body, and IdleTimeout
		// reclaims connections nobody is using.
		IdleTimeout: 120 * time.Second,
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

// jwksProvider is the public half of the signer, narrowed so the mux cannot reach the key.
type jwksProvider interface {
	JWKS() token.JWKS
}

func newMux(h authv1connect.AuthServiceHandler, keys jwksProvider, pool database.Pinger, log *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	// Every procedure on this service is public by definition: they are how a caller gets a
	// token in the first place, so there is no interceptor to apply.
	path, svc := authv1connect.NewAuthServiceHandler(h,
		connect.WithReadMaxBytes(maxRequestBytes),
		connect.WithInterceptors(rpc.Recover(log)),
	)
	mux.Handle(path, svc)

	mux.HandleFunc("GET /.well-known/jwks.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/jwk-set+json")
		// Cached, but briefly: verifiers refetch on an unknown kid, so a short TTL is all a
		// key rotation needs to propagate.
		w.Header().Set("Cache-Control", "public, max-age=300")
		if err := json.NewEncoder(w).Encode(keys.JWKS()); err != nil {
			slog.Error("encode jwks", slog.String("error", err.Error()))
		}
	})

	mux.HandleFunc("GET /healthz", database.HealthHandler(pool, 0))

	return mux
}

func sweepExpiredTokens(ctx context.Context, q *db.Queries, log *slog.Logger) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			deleted, err := q.DeleteExpiredRefreshTokens(ctx)
			if err != nil {
				log.WarnContext(ctx, "sweep expired refresh tokens",
					slog.String("error", err.Error()))
				continue
			}
			if deleted > 0 {
				log.InfoContext(ctx, "swept expired refresh tokens", slog.Int64("rows", deleted))
			}
		}
	}
}
