#!/usr/bin/env bash
set -euo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/opt/family-manager}"
PROJECT="family-manager"
COMPOSE_FILE="docker-compose.prod.yml"
STATE_FILE=".deployed-tag"
SERVICES=(auth family finance notes recipes)
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-180}"

log() { printf '==> %s\n' "$*"; }
fail() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

cd "$DEPLOY_DIR" || fail "$DEPLOY_DIR does not exist — provision the box first"

compose() { docker compose -p "$PROJECT" -f "$COMPOSE_FILE" "$@"; }

[[ -f "$COMPOSE_FILE" ]] || fail "$DEPLOY_DIR/$COMPOSE_FILE is missing"
[[ -f .env ]] || fail "$DEPLOY_DIR/.env is missing — copy infra/.env.example and fill it in"
[[ -f Caddyfile ]] || fail "$DEPLOY_DIR/Caddyfile is missing"
[[ -f nats.conf ]] || fail "$DEPLOY_DIR/nats.conf is missing — nats will not start without it"

KEY=secrets/auth-signing-key.pem
KEY_UID=65532
if [[ ! -s "$KEY" ]]; then
	fail "$DEPLOY_DIR/$KEY is missing or empty — auth cannot start without it. Generate one with:
    openssl ecparam -name prime256v1 -genkey -noout -out $DEPLOY_DIR/$KEY
  and re-run. There is deliberately no fallback: an auth service that invents its own signing
  key silently invalidates every issued token."
fi
if [[ "$(stat -c '%u:%g:%a' "$KEY")" != "$KEY_UID:$KEY_UID:400" ]]; then
	log "fixing $KEY ownership/mode (was $(stat -c '%U:%G %a' "$KEY"), want $KEY_UID:$KEY_UID 0400)"
	chown "$KEY_UID:$KEY_UID" "$KEY"
	chmod 0400 "$KEY"
fi

[[ -d postgres-init ]] || fail \
	"postgres-init/ is missing — on a fresh pgdata volume the services have no databases"

tag="${1:-}"
if [[ -z "$tag" ]]; then
	tag="$(grep -E '^IMAGE_TAG=' .env | tail -n1 | cut -d= -f2- || true)"
	tag="${tag:-latest}"
	log "no tag given, using IMAGE_TAG=$tag from .env"
fi

previous="$(cat "$STATE_FILE" 2>/dev/null || true)"
export IMAGE_TAG="$tag"

health_of() {
	local cid
	cid="$(compose ps -q "$1" 2>/dev/null | head -n1)"
	if [[ -z "$cid" ]]; then
		printf 'absent\n'
		return
	fi
	docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' \
		"$cid" 2>/dev/null || printf 'absent\n'
}

wait_healthy() {
	local deadline=$((SECONDS + HEALTH_TIMEOUT)) svc state
	for svc in "${SERVICES[@]}"; do
		while :; do
			state="$(health_of "$svc")"
			case "$state" in
			healthy)
				log "$svc healthy"
				break
				;;
			none)
				printf '%s has no healthcheck; cannot verify the deploy\n' "$svc" >&2
				return 1
				;;
			esac
			if ((SECONDS >= deadline)); then
				printf '%s is %s after %ss\n' "$svc" "$state" "$HEALTH_TIMEOUT" >&2
				compose logs --tail 50 "$svc" >&2 || true
				return 1
			fi
			sleep 3
		done
	done
}

log "pulling $tag"
compose pull

log "starting stack at $tag (previous: ${previous:-none})"
compose up -d --remove-orphans

if wait_healthy; then
	printf '%s\n' "$tag" > "$STATE_FILE"
	docker image prune -f >/dev/null || true
	log "deployed $tag"
	exit 0
fi

printf 'ERROR: %s\n' "$tag did not come up healthy" >&2

if [[ -z "$previous" || "$previous" == "$tag" ]]; then
	fail "no previous tag recorded — leaving the stack up, investigate with: docker compose -f $COMPOSE_FILE logs"
fi

log "rolling back to $previous"
export IMAGE_TAG="$previous"
compose up -d --remove-orphans

if wait_healthy; then
	log "rolled back to $previous; $tag needs fixing before the next deploy"
else
	printf 'ERROR: %s\n' "rollback to $previous is also unhealthy — this is not an image problem, check postgres/nats and the logs" >&2
fi
exit 1
