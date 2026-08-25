# Tech stack

One row per layer: what, why, what it replaced, and **where it is pinned** — the file an agent
edits to change it. Decisions that could reasonably have gone the other way have an ADR in
[docs/adr/](adr/); read the ADR before proposing a swap.

## At a glance

| layer | choice | pinned in | ADR |
|---|---|---|---|
| monorepo orchestration | Turborepo + pnpm workspaces | `turbo.json`, `pnpm-workspace.yaml` | — |
| Go multi-module | `go.work` | `go.work` | — |
| task runner | just | `justfile` | — |
| app↔service transport | **ConnectRPC** (HTTP/1.1 + JSON) | `libs/proto/buf.gen.yaml` | [0001](adr/0001-connectrpc.md) |
| service↔service | gRPC | same protos, `:9090` | [0001](adr/0001-connectrpc.md) |
| contracts + codegen | protobuf + buf | `libs/proto/buf.yaml`, `buf.gen.yaml` | [0001](adr/0001-connectrpc.md) |
| events | NATS JetStream | `docker-compose.yml`, `libs/go/events` | [0003](adr/0003-nats-jetstream.md) |
| database | Postgres 16 + pgx/v5 | `docker-compose.yml`, service `go.mod` | — |
| SQL → Go | sqlc | `services/*/sqlc.yaml` | — |
| migrations | golang-migrate, numbered SQL | `services/*/internal/db/migrations` | — |
| object storage | MinIO (dev) · Cloudflare R2 (prod), one `minio-go` client | `docker-compose.yml`, `infra/` | [0007](adr/0007-object-storage.md) |
| Go config | viper, env-prefixed | `services/*/internal/config` | — |
| auth | ES256 JWT (15 min) + rotating refresh, JWKS | `services/auth`, `libs/go/auth` | [0005](adr/0005-auth.md) |
| password hashing | argon2id | `services/auth` only | [0005](adr/0005-auth.md) |
| Go logging | `log/slog`, JSON in prod, trace-correlated | `libs/go/logger` | [0006](adr/0006-observability.md) |
| tracing | OpenTelemetry → OTLP collector | `libs/go/logger`, `infra/` | [0006](adr/0006-observability.md) |
| CI | GitHub Actions running the verify node | `.github/workflows/verify.yml` | [0006](adr/0006-observability.md) |
| Go hot reload | air | `services/*/.air.toml` | — |
| app framework | Expo (managed) + expo-router | `apps/*/package.json` | — |
| app styling | NativeWind (Tailwind for RN) | `packages/theme`, `packages/config` | [0002](adr/0002-nativewind.md) |
| app data layer | TanStack Query over `packages/api` | `packages/api` | — |
| app secrets | expo-secure-store | `packages/auth` | — |
| deployment | Docker Compose on a VPS + Caddy | `infra/` | [0004](adr/0004-compose-vps.md) |
| language versions | Go 1.23 · Node ≥22 · pnpm 11 | `go.work`, `package.json` | — |
| Go module paths | `github.com/nnc/family-manager/{services,libs/go,sdk/go}/<name>` | each `go.mod` | — |
| TS package names | `@fm/<name>` | each `package.json` | — |

## Go services

```
gRPC (:9090, service↔service)  ─┐
                                ├─→ internal/handler ─→ sqlc db/ ─→ Postgres
Connect (:8080, apps, JSON)    ─┘        │
                                         └─→ libs/go/events ─→ NATS JetStream
```

- **pgx/v5** directly, no ORM. Queries are written as SQL in `internal/db/queries/*.sql` and
  compiled by **sqlc** into `db/`. Hand-writing a query struct instead of a `.sql` file is a
  review reject — the graph extracts `PERSISTS_TO` edges from those SQL files, so invisible
  queries mean an incomplete graph.
- **viper** config, env-prefixed per service (`AUTH_`, `FAMILY_`…). Secrets have no defaults —
  a missing secret must crash at boot, not silently default.
- **slog** from `libs/go/logger`: text handler in dev, JSON in prod, request id + user id in
  the context. No `fmt.Println`, no logrus/zap — one logger repo-wide.
- **air** for hot reload in dev; `just up` supplies Postgres/MinIO/NATS.
- Shared concerns are libraries, not copies: `libs/go/auth` (token verify middleware),
  `libs/go/database` (pool + migration runner), `libs/go/events` (JetStream publish/subscribe),
  `libs/go/logger`. If two services need the same helper, it moves to `libs/go/*` on the second
  use, not the third.

## Contracts

Every cross-boundary call is a protobuf method in `libs/proto/<domain>/v1/`. buf generates:

| output | plugin | consumed by |
|---|---|---|
| `sdk/go/<domain>/v1/*.pb.go` | protoc-gen-go | services |
| `sdk/go/<domain>/v1/<d>v1connect/*.connect.go` | protoc-gen-connect-go | service servers + Go clients |
| `sdk/typescript/<domain>/v1/*_pb.ts` | protoc-gen-es (v2) | `packages/api` via `createClient` |

connect-es v2 needs no separate client plugin: `protoc-gen-es` emits the service descriptor and
`@connectrpc/connect` builds the client from it. Generated code is never hand-edited.
`buf lint` and `buf breaking` run in the verify node — a breaking change fails the build before
it reaches the human gate.

## Expo apps

- **expo-router** file-based navigation; screens live only in `apps/*`.
- **NativeWind**: Tailwind classes on React Native. The token source is a Tailwind preset in
  `packages/theme`, imported by every app's `tailwind.config.js` via `packages/config`. No
  per-app color or spacing literals.
- **TanStack Query** wraps every Connect call inside `packages/api` (query keys, cache,
  retry, optimistic updates). Apps never import `sdk/typescript` or call `fetch` directly.
- **expo-secure-store** holds tokens; `packages/auth` owns refresh and the auth state machine.
  Tokens never touch AsyncStorage or Zustand.
- Metro must watch the workspace root or shared packages will not hot-reload — that config
  lives in `packages/config`, not copy-pasted per app.

## Auth in one paragraph

`services/auth` is the only issuer. Apps get a 15-minute **ES256 access JWT** plus an opaque
**refresh token** (hashed at rest, rotated on every use, chain-revoked on reuse), stored in
expo-secure-store by `packages/auth`. Every other service verifies the JWT **locally** with
public keys cached from `services/auth`'s `/.well-known/jwks.json` — no per-request call to the
auth service, no shared signing secret. Middleware lives in `libs/go/auth` and nowhere else.
Details and key rotation: [ADR 0005](adr/0005-auth.md).

## Observability

`libs/go/logger` wires slog + OpenTelemetry together: one server span per Connect/gRPC request,
`traceparent` propagated across service calls **and JetStream messages**, pgx instrumented, and
`trace_id`/`span_id` stamped on every log record. Exporter is OTLP; unset endpoint = no export,
ids still generated. [ADR 0006](adr/0006-observability.md).

## Local stack

```sh
just tools     # buf, sqlc, air, golang-migrate into $GOBIN (once)
just up        # postgres:5432  minio:9000/9001  nats:4222 (monitor :8222)
just install
just dev-apps
cd services/<name> && air
```

## Testing

| side | tools |
|---|---|
| Go unit | stdlib `testing` + testify assertions |
| Go integration | testcontainers-go (Postgres, NATS) — never the shared dev DB |
| TS unit | Jest + React Native Testing Library |
| contract | `buf lint`, `buf breaking` against the `main` branch |

## Versions and upgrades

Language and tool versions are pinned in `go.work`, `package.json` (`packageManager`,
`engines`) and `docker-compose.yml` image tags. Bumping any of them is a change of its own —
`packages/config` and shared `libs/go/*` bumps go through the human gate because their blast
radius is the whole repo (`just impact pkg:@fm/config`).

## Explicitly not in the stack

| rejected | why |
|---|---|
| GraphQL | one contract language is enough; protobuf already generates both sides |
| an ORM (GORM/ent) | sqlc gives typed queries without hiding SQL, and keeps the graph's `PERSISTS_TO` edges extractable |
| Redis | JetStream covers queues/streams; Postgres covers everything else at this scale |
| a monorepo-wide shared Go module | one module per service/lib keeps `go.work` boundaries real |
| Flutter | the existing `*-android` apps are being retired in favour of Expo (see docs/architecture.md) |
