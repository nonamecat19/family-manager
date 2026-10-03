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

	fmauth "github.com/nnc/family-manager/libs/go/auth"
	"github.com/nnc/family-manager/libs/go/database"
	"github.com/nnc/family-manager/libs/go/events"
	"github.com/nnc/family-manager/libs/go/logger"
	"github.com/nnc/family-manager/libs/go/rpc"
	"github.com/nnc/family-manager/sdk/go/notifications/v1/notificationsv1connect"
	"github.com/nnc/family-manager/services/notifications/internal/config"
	dbfs "github.com/nnc/family-manager/services/notifications/internal/db"
	"github.com/nnc/family-manager/services/notifications/internal/expo"
	"github.com/nnc/family-manager/services/notifications/internal/family"
	"github.com/nnc/family-manager/services/notifications/internal/handler"
	"github.com/nnc/family-manager/services/notifications/internal/notify"
	"github.com/nnc/family-manager/services/notifications/internal/store"
)

const maxRequestBytes = 64 << 10

var healthcheck = flag.Bool("healthcheck", false,
	"probe this container's own /healthz over loopback and exit")

var consumedDomains = []string{"family", "finance", "recipes"}

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

	log := logger.New(logger.Options{Service: "notifications", Level: cfg.LogLevel, JSON: cfg.LogJSON})
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

	verifier, err := fmauth.NewVerifier(fmauth.VerifierConfig{
		JWKSURL:  cfg.JWKSURL,
		Issuer:   cfg.Issuer,
		Audience: cfg.Audience,
	})
	if err != nil {
		return err
	}

	st := store.New(pool)
	notifier := notify.New(notify.Options{
		Queries:      st.Queries(),
		Sender:       expo.New(cfg.ExpoURL, cfg.ExpoAccessToken, 0),
		Families:     family.New(cfg.FamilyAddr, 0),
		Log:          log,
		ReceiptDelay: cfg.ReceiptDelay,
	})

	bus, err := events.Connect(events.Config{URL: cfg.NATSURL, Name: "notifications"})
	if err != nil {
		return err
	}
	defer bus.Close()
	for _, domain := range consumedDomains {
		if err := bus.EnsureStream(ctx, domain); err != nil {
			return err
		}
	}
	stopConsumers, err := notifier.Subscribe(ctx, bus)
	if err != nil {
		return err
	}
	defer stopConsumers()
	go notifier.RunReceipts(ctx, cfg.ReceiptInterval)

	h := handler.New(handler.Options{Queries: st.Queries(), Tx: st, Log: log})
	srv := newServer(cfg.HTTPPort, publicMux(h, verifier, pool, log))

	errc := make(chan error, 1)
	go func() {
		log.Info("listening", slog.String("listener", "public"), slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- fmt.Errorf("public listener: %w", err)
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

func publicMux(h *handler.Handler, verifier *fmauth.Verifier, pool database.Pinger, log *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()
	path, svc := notificationsv1connect.NewNotificationsServiceHandler(
		h,
		connect.WithReadMaxBytes(maxRequestBytes),
		connect.WithInterceptors(rpc.Recover(log), rpc.Observe(log), fmauth.Interceptor(verifier)),
	)
	mux.Handle(path, svc)
	mux.HandleFunc("GET /healthz", database.HealthHandler(pool, 0))
	return mux
}

func newServer(port string, mux *http.ServeMux) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           mux,
		Protocols:         h1AndUnencryptedH2(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

func h1AndUnencryptedH2() *http.Protocols {
	p := new(http.Protocols)
	p.SetHTTP1(true)
	p.SetUnencryptedHTTP2(true)
	return p
}
