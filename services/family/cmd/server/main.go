// Command server runs the family service: Connect/JSON on :8080 for apps, gRPC on :9090 for
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
	"github.com/nnc/family-manager/sdk/go/family/v1/familyv1connect"
	"github.com/nnc/family-manager/services/family/db"
	"github.com/nnc/family-manager/services/family/internal/config"
	dbfs "github.com/nnc/family-manager/services/family/internal/db"
	"github.com/nnc/family-manager/services/family/internal/handler"
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

	log := logger.New(logger.Options{Service: "family", Level: cfg.LogLevel, JSON: cfg.LogJSON})
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

	bus, err := events.Connect(events.Config{URL: cfg.NATSURL, Name: "family"})
	if err != nil {
		// A broker outage must not stop the service answering reads and writes; events are
		// best-effort notifications, not the source of truth.
		log.Warn("events disabled", slog.String("error", err.Error()))
	} else {
		defer bus.Close()
		if err := bus.EnsureStream(ctx, "family"); err != nil {
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

	h := handler.New(handler.Options{
		Queries:       db.New(pool),
		Bus:           busOrNil(bus),
		Log:           log,
		InvitationTTL: cfg.InvitationTTL,
	})

	// Two listeners, because the two audiences are not equally trusted.
	//
	// Public (HTTPPort): apps. Every procedure requires a verified token, and
	// GetUserMembership is refused outright — it names a user id in the request, so on a
	// token-authenticated port it would let any signed-in user read anyone's household.
	//
	// Internal (GRPCPort): sibling services. No token — services/auth calls it *before* a
	// token exists. It must never be published to the host; compose leaves its port unmapped
	// so it is reachable only on the compose network.
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

// internalOnly lists procedures that must not be served on the token-authenticated public
// port. They take a subject id as an argument instead of reading it from the token, so
// exposing them to apps would be a horizontal privilege escalation.
var internalOnly = []string{
	familyv1connect.FamilyServiceGetUserMembershipProcedure,
	familyv1connect.FamilyServiceCheckMembershipProcedure,
}

func publicMux(h *handler.Handler, verifier *fmauth.Verifier, pool database.Pinger, log *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()
	path, svc := familyv1connect.NewFamilyServiceHandler(
		// Recover is outermost so a panic inside the auth interceptor is answered too.
		h,
		connect.WithReadMaxBytes(maxRequestBytes),
		connect.WithInterceptors(rpc.Recover(log), fmauth.Interceptor(verifier)),
	)
	// Registered under the service path, then shadowed per procedure: a more specific
	// pattern wins in ServeMux, so the block cannot be bypassed by casing or query strings.
	mux.Handle(path, svc)
	for _, procedure := range internalOnly {
		mux.HandleFunc(procedure, func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		})
	}
	mux.HandleFunc("GET /healthz", database.HealthHandler(pool, 0))
	return mux
}

func internalMux(h *handler.Handler, pool database.Pinger, log *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()
	// No interceptor: the caller is a sibling service on a private network, and auth calls
	// this before any token exists.
	path, svc := familyv1connect.NewFamilyServiceHandler(h,
		connect.WithReadMaxBytes(maxRequestBytes),
		connect.WithInterceptors(rpc.Recover(log)),
	)
	mux.Handle(path, svc)
	mux.HandleFunc("GET /healthz", database.HealthHandler(pool, 0))
	return mux
}

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
