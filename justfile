set dotenv-load := true

default:
    @just --list


up:
    docker compose up -d

up-all:
    docker compose -f docker-compose.yml -f docker-compose.services.yml up -d --build

down:
    docker compose down

down-all:
    docker compose -f docker-compose.yml -f docker-compose.services.yml down


install:
    pnpm install

dev-apps:
    pnpm turbo run dev

build-apps:
    pnpm turbo run build

check-ts:
    pnpm turbo run lint typecheck test


go-sync:
    go work sync

check-go:
    #!/usr/bin/env bash
    set -euo pipefail
    for dir in $(go list -m -f '{{{{.Dir}}'); do
      echo "--- $dir"
      (cd "$dir" && go build ./... && go vet ./... && go test -race -shuffle=on ./...)
    done

lint-go:
    #!/usr/bin/env bash
    set -euo pipefail
    command -v golangci-lint >/dev/null || {
      echo "golangci-lint not installed; run 'just tools'" >&2; exit 1; }
    for dir in $(go list -m -f '{{{{.Dir}}'); do
      echo "--- $dir"
      (cd "$dir" && golangci-lint run ./...)
    done

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


proto:
    cd libs/proto && buf generate

proto-check:
    cd libs/proto && buf lint && buf breaking --against '../../.git#branch=master,subdir=libs/proto'

sqlc service:
    cd services/{{service}} && sqlc generate


tools:
    go install github.com/bufbuild/buf/cmd/buf@latest
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
    go install github.com/air-verse/air@latest
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
    go install golang.org/x/vuln/cmd/govulncheck@latest

hashpw:
    cd services/auth && go run ./cmd/hashpw


graph:
    node tools/repo-graph/extract.mjs

impact node:
    node tools/repo-graph/extract.mjs --impact {{node}}

graph-check:
    node tools/repo-graph/extract.mjs --check


gate-check:
    node tools/autonomy/gate-check.mjs

gate-check-branch range="master..HEAD":
    node tools/autonomy/gate-check.mjs --range {{range}}

backlog *args:
    node tools/backlog/backlog.mjs {{args}}


verify: check-go check-ts graph


test-mobile flow="":
    @just test-mobile-run {{flow}}

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

build-apk env="production":
    #!/usr/bin/env bash
    set -euo pipefail
    export EXPO_PUBLIC_API_ENV="{{env}}"
    cd apps/recipes
    npx expo prebuild --platform android --no-install --clean
    cd android
    JAVA_HOME=${JAVA_HOME:-/usr/lib/jvm/java-17-openjdk} \
    ANDROID_HOME=${ANDROID_HOME:-$HOME/Android} \
    ./gradlew assembleDebug

build-apk-release env="production":
    #!/usr/bin/env bash
    set -euo pipefail
    export EXPO_PUBLIC_API_ENV="{{env}}"
    cd apps/recipes
    npx expo prebuild --platform android --no-install --clean
    cd android
    JAVA_HOME=${JAVA_HOME:-/usr/lib/jvm/java-17-openjdk} \
    ANDROID_HOME=${ANDROID_HOME:-$HOME/Android} \
    ./gradlew assembleRelease -x lintVitalRelease

install-apk-release:
    #!/usr/bin/env bash
    set -euo pipefail
    export ANDROID_HOME=${ANDROID_HOME:-$HOME/Android}
    "$ANDROID_HOME/platform-tools/adb" install -r apps/recipes/android/app/build/outputs/apk/release/app-release.apk

install-apk:
    #!/usr/bin/env bash
    set -euo pipefail
    export ANDROID_HOME=${ANDROID_HOME:-$HOME/Android}
    "$ANDROID_HOME/platform-tools/adb" install -r apps/recipes/android/app/build/outputs/apk/debug/app-debug.apk


desktop-dev:
    cd apps/notes/desktop/src-tauri && cargo tauri dev

desktop-build:
    pnpm --filter @fm/app-notes exec expo export --platform web
    cd apps/notes/desktop/src-tauri && cargo tauri build

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
    cargo fmt --all --check
    cargo clippy --all-targets --all-features -- -D warnings
