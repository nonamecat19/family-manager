# AGENTS.md — family-manager

Monorepo of Expo apps + Go microservices, driven by **graph engineering**: a knowledge graph
of the repo (what exists and what depends on what) and a task graph (how work is executed).
Both are described here. Read this file before touching anything.

- Task-graph detail, gates, stop rule → [docs/workflow.md](docs/workflow.md)
- Repo ontology (entity/relation types) → [docs/ontology.yaml](docs/ontology.yaml)
- Target layout + migration state → [docs/architecture.md](docs/architecture.md)
- Tech stack, layer by layer → [docs/stack.md](docs/stack.md); decisions → [docs/adr/](docs/adr/)
- MCP servers and why → [docs/mcp.md](docs/mcp.md)
- Generated graph → `docs/graph/graph.json`, `docs/graph/graph.mmd` (`just graph`)

## Stack

Turborepo + pnpm (TypeScript/Expo) · go.work (Go) · just (task runner) · buf + sqlc (codegen)
· Postgres + MinIO + NATS JetStream via docker compose.

Apps talk to services over **ConnectRPC** (HTTP/1.1 + JSON, `:8080`); services talk to each
other over gRPC and publish domain events to JetStream. Expo apps use expo-router, NativeWind
and TanStack Query. Deployment is Compose on a VPS behind Caddy. Full table, versions and the
"deliberately not in the stack" list: [docs/stack.md](docs/stack.md). Do not introduce a
library that displaces a row in that table without an ADR.

## Layout

```
apps/        Expo apps (family-manager, shopping, finance)
packages/    shared TS: ui, auth, api, theme, config
services/    Go microservices, app-specific: auth, family, notifications, shopping
libs/go/     shared Go: auth, logger, rpc, database, events, storage
libs/proto/  .proto contracts — the ONLY cross-boundary contract surface
sdk/         generated clients: typescript/, go/
infra/       deployment
docs/        this workflow, ontology, generated graph
tools/       repo tooling (repo-graph extractor)
```

Naming, fixed: Go modules are `github.com/nnc/family-manager/<path>` (no `-service` suffix), TS
packages are `@fm/<name>`, proto packages are `<domain>.v1`, event subjects are
`<domain>.<entity>.<verb>`. Remaining Flutter dirs (`finance-android`, `notes-android`,
`notifications-android`) and `finance-legacy` are being retired — see docs/architecture.md.

## The knowledge graph

`just graph` re-extracts the repo into typed nodes and edges under `docs/graph/`. Every node
and edge carries `source`, `line`, `commit`, `extracted_at` — no fact without provenance.

Node types: `app pkg service golib gomod proto rpc msg table migration infra ext`.
Edge types: `DEPENDS_ON IMPLEMENTS CONSUMES DECLARES USES_MESSAGE PERSISTS_TO CREATES OWNED_BY RUNS_ON GENERATES`.

Use it, do not guess:

```sh
just graph                       # rebuild
just impact service:auth         # who breaks if I change this
node tools/repo-graph/extract.mjs --check   # CI: fail if stale
```

**Rule: before editing any file under `services/`, `libs/`, `packages/`, or `libs/proto`, run
`just impact <node>` and state the blast radius in your plan.** An edit whose blast radius you
did not check is an unplanned edit.

Changing the ontology: edit `docs/ontology.yaml` FIRST, then the mirror in
`tools/repo-graph/extract.mjs`, then regenerate. Never let the two drift.

## The task graph

Work is a DAG, not a script. Shape:

```
        ┌─ worker: service ─┐
scope ──┼─ worker: package ─┼─→ verify ─→ merge ─→ human gate
  │     └─ worker: app ─────┘      ↑
  └─ impact query (graph) ─────────┘
```

1. **Scope** — read the request, run `just impact` on every node you intend to touch. Output:
   the node list, the blast radius, the acceptance check per node.
2. **Split only what is independent.** Fan out across boundaries that never read each other's
   output (a Go service and an Expo screen behind an unchanged contract). Anything sequential —
   contract change → codegen → both sides — stays with ONE agent. More agents on sequential work
   makes it worse, measurably.
3. **Verify in a separate context** — `just verify` plus a reviewer that did not write the code.
4. **Merge with one owner.** Never let parallel findings land without a single merger.
5. **Human gate** on irreversible edges only: migrations, deploys, published contracts,
   dependency bumps in `packages/config`. Not on ordinary code edits.

Guardrails: one writer per file; loops capped at 3 rounds; the routing lives in this file, the
model fills the jobs, not the plan.

Full rationale and the per-change-type graphs (contract change, new service, new app, schema
migration) are in [docs/workflow.md](docs/workflow.md).

## Skills

Project skills in `.claude/skills/` — invoke instead of improvising:

| skill | use for |
|---|---|
| `repo-graph` | build/query the knowledge graph, blast radius, staleness |
| `plan-graph` | turn a request into a task DAG with gates before writing code |
| `new-service` | scaffold a Go microservice matching repo conventions |
| `new-expo-app` | scaffold an Expo app wired to shared packages |
| `contract-change` | edit a `.proto`, regenerate SDKs, walk the impact list |

## Autonomous mode

The default operating mode: a human writes a brief, the machine does the rest. Policy —
what runs unattended, what stops, the caps, the merge rule — is [docs/autonomy.md](docs/autonomy.md).
It is enforced by `just gate-check` (a script), not by prose.

```
/brief "<idea>"          human's only creative step -> docs/briefs/NNNN-slug.md
   -> decomposer agent   -> docs/backlog/NNNN-slug.json  (units, deps, acceptance, gates)
/autopilot NNNN-slug     branch auto/NNNN-slug; per unit: implement -> acceptance ->
                         verifier -> gate-check -> commit; blocked units escalate and the
                         loop CONTINUES; then PR, CI, squash-merge when green
/escalations             human answers; blocked units unblock; run resumes
/graph-health            periodic structural audit
```

Backlog state is a **file with a deterministic CLI** (`just backlog status|next|start|done|block|fail`),
not model memory — a run that loses its session re-reads the file and continues where it stopped.
Three failed attempts blocks a unit automatically; two consecutive unit failures stop the line.

Never commit to master directly, never apply a migration to a non-local database, never work
around a `gate-check` STOP — escalate and take the next unit.

## Commands, agents, hooks

Slash commands (`.claude/commands/`): `/brief`, `/autopilot`, `/escalations` (autonomous mode);
`/feature <desc>` (supervised plan → implement → verify loop), `/impact <node>`,
`/migrate <service> <change>` (ends at the human gate), `/graph-health`.

Subagents (`.claude/agents/`): `decomposer` (brief → backlog), `verifier` (read-only, fresh
context, **one question per spawn** — diverse skeptics beat one checklist), `go-implementer`,
`expo-implementer`. Spawn implementers
only for branches the plan marked PARALLEL with disjoint file lists; sequential work stays with
one agent.

Hooks (`.claude/settings.json`) enforce mechanically what prose cannot:

| hook | effect |
|---|---|
| PreToolUse guard | **blocks** hand-edits to `sdk/**`, `*.pb.go`, `*_pb.ts`, `services/*/db/**`, `docs/graph/**`, `.env` |
| PostToolUse refresh | re-runs `just graph` when a structural file changes (`*.proto`, `*.sql`, `go.mod`, `package.json`, compose) |
| Stop check | refuses to end a session with a stale graph, an uncommitted migration, or a contract change whose impact list was not walked |

If a hook blocks you, it is right and the plan is wrong — fix the approach, do not work around it.

## Commands

```sh
just tools         # buf, sqlc, air, golang-migrate (once)
just up            # docker compose: postgres + minio + nats
just install       # pnpm install
just dev-apps      # turbo dev (persistent)
just proto         # buf generate libs/proto -> sdk/go + sdk/typescript
just sqlc auth     # sqlc generate for services/auth
just check-go      # go build + vet + test across the workspace
just check-ts      # turbo lint + typecheck + test
just verify        # everything above + graph rebuild — the verify node
just impact <id>   # blast radius
```

## Conventions

**Auth**: `services/auth` is the only issuer. Other services verify the ES256 access JWT locally
through the `libs/go/auth` interceptor (JWKS, cached) — never by calling the auth service, never
with a shared secret. Apps keep tokens in expo-secure-store via `@fm/auth`. See
[ADR 0005](docs/adr/0005-auth.md); do not hand-roll token logic anywhere else.

**Go services** (`services/<name>`, module `github.com/nnc/family-manager/services/<name>`):
`cmd/server/main.go` · `internal/config` (viper) · `internal/grpc` · `internal/handler` ·
`internal/db/{migrations,queries}` + `sqlc.yaml` → generated `db/` · `.air.toml` for hot reload ·
`.env.example` committed, `.env` never. Shared concerns (auth middleware, logging, pg pool,
event bus) come from `libs/go/*` — do not re-implement them per service.

**Contracts**: every `.proto` lives in `libs/proto/<domain>/v1/`; domain events are contracts too
(`events.proto`, subjects `<domain>.<entity>.<verb>`). Services never import another service's Go
package; they import generated stubs from `sdk/go`. Adding a field is additive-only; removing or
renaming a field, or renumbering, is a breaking change — `just proto-check` fails it and it goes
through the human gate.

**TypeScript**: apps hold screens (expo-router) only. Anything reusable goes to `packages/*`;
anything talking to a service goes through `sdk/typescript` (generated) wrapped by `packages/api`
(TanStack Query). Styling is NativeWind against the `packages/theme` preset — no literal colors
or spacing in an app. Cross-package imports use `workspace:*`.

**Every new module registers itself**: a Go module in `go.work`, a TS package in
`pnpm-workspace.yaml` globs (automatic under `apps/`, `packages/`). Then `just graph`.

CI (`.github/workflows/verify.yml`) runs the same verify node on every PR as four independent
jobs: go, ts, contracts (`buf lint` + `buf breaking`), graph (`--check`). A structural change
committed without `just graph` fails CI — that is deliberate, it keeps the graph trustworthy.

## Definition of done

1. `just verify` green.
2. `docs/graph/graph.json` regenerated and committed if the change added or removed a node/edge.
3. Blast-radius list from step 1 of the task graph is fully addressed or explicitly deferred.
4. Migrations and contract changes acknowledged at the human gate.
