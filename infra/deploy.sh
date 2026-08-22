#!/usr/bin/env bash
# Deploy family-manager on the VPS. Runs ON the box, from /opt/family-manager, either by hand or
# over SSH from the GitHub Actions push-deploy job:
#
#   ssh root@79.108.160.103 '/opt/family-manager/deploy.sh <commit-sha>'
#
# Pulls the pinned tag, brings the stack up, waits for all four services to report healthy, and
# rolls back to the previously deployed tag if any of them does not. Safe to run twice in
# a row with the same tag: everything it does is idempotent.
#
# It never removes volumes. `docker compose down -v`, `docker volume rm` and `docker system
# prune --volumes` do not appear anywhere below and must not be added — pgdata is the database,
# and caddydata is the issued certificates.
set -euo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/opt/family-manager}"
PROJECT="family-manager"
COMPOSE_FILE="docker-compose.prod.yml"
STATE_FILE=".deployed-tag"
SERVICES=(auth family finance recipes)
# Migrations run at boot, so first start on an empty database is the slow case. 180s is chosen
# to be longer than that, not longer than a healthy restart.
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-180}"

log() { printf '==> %s\n' "$*"; }
fail() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

cd "$DEPLOY_DIR" || fail "$DEPLOY_DIR does not exist — provision the box first"

compose() { docker compose -p "$PROJECT" -f "$COMPOSE_FILE" "$@"; }

# --- preflight ---------------------------------------------------------------
# Everything that would fail *after* the containers are replaced is checked before they are.
[[ -f "$COMPOSE_FILE" ]] || fail "$DEPLOY_DIR/$COMPOSE_FILE is missing"
[[ -f .env ]] || fail "$DEPLOY_DIR/.env is missing — copy infra/.env.example and fill it in"
[[ -f Caddyfile ]] || fail "$DEPLOY_DIR/Caddyfile is missing"
# nats.conf carries the JetStream store limits, which have no command-line equivalent. Without
# it nats exits immediately, and family/finance/recipes all wait on nats being healthy — so a
# missing file here is three services down, which is worth catching before anything is replaced.
[[ -f nats.conf ]] || fail "$DEPLOY_DIR/nats.conf is missing — nats will not start without it"

# The signing key, and its permissions. Compose bind-mounts a file-backed secret with the host's
# ownership and mode — `mode:` on the secret is rejected and the service-level uid/gid/mode are
# ignored with a warning — so a key that is root-owned 0600 (which is what a careful admin, or a
# hardening script, will leave behind) is unreadable by the distroless `nonroot` uid the auth
# image runs as, and auth exits at boot while the other three come up fine. That is a deploy
# that looks 3/4 successful and has no working login, so it is corrected here rather than
# reported. deploy.sh runs as root, so the chown is ours to make.
KEY=secrets/auth-signing-key.pem
KEY_UID=65532 # distroless nonroot
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
	# No argument means "redeploy whatever .env pins", which is how a config-only change is
	# rolled out. It is not how CI calls this.
	tag="$(grep -E '^IMAGE_TAG=' .env | tail -n1 | cut -d= -f2- || true)"
	tag="${tag:-latest}"
	log "no tag given, using IMAGE_TAG=$tag from .env"
fi

previous="$(cat "$STATE_FILE" 2>/dev/null || true)"
export IMAGE_TAG="$tag"

# --- health ------------------------------------------------------------------
# Read from the containers' own healthchecks (`/server -healthcheck`, added with the compose
# healthcheck blocks), not probed from outside.
#
# This used to `compose exec -T caddy wget` its way to each service, because the service images
# are distroless — no shell, no wget — and only Caddy publishes ports. That worked, but it made
# the deploy's notion of health a second, parallel definition of the same thing, and it could
# not start until Caddy could. Now the container answers the question compose already asks it,
# which means `docker ps` and this script agree, and a service that goes unhealthy an hour
# later is visible in the same place.
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
			# A service with no healthcheck block cannot be waited on, and silently treating
			# that as success is how this script would stop noticing a broken deploy.
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

# --- deploy ------------------------------------------------------------------
log "pulling $tag"
# Pull first and separately: a bad tag or an expired GHCR credential should fail while the old
# containers are still serving, not halfway through replacing them.
compose pull

log "starting stack at $tag (previous: ${previous:-none})"
# --remove-orphans cleans up containers dropped from the file — notably minio, if this box was
# ever brought up with the dev topology. It removes containers, never volumes.
compose up -d --remove-orphans

if wait_healthy; then
	printf '%s\n' "$tag" > "$STATE_FILE"
	# Dangling images only. Not `-a` (which would delete the previous tag and make the rollback
	# above a re-download at the worst moment) and never `--volumes`.
	docker image prune -f >/dev/null || true
	log "deployed $tag"
	exit 0
fi

# --- rollback ----------------------------------------------------------------
printf 'ERROR: %s\n' "$tag did not come up healthy" >&2

if [[ -z "$previous" || "$previous" == "$tag" ]]; then
	# Nothing better to go back to. The stack is left running on purpose: a partly-healthy
	# stack still serves the services that did come up, and `compose down` would turn a
	# degraded deploy into a total outage while someone reads the logs.
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
