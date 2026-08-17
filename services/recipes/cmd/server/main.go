// Command server runs the recipes service: Connect/JSON on :8080 for apps, gRPC on :9090 for
// sibling services, both served by the same handler.
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
	"github.com/nnc/family-manager/libs/go/rpc"
	"github.com/nnc/family-manager/libs/go/storage"
	"github.com/nnc/family-manager/sdk/go/recipes/v1/recipesv1connect"
	"github.com/nnc/family-manager/services/recipes/db"
	"github.com/nnc/family-manager/services/recipes/internal/config"
	dbfs "github.com/nnc/family-manager/services/recipes/internal/db"
	"github.com/nnc/family-manager/services/recipes/internal/handler"
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

	log := logger.New(logger.Options{Service: "recipes", Level: cfg.LogLevel, JSON: cfg.LogJSON})
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

	bus, err := events.Connect(events.Config{URL: cfg.NATSURL, Name: "recipes"})
	if err != nil {
		// A broker outage must not stop the service answering reads and writes; events are
		// best-effort notifications, not the source of truth.
		log.Warn("events disabled", slog.String("error", err.Error()))
	} else {
		defer bus.Close()
		if err := bus.EnsureStream(ctx, "recipes"); err != nil {
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

	images, err := imagesOrNil(ctx, cfg, log)
	if err != nil {
		return err
	}

	h := handler.New(handler.Options{
		Queries:     db.New(pool),
		Bus:         busOrNil(bus),
		Images:      images,
		ImageBucket: cfg.StorageBucket,
		Log:         log,
	})

	// Recipes has no internal-only procedures (unlike family): every procedure takes the
	// family_id from the token claim, none from the request body. So the public listener is
	// the only one — gRPC on :9090 is the same handler without the auth interceptor, for
	// sibling services that may want to query a recipe by id (e.g. notifications).
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
	path, svc := recipesv1connect.NewRecipesServiceHandler(
		// Recover is outermost so a panic inside the auth interceptor is answered too.
		h,
		connect.WithReadMaxBytes(maxRequestBytes),
		connect.WithInterceptors(rpc.Recover(log), fmauth.Interceptor(verifier)),
	)
	mux.Handle(path, svc)
	mux.HandleFunc("GET /healthz", database.HealthHandler(pool, 0))
	return mux
}

func internalMux(h *handler.Handler, pool database.Pinger, log *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()
	// No interceptor: the caller is a sibling service on a private network.
	path, svc := recipesv1connect.NewRecipesServiceHandler(h,
		connect.WithReadMaxBytes(maxRequestBytes),
		connect.WithInterceptors(rpc.Recover(log)),
	)
	mux.Handle(path, svc)
	mux.HandleFunc("GET /healthz", database.HealthHandler(pool, 0))
	return mux
}

// maxRequestBytes bounds a decoded request body. UploadRecipeImage caps the photo itself at
// 8 MiB, but that check runs after the whole body is in memory; this one refuses the read.
// The headroom covers base64 expansion in the JSON encoding plus the rest of the message.
const maxRequestBytes = 16 << 20 // 16 MiB

// h2c so gRPC clients and Connect/JSON clients share one port without TLS termination here.
func newServer(port string, mux *http.ServeMux) *http.Server {
	return &http.Server{
		Addr: fmt.Sprintf(":%s", port),
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
}

// busOrNil keeps the handler's Bus field nil when the broker is unavailable, which makes it
// fall back to its no-op publisher.
func busOrNil(bus *events.Bus) handler.EventBus {
	if bus == nil {
		return nil
	}
	return bus
}

// imagesOrNil builds the object storage client when configured. A missing STORAGE_ENDPOINT
// means image upload runs in degraded mode (UploadRecipeImage errors, everything else works)
// rather than blocking boot — same tradeoff as an unreachable NATS.
func imagesOrNil(ctx context.Context, cfg *config.Config, log *slog.Logger) (handler.ImageStore, error) {
	if cfg.StorageEndpoint == "" {
		log.Warn("image storage disabled: RECIPES_STORAGE_ENDPOINT not set")
		return nil, nil
	}
	client, err := storage.New(storage.Config{
		Provider:  storage.Provider(cfg.StorageProvider),
		Endpoint:  cfg.StorageEndpoint,
		AccessKey: cfg.StorageAccessKey,
		SecretKey: cfg.StorageSecretKey,
		UseSSL:    cfg.StorageUseSSL,
		PublicURL: cfg.StoragePublicURL,
	})
	if err != nil {
		return nil, err
	}
	// A no-op on R2, where the bucket is provisioned out of band.
	if err := client.EnsureBucket(ctx, cfg.StorageBucket); err != nil {
		return nil, err
	}
	return client, nil
}
