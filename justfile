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

# --------------------------------------------------------------- mobile ----

# Run Maestro mobile UI tests against the recipes app (or a specific flow).
# Requires: emulator running, APK installed, backend running (`just up`).
test-mobile flow="":
    @just test-mobile-run {{flow}}

test-mobile-run flow:
    #!/usr/bin/env bash
    set -euo pipefail
    export PATH="$PATH:$HOME/.maestro/bin"
    if [ -z "{{flow}}" ]; then
      maestro test .maestro/flows/
    else
      maestro test ".maestro/flows/recipes/{{flow}}.yaml"
    fi

# Build the recipes APK (debug). Requires ANDROID_HOME and JAVA_HOME (Java 17).
# In Docker if local toolchain is unavailable.
build-apk:
    #!/usr/bin/env bash
    set -euo pipefail
    cd apps/recipes
    if [ -d android ]; then
      echo "Prebuild already done"
    else
      npx expo prebuild --platform android --no-install
    fi
    cd android
    JAVA_HOME=${JAVA_HOME:-/usr/lib/jvm/java-17-openjdk} \
    ANDROID_HOME=${ANDROID_HOME:-$HOME/Android} \
    ./gradlew assembleDebug

# Install the debug APK on a connected emulator/device.
install-apk:
    #!/usr/bin/env bash
    set -euo pipefail
    export ANDROID_HOME=${ANDROID_HOME:-$HOME/Android}
    "$ANDROID_HOME/platform-tools/adb" install -r apps/recipes/android/app/build/outputs/apk/debug/app-debug.apk
