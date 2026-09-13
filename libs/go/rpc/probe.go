package rpc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

const DefaultProbeTimeout = 3 * time.Second

func Probe(port string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = DefaultProbeTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	url := "http://" + net.JoinHostPort("127.0.0.1", port) + "/healthz"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("probe: build request: %w", err)
	}

	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return fmt.Errorf("probe: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("probe: %s returned %d", url, resp.StatusCode)
	}
	return nil
}
