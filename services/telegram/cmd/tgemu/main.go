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

	"github.com/nnc/family-manager/services/telegram/internal/tgemu"
)

func main() {
	addr := flag.String("addr", envOr("TGEMU_ADDR", ":8081"), "listen address")
	healthcheck := flag.Bool("healthcheck", false, "probe /_emu/healthz and exit")
	flag.Parse()

	if *healthcheck {
		os.Exit(probe(*addr))
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	srv := &http.Server{
		Addr:              *addr,
		Handler:           tgemu.New(log).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()

	log.Info("telegram emulator listening", "addr", *addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("serve", "error", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func probe(addr string) int {
	host := addr
	if len(host) > 0 && host[0] == ':' {
		host = "127.0.0.1" + host
	}
	client := &http.Client{Timeout: 3 * time.Second}
	res, err := client.Get(fmt.Sprintf("http://%s/_emu/healthz", host))
	if err != nil {
		return 1
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
