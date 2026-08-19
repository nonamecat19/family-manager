# ADR 0008 — A shared interceptor chain in `libs/go/rpc`, request ids before traces

- Status: accepted
- Date: 2026-08-19
- Amends: [0006](0006-observability.md) (the correlation story, narrowed to what runs)

## Context

[0006](0006-observability.md) decided on OpenTelemetry: server spans per request,
`traceparent` on outgoing calls and on JetStream messages, `trace_id` on every slog record.
None of it is built. What was built is the slog half — `logger.WithRequestID`,
`logger.WithUserID` and a handler that copies both onto every record — and until this week
nothing in the repo called any of it. The feature existed, was tested, and had never run.

Meanwhile each service had grown its own copy of three things that are not per-service
decisions: the internal-error helper (which wrapped pgx's error into the response body), the
absence of any panic handling, and the absence of any access log.

Answering "the invite never arrived" still needed four `docker logs` and a clock.

## Decision

One module, `libs/go/rpc`, holding the Connect-side concerns every service repeats, installed
in this order on every mux — public and internal:

| position | interceptor | why here |
|---|---|---|
| outermost | `Recover` | a panic anywhere below it, including in auth, becomes a response instead of a dead connection |
| middle | `Observe` | a call rejected for a bad token still gets an id and an access line |
| innermost | `auth.Interceptor` | everything above it runs whether or not the caller is known |

- **Request ids, not trace ids, for now.** `Observe` takes the caller's `X-Request-Id` or mints
  one, puts it on the context via `logger.WithRequestID`, echoes it on the response, and
  `ForwardRequestID` copies it onto outgoing calls. One id per request, across hops.
- **`rpc.Internal` is the only way to return an unexpected failure.** The cause goes to the log;
  the caller gets `internal error (ref <request id>)`. The reference a user reads off their
  screen is the string that finds the request in every service's logs.
- **`auth.Interceptor` stamps `user_id`** so a line can say who, not only what.

An id from a caller is bounded at 64 bytes and restricted to printable ASCII before it is used.
It goes verbatim into structured log lines, and a request id containing a newline is a way for
a client to forge log entries.

## Alternatives

- **Build OTel now.** Still the right destination, and this does not displace it: `Observe` is
  where the span starts when it arrives, and `traceparent` supersedes `X-Request-Id` for the
  cross-hop job. Doing it first would have left the pgx-error leak and the connection-killing
  panics in place for as long as the collector took to stand up.
- **Keep the helpers per-service.** They had already drifted — finance's `notFoundOr` leaked
  where auth's did not — and four copies is four places to fix the same thing.
- **Log the cause to the caller in dev only.** An environment-dependent response body is a
  behaviour difference nobody tests, in the direction where the untested branch is the one that
  faces the internet.

## Consequences

- Every unexpected error is now opaque to the caller. Debugging from a bug report means asking
  for the reference; without it there is nothing to grep. Bug templates should ask for it.
- Access lines are logged at DEBUG for success and WARN for failure, so production at
  `LOG_LEVEL=info` records failures and not throughput. Metrics are still 0006's open question.
- `libs/go/rpc` depends on `libs/go/logger`; nothing may depend on `libs/go/rpc` from a place
  that is not a service edge, or the interceptor chain stops being the only wiring point.
- When OTel lands, `Observe` gains the span and keeps the id: an id that survives a backend
  swap is worth more than one tied to a collector being up.
