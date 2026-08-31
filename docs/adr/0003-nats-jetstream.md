# ADR 0003 — NATS JetStream as the event backbone

- Status: accepted
- Date: 2026-08-06

## Context

`notifications` must react to things that happen in `family`, `shopping` and `recipes` without
those services knowing it exists. Synchronous gRPC would make every writer depend on the
notifier and fail when it is down.

## Decision

NATS JetStream, one container in `docker-compose.yml`, wrapped by `libs/go/events`. Services
publish domain events and subscribe with durable consumers. Subjects follow
`<domain>.<entity>.<verb>` — `family.member.invited`, `shopping.list.completed`. Event payloads
are protobuf messages defined in `libs/proto/<domain>/v1/events.proto`, so events are contracts
too and `just impact` sees them.

Publishing rule: a service publishes **after** its transaction commits. Where losing an event
would be a correctness bug (payments, invites), use a transactional outbox table drained by a
publisher goroutine — at-least-once, with idempotent consumers.

## Alternatives

- **Postgres outbox + LISTEN/NOTIFY only** — no new infra, but fan-out, replay and durable
  consumer groups all become hand-written.
- **Redis Streams** — similar shape, weaker delivery guarantees, and we would still add
  Postgres-side machinery for reliability.
- **Sync gRPC only** — couples writers to the notifier's availability.

## Consequences

- One more container in dev and in `infra/`; JetStream needs a persistent volume.
- Consumers must be idempotent — at-least-once, not exactly-once.
- Integration tests use testcontainers with a real NATS, never a mock bus.
- Every new subject is a contract change: define the payload in `events.proto`, run
  `just proto`, and check `just impact` before renaming a subject.
