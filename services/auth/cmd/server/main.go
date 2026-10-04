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

	fmauth "github.com/nnc/family-manager/libs/go/auth"
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
	"github.com/nnc/family-manager/services/auth/internal/ratelimit"
	"github.com/nnc/family-manager/services/auth/internal/store"
	"github.com/nnc/family-manager/services/auth/internal/throttle"
	"github.com/nnc/family-manager/services/auth/internal/token"
)

const maxRequestBytes = 1 << 20

const sweepInterval = time.Hour

var healthcheck = flag.Bool("healthcheck", false,
	"probe this container's own /healthz over loopback and exit")

func main() {
	flag.Parse()

	if *healthcheck {
		if err := probe(); err != nil {
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
		Queries:  db.New(pool),
		Tx:       store.New(pool),
		Signer:   signer,
		Family:   lookup,
		Log:      log,
		HashGate: password.NewGate(cfg.HashConcurrency),
		Throttle: throttle.New(throttle.Params{
			Threshold: cfg.LoginFailureThreshold,
			Base:      cfg.LoginLockoutBase,
			Max:       cfg.LoginLockoutMax,
		}, nil),
		Decisions:  throttle.New(throttle.DefaultParams(), nil),
		Starts:     ratelimit.New(cfg.DeviceLoginPerIP, cfg.DeviceLoginWindow, nil),
		MaxPending: cfg.DeviceLoginMaxActive,
		RefreshTTL: cfg.RefreshTTL,
	})

	go sweepExpiredTokens(ctx, db.New(pool), log, sweepInterval)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:           newMux(h, signer, pool, log),
		Protocols:         h1AndUnencryptedH2(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
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

type jwksProvider interface {
	JWKS() token.JWKS
}

type sessionVerifier interface {
	jwksProvider
	fmauth.TokenVerifier
}

var publicProcedures = []string{
	authv1connect.AuthServiceLoginProcedure,
	authv1connect.AuthServiceRegisterProcedure,
	authv1connect.AuthServiceRefreshProcedure,
	authv1connect.AuthServiceLogoutProcedure,
	authv1connect.AuthServiceRedeemLinkTokenProcedure,
	authv1connect.AuthServiceStartDeviceLoginProcedure,
	authv1connect.AuthServicePollDeviceLoginProcedure,
}

func newMux(h authv1connect.AuthServiceHandler, keys sessionVerifier, pool database.Pinger, log *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	path, svc := authv1connect.NewAuthServiceHandler(h,
		connect.WithReadMaxBytes(maxRequestBytes),
		connect.WithInterceptors(
			rpc.Recover(log),
			rpc.Observe(log),
			fmauth.Interceptor(keys, publicProcedures...),
		),
	)
	mux.Handle(path, svc)

	mux.HandleFunc("GET /.well-known/jwks.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/jwk-set+json")
		w.Header().Set("Cache-Control", "public, max-age=300")
		if err := json.NewEncoder(w).Encode(keys.JWKS()); err != nil {
			slog.Error("encode jwks", slog.String("error", err.Error()))
		}
	})

	mux.HandleFunc("GET /healthz", database.HealthHandler(pool, 0))
	mux.Handle("GET "+rpc.MetricsPath, rpc.MetricsHandler())

	return mux
}

type sweeper interface {
	DeleteExpiredRefreshTokens(ctx context.Context) (int64, error)
	DeleteExpiredLinkTokens(ctx context.Context) (int64, error)
	DeleteOrphanChains(ctx context.Context) (int64, error)
	DeleteExpiredLoginGrants(ctx context.Context) (int64, error)
}

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
	} else if deleted > 0 {
		log.InfoContext(ctx, "swept expired refresh tokens", slog.Int64("rows", deleted))
	}

	deleted, err = q.DeleteExpiredLinkTokens(ctx)
	if err != nil {
		log.WarnContext(ctx, "sweep expired link tokens", slog.String("error", err.Error()))
	} else if deleted > 0 {
		log.InfoContext(ctx, "swept expired link tokens", slog.Int64("rows", deleted))
	}

	deleted, err = q.DeleteOrphanChains(ctx)
	if err != nil {
		log.WarnContext(ctx, "sweep orphan chains", slog.String("error", err.Error()))
	} else if deleted > 0 {
		log.InfoContext(ctx, "swept orphan chains", slog.Int64("rows", deleted))
	}

	deleted, err = q.DeleteExpiredLoginGrants(ctx)
	if err != nil {
		log.WarnContext(ctx, "sweep expired login grants", slog.String("error", err.Error()))
	} else if deleted > 0 {
		log.InfoContext(ctx, "swept expired login grants", slog.Int64("rows", deleted))
	}
}

func h1AndUnencryptedH2() *http.Protocols {
	p := new(http.Protocols)
	p.SetHTTP1(true)
	p.SetUnencryptedHTTP2(true)
	return p
}
