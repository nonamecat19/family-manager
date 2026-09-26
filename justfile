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
    cd libs/proto && buf generate --template buf.gen.java.yaml --path notes

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

gate-check-range range="origin/master..HEAD":
    node tools/autonomy/gate-check.mjs --range {{range}}

backlog *args:
    node tools/backlog/backlog.mjs {{args}}


verify: check-go check-ts graph


tg-up:
    docker compose -f docker-compose.yml -f docker-compose.services.yml -f docker-compose.tgemu.yml up -d --build

tg-down:
    docker compose -f docker-compose.yml -f docker-compose.services.yml -f docker-compose.tgemu.yml down

test-telegram-emu:
    cd services/telegram && go test -race -count=1 ./internal/tgemu/

test-telegram:
    cd services/telegram && go test -tags e2e -count=1 -v ./e2e/

test-android app flow="":
    #!/usr/bin/env bash
    set -euo pipefail
    export PATH="$PATH:$HOME/.maestro/bin"
    adb=${ANDROID_HOME:-$HOME/Android/Sdk}/platform-tools/adb
    for port in 8081 8082 8083 8084 8085 8086 8087; do "$adb" reverse tcp:$port tcp:$port >/dev/null; done
    email="e2e-$(date +%s)@example.test"
    if [ -z "{{flow}}" ]; then
      maestro test -e TEST_EMAIL="$email" ".maestro/flows/{{app}}/"
    else
      maestro test -e TEST_EMAIL="$email" ".maestro/flows/{{app}}/{{flow}}.yaml"
    fi


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

build-apk-release env="production" app="recipes":
    #!/usr/bin/env bash
    set -euo pipefail
    export EXPO_PUBLIC_API_ENV="{{env}}"
    sdk_dir="${ANDROID_HOME:-$HOME/Android/Sdk}"
    if [ ! -d "$sdk_dir/platform-tools" ]; then
      sdk_dir="${ANDROID_SDK_ROOT:-$HOME/Android}"
    fi
    export ANDROID_HOME="$sdk_dir" ANDROID_SDK_ROOT="$sdk_dir"
    cd "apps/{{app}}"
    npx expo prebuild --platform android --no-install --clean
    cd android
    JAVA_HOME=${JAVA_HOME:-/usr/lib/jvm/java-17-openjdk} \
    ./gradlew assembleRelease -x lintVitalRelease

install-apk-release app="recipes":
    #!/usr/bin/env bash
    set -euo pipefail
    sdk_dir="${ANDROID_HOME:-$HOME/Android/Sdk}"
    if [ ! -d "$sdk_dir/platform-tools" ]; then
      sdk_dir="${ANDROID_SDK_ROOT:-$HOME/Android}"
    fi
    export ANDROID_HOME="$sdk_dir"
    "$ANDROID_HOME/platform-tools/adb" install -r "apps/{{app}}/android/app/build/outputs/apk/release/app-release.apk"

check-ios-widgets app:
    #!/usr/bin/env bash
    set -euo pipefail
    cd "apps/{{app}}"
    npx expo prebuild --platform ios --no-install --clean
    project="$(find ios -maxdepth 2 -name project.pbxproj -print -quit)"
    if [ -z "$project" ] || ! grep -Eq 'WidgetExtension|WidgetKit' "$project"; then
      echo "Widget target missing from $project" >&2
      exit 1
    fi
    if [ "$(uname -s)" != "Darwin" ]; then
      if [ "${CI:-}" = "true" ]; then
        echo "iOS widgets require macOS in CI" >&2
        exit 1
      fi
      echo "Widget target found; skipping iOS compile outside macOS."
      exit 0
    fi
    cd ios
    pod install
    workspace="$(find . -maxdepth 1 -name '*.xcworkspace' -print -quit)"
    scheme="$(basename "$workspace" .xcworkspace)"
    xcodebuild -workspace "$workspace" -scheme "$scheme" -configuration Debug -sdk iphonesimulator CODE_SIGNING_ALLOWED=NO build

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
