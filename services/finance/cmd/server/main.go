// Command server runs the finance service: Connect/JSON on :8080 for apps, gRPC on the same
// port via h2c for sibling services.
package main

import (
	"context"
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

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database"
	"github.com/nnc/family-manager/libs/go/events"
	"github.com/nnc/family-manager/libs/go/logger"
	"github.com/nnc/family-manager/libs/go/rpc"
	"github.com/nnc/family-manager/sdk/go/finance/v1/financev1connect"
	"github.com/nnc/family-manager/services/finance/db"
	"github.com/nnc/family-manager/services/finance/internal/config"
	dbfs "github.com/nnc/family-manager/services/finance/internal/db"
	"github.com/nnc/family-manager/services/finance/internal/handler"
)

// maxRequestBytes bounds a decoded request body. Nothing this service accepts is large — the
// biggest message is a household with its members — so the cap is small enough that an
// oversize body is refused during the read rather than after it is buffered.
const maxRequestBytes = 1 << 20 // 1 MiB

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
		// Recover is outermost so a panic inside the interceptors below it is answered too, and
		// Observe is above auth so a rejected token still gets an access line and an id.
		h,
		connect.WithReadMaxBytes(maxRequestBytes),
		connect.WithInterceptors(rpc.Recover(log), rpc.Observe(log), fmauth.Interceptor(verifier)),
	)
	mux.Handle(path, svc)
	mux.HandleFunc("GET /healthz", database.HealthHandler(pool, 0))

	// h2c so gRPC (sibling services) and Connect/JSON (apps) share one port.
	srv := &http.Server{
		Addr: fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler: h2c.NewHandler(mux, &http2.Server{
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
