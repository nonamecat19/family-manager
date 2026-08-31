// Command server runs the finance service: Connect/JSON on :8080 for apps, gRPC on :9090 for
// sibling services, both served by the same handler.
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

	bus, err := events.Connect(events.Config{URL: cfg.NATSURL, Name: "finance"})
	if err != nil {
		// A broker outage must not stop the ledger answering reads and writes; events are
		// best-effort notifications, not the source of truth.
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

	// The household roster belongs to services/family; finance projects it from
	// `family.member.*` so that every screen drawing a member does not fan out to a sibling.
	// With no broker the projection simply does not run — the ledger still answers, and the
	// members it knows about are the ones who have signed in.
	if bus != nil {
		stopProjection, perr := projection.NewMembers(st.Queries(), log).Subscribe(ctx, bus)
		if perr != nil {
			log.Warn("member projection disabled", slog.String("error", perr.Error()))
		} else {
			defer stopProjection()
		}
	}

	// Every finance procedure takes family_id and user_id from the token claims and none from
	// the request body, so there is no internal-only procedure here: the gRPC listener is the
	// same handler without the auth interceptor, for sibling services on the private network.
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
		// Recover is outermost so a panic inside the interceptors below it is answered too, and
		// Observe is above auth so a rejected token still gets an access line and an id.
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
	// No interceptor: the caller is a sibling service on a private network.
	path, svc := financev1connect.NewFinanceServiceHandler(h,
		connect.WithReadMaxBytes(maxRequestBytes),
		connect.WithInterceptors(rpc.Recover(log), rpc.Observe(log)),
	)
	mux.Handle(path, svc)
	mux.HandleFunc("GET /healthz", database.HealthHandler(pool, 0))
	return mux
}

// maxRequestBytes bounds a decoded request body. Nothing finance accepts is large — the
// biggest message is a page of transaction ids — so this is an order of magnitude above the
// worst legitimate request rather than sized for an upload.
const maxRequestBytes = 1 << 20 // 1 MiB

// One port for gRPC clients and Connect/JSON clients, with no TLS termination here.
func newServer(port string, mux *http.ServeMux) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           mux,
		Protocols:         h1AndUnencryptedH2(),
		ReadHeaderTimeout: 10 * time.Second,
		// No ReadTimeout or WriteTimeout on purpose: both would have to be sized for the
		// slowest legitimate request, which makes them useless as a defence.
		// ReadHeaderTimeout stops the slowloris that matters, connect.WithReadMaxBytes bounds
		// the body, and IdleTimeout reclaims connections nobody is using.
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

// h1AndUnencryptedH2 is the protocol set every listener here uses: HTTP/1.1 for Connect/JSON
// from the apps, and cleartext HTTP/2 for gRPC from sibling services, on one port.
//
// This replaces golang.org/x/net/http2/h2c, which is deprecated in favour of this field.
func h1AndUnencryptedH2() *http.Protocols {
	p := new(http.Protocols)
	p.SetHTTP1(true)
	p.SetUnencryptedHTTP2(true)
	return p
}
