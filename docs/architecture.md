# Architecture & migration state

## Target layout

```
apps/                Expo apps — screens + navigation only
  family-manager/
  shopping/
  recipes/
  finance/
  notes/             Commonplace — Expo app + a Tauri desktop shell in desktop/
packages/            shared TS, consumed via workspace:*
  ui/                design-system components
  auth/              session/token handling, hooks
  api/               typed wrappers over sdk/typescript, query keys, caching
  theme/             tokens, dark/light
  config/            eslint/tsconfig/babel/metro presets — single source of truth
services/            Go microservices, one module each
  auth/ family/ notes/ notifications/ shopping/
libs/go/             shared Go modules, one module each
  auth/ logger/ database/ events/
libs/proto/          .proto contracts, versioned: <domain>/v1/*.proto
sdk/
  typescript/        generated TS client (buf) — never hand-edited
  go/                generated Go stubs — never hand-edited
infra/               compose overrides, deploy manifests
tools/repo-graph/    knowledge-graph extractor
```

Boundaries, in one line each:

- An app imports packages and `sdk/typescript`. Never another app.
- A package imports other packages and the SDK. Never an app.
- A service imports `libs/go/*` and `sdk/go`. **Never another service's Go package** — cross
  service means a contract in `libs/proto` or an event in `libs/go/events`.
- Generated code (`sdk/*`, `services/*/db/`, `*.pb.go`) is never edited by hand.

## Current state (migration in progress)

## Naming

| thing | scheme | example |
|---|---|---|
| Go module | `github.com/nnc/family-manager/<path>` | `.../services/auth`, `.../libs/go/logger`, `.../sdk/go` |
| TS package | `@fm/<name>` | `@fm/ui`, `@fm/api` |
| proto package | `<domain>.v1` in `libs/proto/<domain>/v1/` | `auth.v1` |
| event subject | `<domain>.<entity>.<verb>` | `family.member.invited` |
| service dir | bare domain, no `-service` suffix | `services/notifications` |

One module prefix, matching the directory tree. `go_package` is never written by hand — buf
managed mode derives it (`libs/proto/buf.gen.yaml`).

## Current state (migration in progress)

| exists today | target | status |
|---|---|---|
| `services/auth/` | — | **done** — moved, module renamed to `github.com/nnc/family-manager/services/auth`, contract moved to `libs/proto/auth/v1` |
| `postgres/init/` | `infra/postgres/` | pending |
| `docker-compose.yml` (infra) + `docker-compose.services.yml` (services) | root, plus `infra/` overrides | keep both at root |
| `notes-android/` | `apps/notes/` + `services/notes/` | **replaced** — Commonplace on `notes.v1`; the Flutter dir is dead weight, delete it in its own change |
| `notifications-android/` | replaced by `apps/*` (Expo) | Flutter, being retired — nothing has replaced it yet |

Nothing in the table is moved automatically. Each move is its own change, and it must end with
`just graph` showing the same edges under the new paths — the graph is how you prove a move did
not silently drop a dependency.

### The move procedure (used for `auth-service`, reuse for the rest)

```sh
just graph && jq -S '[.edges[]|{from,rel,to}]' docs/graph/graph.json > /tmp/before.json
git mv <old> <new>
sed -i 's|<old module path>|<new module path>|g' <new>/go.mod $(grep -rl '<old module path>' <new>)
# go.work: update the `use` line
just graph && jq -S '[.edges[]|{from,rel,to}]' docs/graph/graph.json > /tmp/after.json
diff /tmp/before.json /tmp/after.json   # ONLY path-derived ids may differ; edge count must match
just check-go
```

The diff is the proof. Same edges under new ids = the move dropped nothing. A missing edge means
a dependency was silently severed — fix before committing.

## Object storage is public by default, which is why notes has no image blocks in v1

`libs/go/storage`'s `EnsureBucket` creates the bucket **and sets an anonymous-read policy on
it** (`ADR 0007`). That is right for `services/recipes`, whose photos are illustrations of a
shared recipe, and wrong for `services/notes`, whose whole promise is that a note is private
until its owner shares it: an object under a public-read bucket is fetchable by anyone holding
the URL, whether or not the note was ever shared, and unsharing the note would not take the
image back.

So `services/notes` ships with **no `NOTES_STORAGE_*` environment at all** — not in
`docker-compose.services.yml`, not in `infra/docker-compose.prod.yml`, not in its `.env.example`. With
`STORAGE_ENDPOINT` unset, `imagesOrNil` logs `image storage disabled` and returns nil, exactly
as `services/recipes` does; the service boots, every procedure but one works, `UploadNoteImage`
errors — and, the point of the whole arrangement, `EnsureBucket` is never called, so no notes
bucket and no public policy exist to leak through.

The contract keeps `UploadNoteImage` and `BlockType.BLOCK_TYPE_IMAGE`: deleting either is a
breaking change (`just proto-check`) that buys nothing, and the numbers have to survive for the
day the feature returns. The proto says so in the comments. What the feature actually needs
first is **presigned reads in `libs/go/storage`** — a per-object, expiring URL minted for a
caller the service has already authorised — at which point the notes bucket can be private and
`UploadNoteImage` can be wired to a real editor button. Configuring `NOTES_STORAGE_*` before
that exists re-opens the hole; it is not a feature flag.

## Codegen

- **buf** — `libs/proto` → `sdk/go` (protoc-gen-go + protoc-gen-connect-go) and
  `sdk/typescript/src` (protoc-gen-es v2), configured in `libs/proto/buf.yaml` and
  `buf.gen.yaml`. `just proto` generates, `just proto-check` lints and blocks breaking changes.
  Services predating the migration keep their local protoc Makefile until their contract moves
  into `libs/proto`.
- **sqlc** — per service: `internal/db/queries/*.sql` + `internal/db/migrations/*.sql` → `db/`
  (`just sqlc <service>`).
- **air** — per service hot reload via `.air.toml`.
