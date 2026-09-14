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
	"github.com/nnc/family-manager/sdk/go/finance/v1/financev1connect"
	"github.com/nnc/family-manager/services/finance/internal/config"
	dbfs "github.com/nnc/family-manager/services/finance/internal/db"
	"github.com/nnc/family-manager/services/finance/internal/handler"
	"github.com/nnc/family-manager/services/finance/internal/projection"
	"github.com/nnc/family-manager/services/finance/internal/store"
)

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

	bus, err := events.Connect(events.Config{URL: cfg.NATSURL, Name: "finance"})
	if err != nil {
		log.Warn("events disabled", slog.String("error", err.Error()))
	} else {
		defer bus.Close()
		if err := bus.EnsureStream(ctx, "finance"); err != nil {
			log.Warn("ensure stream failed", slog.String("error", err.Error()))
		}
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
	h := handler.New(handler.Options{
		Queries:         st.Queries(),
		Tx:              st,
		Bus:             busOrNil(bus),
		DefaultCurrency: cfg.BaseCurrency,
		DefaultTimezone: cfg.Timezone,
		Log:             log,
	})

	if bus != nil {
		stopProjection, perr := projection.NewMembers(st.Queries(), log).Subscribe(ctx, bus)
		if perr != nil {
			log.Warn("member projection disabled", slog.String("error", perr.Error()))
		} else {
			defer stopProjection()
		}
	}

	publicSrv := newServer(cfg.HTTPPort, publicMux(h, verifier, pool, log))
	internalSrv := newServer(cfg.GRPCPort, internalMux(h, pool, log))

	errc := make(chan error, 2)
	serve := func(srv *http.Server, name string) {
		log.Info("listening", slog.String("listener", name), slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- fmt.Errorf("%s listener: %w", name, err)
		}
	}
	go serve(publicSrv, "public")
	go serve(internalSrv, "internal")

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := publicSrv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return internalSrv.Shutdown(shutdownCtx)
	}
}

func publicMux(h *handler.Handler, verifier *fmauth.Verifier, pool database.Pinger, log *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()
	path, svc := financev1connect.NewFinanceServiceHandler(
		h,
		connect.WithReadMaxBytes(maxRequestBytes),
		connect.WithInterceptors(rpc.Recover(log), rpc.Observe(log), fmauth.Interceptor(verifier)),
	)
	mux.Handle(path, svc)
	mux.HandleFunc("GET /healthz", database.HealthHandler(pool, 0))
	return mux
}

func internalMux(h *handler.Handler, pool database.Pinger, log *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()
	path, svc := financev1connect.NewFinanceServiceHandler(h,
		connect.WithReadMaxBytes(maxRequestBytes),
		connect.WithInterceptors(rpc.Recover(log), rpc.Observe(log)),
	)
	mux.Handle(path, svc)
	mux.HandleFunc("GET /healthz", database.HealthHandler(pool, 0))
	return mux
}

const maxRequestBytes = 1 << 20

func newServer(port string, mux *http.ServeMux) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           mux,
		Protocols:         h1AndUnencryptedH2(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

func busOrNil(bus *events.Bus) handler.EventBus {
	if bus == nil {
		return nil
	}
	return bus
}

func h1AndUnencryptedH2() *http.Protocols {
	p := new(http.Protocols)
	p.SetHTTP1(true)
	p.SetUnencryptedHTTP2(true)
	return p
}
