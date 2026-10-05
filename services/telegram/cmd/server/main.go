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

	"github.com/nnc/family-manager/libs/go/database"
	"github.com/nnc/family-manager/libs/go/logger"
	"github.com/nnc/family-manager/libs/go/rpc"
	"github.com/nnc/family-manager/sdk/go/auth/v1/authv1connect"
	"github.com/nnc/family-manager/services/telegram/db"
	"github.com/nnc/family-manager/services/telegram/internal/bot"
	"github.com/nnc/family-manager/services/telegram/internal/bots/family"
	"github.com/nnc/family-manager/services/telegram/internal/bots/finance"
	"github.com/nnc/family-manager/services/telegram/internal/bots/notes"
	"github.com/nnc/family-manager/services/telegram/internal/bots/recipes"
	"github.com/nnc/family-manager/services/telegram/internal/config"
	dbfs "github.com/nnc/family-manager/services/telegram/internal/db"
	"github.com/nnc/family-manager/services/telegram/internal/preferences"
	"github.com/nnc/family-manager/services/telegram/internal/runtime"
	"github.com/nnc/family-manager/services/telegram/internal/secret"
	"github.com/nnc/family-manager/services/telegram/internal/session"
	"github.com/nnc/family-manager/services/telegram/internal/telegram"
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

	log := logger.New(logger.Options{Service: "telegram", Level: cfg.LogLevel, JSON: cfg.LogJSON})
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

	box, err := secret.NewBox(cfg.TokenKey)
	if err != nil {
		return err
	}

	queries := db.New(pool)
	calls := &http.Client{Timeout: cfg.CallTimeout}

	sessions := session.NewStore(session.Options{
		Queries: queries,
		Box:     box,
		Auth:    authv1connect.NewAuthServiceClient(calls, cfg.AuthAddr),
	})

	prefs := preferences.New(calls, cfg.FamilyAddr)

	bots, err := buildBots(ctx, cfg, sessions, queries, prefs, calls, log)
	if err != nil {
		return err
	}

	go sweepExpiredStates(ctx, queries, log)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", database.HealthHandler(pool, 0))
	mux.Handle("GET "+rpc.MetricsPath, rpc.MetricsHandler())

	var hooks *runtime.Webhook
	switch cfg.Mode {
	case config.ModeWebhook:
		hooks = runtime.NewWebhook(context.WithoutCancel(ctx), bots, cfg.WebhookSecret, log)
		hooks.Register(mux)
		if err := subscribe(ctx, cfg, bots, log); err != nil {
			return err
		}
	case config.ModePolling:
		for _, b := range bots {
			go poll(ctx, b, queries, cfg.PollTimeout, log)
		}
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		log.Info("listening",
			slog.String("addr", srv.Addr), slog.String("mode", string(cfg.Mode)))
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
		err := srv.Shutdown(shutdownCtx)
		if hooks != nil {
			hooks.Wait()
		}
		return err
	}
}

const stateSweepInterval = 15 * time.Minute

type stateSweeper interface {
	DeleteExpiredChatStates(ctx context.Context) (int64, error)
}

func sweepExpiredStates(ctx context.Context, q stateSweeper, log *slog.Logger) {
	ticker := time.NewTicker(stateSweepInterval)
	defer ticker.Stop()

	for {
		deleted, err := q.DeleteExpiredChatStates(ctx)
		switch {
		case err != nil:
			log.WarnContext(ctx, "sweep expired chat states", slog.String("error", err.Error()))
		case deleted > 0:
			log.InfoContext(ctx, "swept expired chat states", slog.Int64("rows", deleted))
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func buildBots(
	ctx context.Context,
	cfg *config.Config,
	sessions *session.Store,
	states bot.States,
	prefs bot.Preferences,
	calls *http.Client,
	log *slog.Logger,
) (map[string]runtime.Dispatcher, error) {
	bots := make(map[string]runtime.Dispatcher, len(cfg.Bots))

	for _, spec := range cfg.Bots {
		addr := cfg.AddrFor(spec.Name)
		if addr == "" {
			return nil, fmt.Errorf("config: TELEGRAM_%s_TOKEN is set but its service address is not",
				upper(spec.Name))
		}

		api, err := telegram.New(telegram.Options{
			Token:   spec.Token,
			APIBase: cfg.APIBase,
			Timeout: cfg.PollTimeout + 30*time.Second,
		})
		if err != nil {
			return nil, err
		}

		opts, err := optionsFor(spec.Name, calls, addr)
		if err != nil {
			return nil, err
		}

		opts.Name = spec.Name
		opts.API = api
		opts.Sessions = sessions
		opts.States = states
		opts.Prefs = prefs
		opts.Log = log
		opts.Timeout = cfg.CallTimeout + 5*time.Second

		b := bot.New(opts)
		if err := b.Connect(ctx); err != nil {
			return nil, err
		}
		bots[spec.Name] = b
	}

	return bots, nil
}

func optionsFor(name string, calls *http.Client, addr string) (bot.Options, error) {
	switch name {
	case "finance":
		return finance.Bot(calls, addr), nil
	case "recipes":
		return recipes.Bot(calls, addr), nil
	case "notes":
		return notes.Bot(calls, addr), nil
	case "family":
		return family.Bot(calls, addr), nil
	default:
		return bot.Options{}, fmt.Errorf("config: no bot named %q", name)
	}
}

func subscribe(
	ctx context.Context, cfg *config.Config, bots map[string]runtime.Dispatcher, log *slog.Logger,
) error {
	for name, b := range bots {
		hookURL := cfg.PublicURL + runtime.WebhookPath(name)
		if err := b.API().SetWebhook(ctx, hookURL, cfg.WebhookSecret); err != nil {
			return fmt.Errorf("set webhook for %s: %w", name, err)
		}
		log.Info("webhook registered", slog.String("bot", name), slog.String("url", hookURL))
	}
	return nil
}

func poll(
	ctx context.Context,
	b runtime.Dispatcher,
	offsets runtime.Offsets,
	timeout time.Duration,
	log *slog.Logger,
) {
	if err := runtime.Poll(ctx, b, offsets, timeout, log); err != nil {
		log.ErrorContext(ctx, "poller stopped",
			slog.String("bot", b.Name()), slog.String("error", err.Error()))
	}
}

func upper(s string) string {
	out := []rune(s)
	for i, r := range out {
		if r >= 'a' && r <= 'z' {
			out[i] = r - 32
		}
	}
	return string(out)
}
