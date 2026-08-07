set dotenv-load := true

default:
    @just --list

# ---------------------------------------------------------------- infra ----

up:
    docker compose up -d

down:
    docker compose down

# ------------------------------------------------------------- node side ----

install:
    pnpm install

dev-apps:
    pnpm turbo run dev

build-apps:
    pnpm turbo run build

check-ts:
    pnpm turbo run lint typecheck test

# --------------------------------------------------------------- go side ----

go-sync:
    go work sync

# Build, vet and test every module in go.work (the repo root is not a module).
check-go:
    #!/usr/bin/env bash
    set -euo pipefail
    for dir in $(go list -m -f '{{{{.Dir}}'); do
      echo "--- $dir"
      (cd "$dir" && go build ./... && go vet ./... && go test ./...)
    done

# ------------------------------------------------------------- contracts ----

# Regenerate Go stubs + Connect handlers + TypeScript SDK from libs/proto.
proto:
    cd libs/proto && buf generate

# Lint contracts and check for breaking changes against origin/master.
proto-check:
    cd libs/proto && buf lint && buf breaking --against '../../.git#branch=master,subdir=libs/proto'

# sqlc codegen for one service, e.g. `just sqlc auth`
sqlc service:
    cd services/{{service}} && sqlc generate

# ----------------------------------------------------------------- tools ----

# Install the Go-side toolchain (buf, sqlc, air, golang-migrate) into $GOBIN.
tools:
    go install github.com/bufbuild/buf/cmd/buf@latest
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
    go install github.com/air-verse/air@latest
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# ----------------------------------------------------------------- graph ----

# Rebuild the repo knowledge graph (docs/graph/graph.json + graph.mmd).
graph:
    node tools/repo-graph/extract.mjs

# Blast radius for a node id, e.g. `just impact service:auth`
impact node:
    node tools/repo-graph/extract.mjs --impact {{node}}

# Fail if the committed graph is stale (for CI).
graph-check:
    node tools/repo-graph/extract.mjs --check

# ------------------------------------------------------------- autonomy ----

# Enforce docs/autonomy.md on the staged diff. exit 3 = STOP, escalate.
gate-check:
    node tools/autonomy/gate-check.mjs

# Same check across a whole branch, before merging.
gate-check-branch range="master..HEAD":
    node tools/autonomy/gate-check.mjs --range {{range}}

# Backlog state machine, e.g. `just backlog status 0007-shopping`
backlog *args:
    node tools/backlog/backlog.mjs {{args}}

# ----------------------------------------------------------------- gates ----

# The verify node of the task graph. Run before any merge.
verify: check-go check-ts graph
