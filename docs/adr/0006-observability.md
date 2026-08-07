# ADR 0006 — OpenTelemetry traces correlated with slog, CI runs the verify node

- Status: accepted
- Date: 2026-08-06

## Context

With four services, an event bus and three apps, "it is slow" and "the invite never arrived" are
not answerable from one log file. And nothing currently enforces the workflow in
`docs/workflow.md` — the verify node only runs when an agent remembers to run it.

## Decision

**Tracing.** OpenTelemetry, wired in `libs/go/logger`:

- Connect and gRPC interceptors start a server span per request; outgoing clients propagate
  `traceparent` (W3C).
- JetStream publishes carry `traceparent` in message headers, so an event consumer's span is a
  child of the request that produced it — the whole point, since events are where causality is
  otherwise lost.
- pgx is instrumented via `otelpgx`, so slow queries appear in the trace.
- Every `slog` record gets `trace_id` and `span_id` from the context. One logger, one interceptor
  chain, no per-service wiring.
- Exporter: OTLP to a collector defined in `infra/`. Locally, `OTEL_EXPORTER_OTLP_ENDPOINT`
  unset = no exporter, spans still create ids for log correlation.

**CI.** GitHub Actions runs the verify node on every push and PR:

| job | command |
|---|---|
| go | `just check-go` |
| ts | `just check-ts` |
| contracts | `just proto-check` (buf lint + breaking vs base) |
| graph | `just graph-check` — fails when `docs/graph/graph.json` is stale |

The graph job is what keeps the knowledge graph honest: a PR that adds a service, contract or
table but does not regenerate the graph fails, so no agent can reason from a stale graph.

## Alternatives

- **Logs only** — cheapest, but cross-service and event causality is reconstructed by hand.
- **Prometheus metrics first** — answers "how many" not "why this request"; add later, on top of
  the same collector.
- **A hosted APM SDK** — one more vendor client in every service; OTLP keeps the backend swappable.

## Consequences

- Trace context must be threaded through `context.Context` everywhere — a goroutine started with
  a bare context loses the trace. Review point.
- CI needs a Go toolchain, pnpm, buf and Node; buf runs as an action, not a repo dependency.
- `just graph-check` makes `docs/graph/graph.json` a committed artifact. Regenerating it is part
  of every structural change, not a chore.
- Sampling: 100% locally, head sampling at 10% in prod with errors always sampled.
