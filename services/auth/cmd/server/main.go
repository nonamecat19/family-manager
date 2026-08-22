// Command server runs the auth service: auth.v1.AuthService over Connect, plus the JWKS
// endpoint every other service fetches to verify the tokens minted here.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
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
	"github.com/nnc/family-manager/services/auth/internal/password"
	"github.com/nnc/family-manager/services/auth/internal/token"
)

// maxRequestBytes bounds a decoded request body. Nothing this service accepts is large — the
// biggest message is a household with its members — so the cap is small enough that an
// oversize body is refused during the read rather than after it is buffered.
const maxRequestBytes = 1 << 20 // 1 MiB

// sweepInterval is how often expired refresh tokens are deleted.
const sweepInterval = time.Hour

// healthcheck makes the service binary its own container healthcheck. The distroless image
// ships no shell and no wget, so this is the only executable available to probe with.
var healthcheck = flag.Bool("healthcheck", false,
	"probe this container's own /healthz over loopback and exit")

func main() {
	flag.Parse()

	if *healthcheck {
		if err := probe(); err != nil {
			// stderr, not the service logger: this process is a probe, and its output is read
			// by `docker inspect`, not collected as service logs.
			fmt.Fprintln(os.Stderr, "healthcheck:", err)
			os.Exit(1)
		}
		return
	}

	if err := run(); err != nil {
		slog.Error("fatal", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

// probe reads the port from the same config the server binds, so the two cannot disagree.
func probe() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	return rpc.Probe(cfg.HTTPPort, 0)
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
		HashGate:   password.NewGate(cfg.HashConcurrency),
		RefreshTTL: cfg.RefreshTTL,
	})

	// Expired refresh tokens are rows nobody will ever read again; sweeping them keeps the
	// unique index on token_hash from growing without bound.
	go sweepExpiredTokens(ctx, db.New(pool), log, sweepInterval)

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
		connect.WithInterceptors(rpc.Recover(log), rpc.Observe(log)),
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

// sweeper is the one query the sweep needs, named so the loop can be tested without a
// database behind it.
type sweeper interface {
	DeleteExpiredRefreshTokens(ctx context.Context) (int64, error)
}

// sweepExpiredTokens deletes expired refresh tokens every interval, and once on entry.
//
// The immediate pass is the point of the change: the loop used to wait a full interval before
// its first run, so a service that restarts more often than that — a deploy, a crash loop, a
// VPS reboot — never swept at all, and the backlog it was written to prevent grew anyway.
func sweepExpiredTokens(ctx context.Context, q sweeper, log *slog.Logger, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	sweepOnce(ctx, q, log)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweepOnce(ctx, q, log)
		}
	}
}

func sweepOnce(ctx context.Context, q sweeper, log *slog.Logger) {
	deleted, err := q.DeleteExpiredRefreshTokens(ctx)
	if err != nil {
		log.WarnContext(ctx, "sweep expired refresh tokens", slog.String("error", err.Error()))
		return
	}
	if deleted > 0 {
		log.InfoContext(ctx, "swept expired refresh tokens", slog.Int64("rows", deleted))
	}
}
