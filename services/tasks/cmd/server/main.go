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
	"github.com/nnc/family-manager/sdk/go/tasks/v1/tasksv1connect"
	"github.com/nnc/family-manager/services/tasks/internal/config"
	"github.com/nnc/family-manager/services/tasks/internal/calsync"
	"github.com/nnc/family-manager/services/tasks/internal/crypto"
	"github.com/nnc/family-manager/services/tasks/internal/family"
	"github.com/nnc/family-manager/services/tasks/internal/gcal"
	"github.com/nnc/family-manager/services/tasks/internal/handler"
	"github.com/nnc/family-manager/services/tasks/internal/members"
	"github.com/nnc/family-manager/services/tasks/internal/schedule"
	"github.com/nnc/family-manager/services/tasks/internal/store"
	dbfs "github.com/nnc/family-manager/services/tasks/internal/db"
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

	log := logger.New(logger.Options{Service: "tasks", Level: cfg.LogLevel, JSON: cfg.LogJSON})
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

	bus, err := events.Connect(events.Config{URL: cfg.NATSURL, Name: "tasks"})
	if err != nil {
		log.Warn("events disabled", slog.String("error", err.Error()))
	} else {
		defer bus.Close()
		if err := bus.EnsureStream(ctx, "tasks"); err != nil {
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

	familyPub := family.New(cfg.FamilyPublicAddr, cfg.FamilyAddr, 2*time.Second)
	familyInternal := family.New(cfg.FamilyAddr, cfg.FamilyAddr, 2*time.Second)

	st := store.New(pool)
	h := handler.New(handler.Options{
		Queries:      st.Queries(),
		Tx:           st,
		Bus:          busOrNil(bus),
		Family:       familyInternal,
		FamilyPublic: familyPub,
		Log:          log,
		Now:          time.Now,
	})

	if bus != nil {
		stopMembers, err := members.New(st.Queries(), log).Subscribe(ctx, bus)
		if err != nil {
			log.Warn("member removed subscriber disabled", slog.String("error", err.Error()))
		} else {
			defer stopMembers()
		}
	}

	reminderPoster := schedule.NewReminderPoster(st.Queries(), bus, time.Now, log)
	go schedule.NewReminders(reminderPoster, cfg.ReminderTick, log).Run(ctx)

	if bus != nil && cfg.GoogleClientID != "" {
		googleClient := gcal.NewHTTP(gcal.HTTPOptions{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
		})
		box, err := crypto.NewBox(cfg.TokenKey)
		if err != nil {
			log.Error("create crypto box", slog.String("error", err.Error()))
		} else {
			go calsync.NewPuller(calsync.PullerOptions{
				Queries: st.Queries(),
				Bus:     bus,
				Google:  googleClient,
				Box:     box,
				Tick:    cfg.SyncTick,
				Log:     log,
			}).Run(ctx)
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
	path, svc := tasksv1connect.NewTasksServiceHandler(
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
	path, svc := tasksv1connect.NewTasksServiceHandler(h,
		connect.WithReadMaxBytes(maxRequestBytes),
		connect.WithInterceptors(rpc.Recover(log), rpc.Observe(log)),
	)
	mux.Handle(path, svc)
	mux.Handle("GET "+rpc.MetricsPath, rpc.MetricsHandler())
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