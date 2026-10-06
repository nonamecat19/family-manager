#!/usr/bin/env bash
set -euo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/opt/family-manager}"
COMPOSE_FILE="${COMPOSE_FILE:-$DEPLOY_DIR/docker-compose.prod.yml}"
BACKUP_ROOT="${BACKUP_ROOT:-/var/backups/family-manager}"
KEEP_RUNS="${KEEP_RUNS:-14}"
MIN_FREE_MB="${MIN_FREE_MB:-2048}"

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[fail]\033[0m %s\n' "$*" >&2; exit 1; }

[[ ${EUID} -eq 0 ]] || die "run this as root"
[[ -r "$COMPOSE_FILE" ]] || die "no compose file at $COMPOSE_FILE"

cd "$DEPLOY_DIR"
PG=$(docker compose -f "$COMPOSE_FILE" ps -q postgres) || true
[[ -n "$PG" ]] || die "the postgres container is not running — nothing to back up"

PGUSER=$(docker exec "$PG" printenv POSTGRES_USER)
[[ -n "$PGUSER" ]] || die "could not read POSTGRES_USER from the container"

free_mb=$(df -Pm "$BACKUP_ROOT" 2>/dev/null | awk 'NR==2 {print $4}' || df -Pm / | awk 'NR==2 {print $4}')
(( free_mb >= MIN_FREE_MB )) || die "only ${free_mb} MB free, need ${MIN_FREE_MB} MB — refusing to run"

STAMP=$(date -u +%Y-%m-%dT%H-%M-%SZ)
RUN_DIR="$BACKUP_ROOT/$STAMP"
mkdir -p "$RUN_DIR"
chmod 0700 "$BACKUP_ROOT" "$RUN_DIR"

cleanup_failed() {
  local code=$?
  if (( code != 0 )); then
    warn "run failed — removing the incomplete $RUN_DIR"
    rm -rf "$RUN_DIR"
  fi
}
trap cleanup_failed EXIT

log "Dumping globals (roles, grants)"
docker exec "$PG" pg_dumpall -U "$PGUSER" --globals-only > "$RUN_DIR/globals.sql"
[[ -s "$RUN_DIR/globals.sql" ]] || die "globals dump is empty"

mapfile -t DATABASES < <(docker exec "$PG" psql -U "$PGUSER" -d postgres -tAc \
  "SELECT datname FROM pg_database WHERE NOT datistemplate AND datname <> 'postgres' ORDER BY datname")
(( ${#DATABASES[@]} > 0 )) || die "no databases found — that is not a healthy cluster"

for db in "${DATABASES[@]}"; do
  out="$RUN_DIR/$db.dump"
  log "Dumping $db"
  docker exec "$PG" pg_dump -U "$PGUSER" -Fc --no-owner --no-acl "$db" > "$out"
  docker exec -i "$PG" pg_restore --list < "$out" > "$RUN_DIR/$db.toc" \
    || die "$db dumped but could not be read back — treating the run as failed"
  printf '  %s  %s\n' "$(du -h "$out" | cut -f1)" "$out"
done

if [[ -r "$DEPLOY_DIR/.env" ]]; then
  log "Copying deployment config"
  tar -C "$DEPLOY_DIR" -czf "$RUN_DIR/config.tar.gz" .env secrets 2>/dev/null \
    || warn "config archive incomplete (secrets/ may not exist yet)"
  chmod 0600 "$RUN_DIR/config.tar.gz" 2>/dev/null || true
fi

chmod 0600 "$RUN_DIR"/*.dump "$RUN_DIR/globals.sql" 2>/dev/null || true
ln -sfn "$RUN_DIR" "$BACKUP_ROOT/latest"

trap - EXIT

mapfile -t runs < <(find "$BACKUP_ROOT" -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | sort)
if (( ${#runs[@]} > KEEP_RUNS )); then
  for old in "${runs[@]:0:${#runs[@]}-KEEP_RUNS}"; do
    log "Pruning $old"
    rm -rf "${BACKUP_ROOT:?}/$old"
  done
fi

log "Done: ${#DATABASES[@]} database(s) in $RUN_DIR"
printf '    total %s, %s run(s) retained, %s MB free\n' \
  "$(du -sh "$RUN_DIR" | cut -f1)" \
  "$(find "$BACKUP_ROOT" -mindepth 1 -maxdepth 1 -type d | wc -l)" \
  "$(df -Pm "$BACKUP_ROOT" | awk 'NR==2 {print $4}')"
