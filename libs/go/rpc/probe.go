package rpc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// DefaultProbeTimeout bounds a container health probe. It is shorter than the health handler's
// own database timeout on purpose: a probe that outlives the check it is calling reports
// "timeout" for what the service would have answered "unhealthy".
const DefaultProbeTimeout = 3 * time.Second

// Probe requests /healthz on the given port over loopback and reports whether the service
// considers itself healthy.
//
// It exists so a distroless image can have a container healthcheck. Those images ship no
// shell, no wget and no curl — which is the point of them — so every service in this repo has
// been running with no healthcheck at all, and compose's `depends_on: service_healthy` has had
// nothing to wait on. The service binary is the one executable in the image, so it probes
// itself: `/server -healthcheck`.
//
// Loopback only. A probe that can be aimed at another host is a request-forgery primitive
// sitting inside every container.
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
