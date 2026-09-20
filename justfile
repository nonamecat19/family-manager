set dotenv-load := true

default:
    @just --list

# ---------------------------------------------------------------- infra ----

# Start Postgres, MinIO and NATS in the background.
up:
    docker compose up -d

# Start the backing stores and every application service.
up-all:
    docker compose -f docker-compose.yml -f docker-compose.services.yml up -d --build

# Stop the local infrastructure containers.
down:
    docker compose down

# Stop the containers from both files.
down-all:
    docker compose -f docker-compose.yml -f docker-compose.services.yml down

# ------------------------------------------------------------- node side ----

# Install the pnpm workspace from the lockfile.
install:
    pnpm install

# Run every Expo app in development mode.
dev-apps:
    pnpm turbo run dev

# Build every TypeScript package and app.
build-apps:
    pnpm turbo run build

# Lint, typecheck and test the whole TypeScript side.
check-ts:
    pnpm turbo run lint typecheck test

# --------------------------------------------------------------- go side ----

# Reconcile go.work with every module's go.mod.
go-sync:
    go work sync

# Build, vet and test every module in go.work (the repo root is not a module).
#
# -race: the concurrency in this repo is small but load-bearing — the argon2 gate, the token
# sweep, the pgx pool shared across handlers. A data race there is a corrupted session, and it
# will not show up in a sequential test run.
#
# -shuffle=on: several suites share a package-level fake store. Order dependence between tests
# is a bug that only ever appears on someone else's machine.
#
# Build, vet and race-test every Go module in go.work.
check-go:
    #!/usr/bin/env bash
    set -euo pipefail
    for dir in $(go list -m -f '{{{{.Dir}}'); do
      echo "--- $dir"
      (cd "$dir" && go build ./... && go vet ./... && go test -race -shuffle=on ./...)
    done

# Not part of check-go: it needs a binary that check-go does not, and a missing linter must not
# be indistinguishable from a clean run.
#
# Lint every module in go.work against .golangci.yml.
lint-go:
    #!/usr/bin/env bash
    set -euo pipefail
    command -v golangci-lint >/dev/null || {
      echo "golangci-lint not installed; run 'just tools'" >&2; exit 1; }
    for dir in $(go list -m -f '{{{{.Dir}}'); do
      echo "--- $dir"
      (cd "$dir" && golangci-lint run ./...)
    done

# Reachability, not a dependency list: govulncheck reports the call paths, so an unused
# vulnerable function is not a reason to bump anything.
#
# Scan every Go module for known vulnerabilities it actually calls.
vuln:
    #!/usr/bin/env bash
    set -euo pipefail
    command -v govulncheck >/dev/null || {
      echo "govulncheck not installed; run 'just tools'" >&2; exit 1; }
    failed=0
    for dir in $(go list -m -f '{{{{.Dir}}'); do
      echo "--- $dir"
      (cd "$dir" && govulncheck ./...) || failed=1
    done
    exit "$failed"

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
    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
    go install golang.org/x/vuln/cmd/govulncheck@latest

# Print an argon2id hash for a password, with the parameters services/auth uses. For seeding
# the first account into a fresh database, or resetting one when nobody can sign in to do it
# the normal way. Reads stdin so the password stays out of shell history and `ps`.
#
# Print an argon2id hash for a password typed on stdin.
hashpw:
    cd services/auth && go run ./cmd/hashpw

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
#
# Pre-flight: launch the app and check the login screen renders. No backend needed.
test-mobile-smoke:
    @just test-mobile-run smoke-launch

# Run one Maestro flow by name, or the whole directory when flow is empty.
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

# A debug APK carries no JS bundle or fonts — it fetches them from Metro at runtime, so it
# shows "Unable to load script" on any device that cannot reach your dev machine. This target
# builds the standalone one: JS and the TTFs are packaged in, nothing needs to be running.
#
# Signed with the debug keystore that `expo prebuild` generates (android/app/build.gradle
# points release at signingConfigs.debug). That is fine for sideloading onto family phones and
# is NOT a Play Store build — publishing needs a real keystore that is not in this repo.
# Build the recipes APK (release, self-contained), against production unless told otherwise.
build-apk-release env="production":
    #!/usr/bin/env bash
    set -euo pipefail
    export EXPO_PUBLIC_API_ENV="{{env}}"
    cd apps/recipes
    npx expo prebuild --platform android --no-install --clean
    cd android
    # -x lintVitalRelease: lint reports MainApplication/MainActivity as not Instantiatable,
    # a false positive — both are Kotlin classes extending Application/ReactActivity, which
    # lint's release-vital pass does not resolve here. android/ is regenerated by prebuild
    # --clean on every build, so a build.gradle lint block would be wiped; the flag is the
    # only fix that survives. Correctness is gated by `just verify`, not by this pass.
    JAVA_HOME=${JAVA_HOME:-/usr/lib/jvm/java-17-openjdk} \
    ANDROID_HOME=${ANDROID_HOME:-$HOME/Android} \
    ./gradlew assembleRelease -x lintVitalRelease

# Install the release APK on a connected emulator/device.
install-apk-release:
    #!/usr/bin/env bash
    set -euo pipefail
    export ANDROID_HOME=${ANDROID_HOME:-$HOME/Android}
    "$ANDROID_HOME/platform-tools/adb" install -r apps/recipes/android/app/build/outputs/apk/release/app-release.apk

# Install the debug APK on a connected emulator/device.
install-apk:
    #!/usr/bin/env bash
    set -euo pipefail
    export ANDROID_HOME=${ANDROID_HOME:-$HOME/Android}
    "$ANDROID_HOME/platform-tools/adb" install -r apps/recipes/android/app/build/outputs/apk/debug/app-debug.apk

# -------------------------------------------------------------- desktop ----

# Deliberately builds nothing and starts nothing: `tauri dev` attaches to the Metro dev server
# that apps/notes/desktop/src-tauri/tauri.conf.json points at (devUrl, :8081), so run
# `pnpm --filter @fm/app-notes dev` in another terminal first. Letting tauri own Metro's
# lifecycle would restart it — and drop fast refresh — every time the Rust side recompiles.
#
# Needs a Rust toolchain and webkit2gtk-4.1; see apps/notes/desktop/README.md.
#
# Run the Commonplace desktop shell against the running Expo dev server.
desktop-dev:
    cd apps/notes/desktop/src-tauri && cargo tauri dev

# The export is a separate step rather than tauri's beforeBuildCommand so that it happens
# exactly once — the PKGBUILD does its own export and would otherwise run a second one.
# --platform web, not the app's own `build` script: that one is `expo export --platform all`,
# and the native bundles it also produces are dead weight to a desktop window.
#
# Output: .deb, .rpm and AppImage under apps/notes/desktop/src-tauri/target/release/bundle/.
# The web assets are compiled into the binary, so the artifact does not read apps/notes/dist
# at runtime.
#
# Export the notes app to web, then bundle it as a desktop app (deb/rpm/AppImage).
desktop-build:
    pnpm --filter @fm/app-notes exec expo export --platform web
    cd apps/notes/desktop/src-tauri && cargo tauri build

# Not part of `just verify`, for the same reason lint-go and vuln are not: this repo compiles
# Rust in exactly one directory, most contributors will never install a toolchain, and a
# target that silently passes because the tool is missing is worse than no target at all.
# So the split is: locally a missing cargo is a SKIP (a Rust-less machine is the normal case
# here, and failing it would train people to ignore the gate), but under CI=true it is a
# FAILURE — the desktop job installs the toolchain, and if that install ever breaks, the job
# must go red instead of quietly checking nothing.
#
# Format, lint and typecheck the Tauri desktop shell. Skips when cargo is not installed.
check-desktop:
    #!/usr/bin/env bash
    set -euo pipefail
    if ! command -v cargo >/dev/null; then
      if [ "${CI:-}" = "true" ]; then
        echo "cargo not installed, and CI=true: the desktop check must not be skipped in CI" >&2
        exit 1
      fi
      echo "cargo not installed; skipping the desktop shell check." >&2
      echo "Install a Rust toolchain (see apps/notes/desktop/README.md) to run it." >&2
      exit 0
    fi
    for component in fmt clippy; do
      cargo "$component" --version >/dev/null 2>&1 || {
        echo "cargo-$component not installed; run 'rustup component add ${component/fmt/rustfmt}'" >&2
        exit 1; }
    done
    cd apps/notes/desktop/src-tauri
    # No web export needed: nothing here reads apps/notes/dist until `cargo tauri build`
    # bundles it, so the gate runs on a checkout with no pnpm install behind it.
    cargo fmt --all --check
    cargo clippy --all-targets --all-features -- -D warnings
