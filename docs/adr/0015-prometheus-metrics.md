# ADR 0015 — Prometheus metrics from `rpc.Observe`, scraped off the internal listener

- Status: accepted
- Date: 2026-10-04
- Amends: [0006](0006-observability.md) (metrics, listed there as "add later"),
  [0008](0008-rpc-interceptors.md) (`Observe` now also counts)

## Context

[0008](0008-rpc-interceptors.md) left metrics as 0006's open question, and production runs at
`LOG_LEVEL=info`, where `Observe` writes an access line only for failures. So "how many
requests is finance taking, and how slow are they" has no answer at all in production, and
"is auth leaking goroutines" has none either. Both are counting questions; logs are the wrong
tool for them and traces are still unbuilt.

## Decision

**Prometheus, pulled, with `github.com/prometheus/client_golang`.**

- **RPC metrics live inside `rpc.Observe`**, not in a new interceptor. Every mux already
  installs `Observe`, so every Connect and gRPC procedure is counted without touching a single
  interceptor chain, and a service cannot forget to add it.
  - `rpc_server_requests_total{procedure, code}` — counter; `code` is the Connect code string
    `Observe` already logs (`ok`, `not_found`, `unauthenticated`, …).
  - `rpc_server_request_duration_seconds{procedure}` — histogram, buckets 5 ms to 10 s.
  - `procedure` is bounded: Connect answers an unknown path with 404 before any interceptor
    runs, so a client cannot mint label values.
- **Runtime metrics come from the default registry**, which already carries the Go runtime
  and process collectors (`go_*`, `process_*`). `rpc.MetricsHandler()` is `promhttp.Handler()`
  over that registry; `rpc.MetricsPath` is `/metrics`.
- **`/metrics` is mounted on the internal listener** (`:9090`) where a service has one —
  family, finance, recipes — because Caddy never proxies that port. auth and telegram have one
  listener and mount it there. It sits outside the auth interceptor; the scraper holds no token.
- **Caddy answers `/metrics` and `/actuator/prometheus` with 404** on every API host, so a
  single-listener service (auth today) does not publish its metrics to the internet.
- **notes** exposes Micrometer's `/actuator/prometheus` from Spring; it is scraped like the
  others. Metric names differ between the Go and Java sides; dashboards query them separately.
- **A `prometheus` container in `docker-compose.yml`**, config at
  `infra/prometheus/prometheus.yml`, 7-day retention, published on host `:9095` so it does not
  collide with a Go service running natively under `air` on `:9090`.

## Alternatives

- **A separate `rpc.Metrics` interceptor.** Same numbers, one more line in every chain, and a
  chain that omits it fails silently. `Observe` already holds the procedure, the code and the
  elapsed time; a second interceptor would recompute all three.
- **OTel metrics through the collector 0006 decided on.** Still the destination for traces, but
  the collector does not exist; Prometheus scraping needs nothing running in the services but an
  HTTP handler. Moving later means swapping the handler for an OTLP exporter behind the same
  names.
- **`/metrics` on the public listener everywhere.** One rule instead of two, but it leaves the
  Caddyfile as the only thing between the internet and the metrics; the internal listener does
  not depend on a proxy rule staying in place.
- **A dedicated metrics port per service.** A third listener in every service for something the
  internal listener already does privately.

## Consequences

- `libs/go/rpc` now depends on `client_golang`, and through it every service module does. The
  metrics are process-global (default registry); tests assert deltas, not absolute values.
- A service with both listeners reports one series per procedure covering both — the public
  and internal calls are not split. Add a `listener` label if that ever matters.
- auth's metrics are reachable on `auth:8080/metrics` inside the compose network; only the Caddy
  rule keeps them off the internet. Giving auth an internal listener would remove that reliance.
- Production compose has no Prometheus yet; the VPS memory budget decides that separately.
