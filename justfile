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

# The flows REGISTER ACCOUNTS AND CREATE RECIPES, so build the APK against a local stack —
# `just build-apk local` — before running them. The default APK points at production, where
# these flows would write test data into the family's real database. There is no staging tier.
#
# Requires: emulator running, APK installed, backend running (`just up`).
# Run Maestro mobile UI tests against the recipes app (or a specific flow).
test-mobile flow="":
    @just test-mobile-run {{flow}}

# Fast pre-flight: launches the app and checks the login screen renders. No backend
# needed. Run this before burning time on device-network setup (see apps/recipes/TESTING.md) —
# a build/render crash shows up here in seconds instead of a silent timeout on-device.
test-mobile-smoke:
    @just test-mobile-run smoke-launch

test-mobile-run flow:
    #!/usr/bin/env bash
    set -euo pipefail
    export PATH="$PATH:$HOME/.maestro/bin"
    if [ -z "{{flow}}" ]; then
      maestro test .maestro/flows/
    else
      maestro test ".maestro/flows/recipes/{{flow}}.yaml"
    fi

# Requires ANDROID_HOME and JAVA_HOME (Java 17). In Docker if local toolchain is unavailable.
#
# Defaults to the PRODUCTION backend, matching `pnpm dev` (apps/recipes/app.config.js).
# `just build-apk local` builds against a local stack instead — which is what the Maestro
# flows need, since they write data.
#
# For a real device (not emulator) against a local stack, localhost resolves to the phone, so
# also point the service at your machine's LAN IP:
#   EXPO_PUBLIC_RECIPES_URL=http://192.168.1.20:8084 just build-apk local
# Build the recipes APK (debug), against production unless told otherwise.
build-apk env="production":
    #!/usr/bin/env bash
    set -euo pipefail
    export EXPO_PUBLIC_API_ENV="{{env}}"
    cd apps/recipes
    # --clean: android/ is gitignored and regenerated from app.config.js every time, so
    # config changes (API URL, cleartext, etc.) always take effect instead of silently
    # reusing whatever was prebuilt last.
    npx expo prebuild --platform android --no-install --clean
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
