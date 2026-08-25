#!/usr/bin/env bash
#
# Nightly Postgres backup for the family-manager VPS. Installed at /opt/family-manager/backup.sh
# and driven by the family-manager-backup.timer systemd unit (see bootstrap-vps.sh).
#
# WHAT THIS PROTECTS AGAINST, AND WHAT IT DOES NOT
#
# The dumps land on the same disk as the database they came from. That covers the failures that
# actually happen day to day: a migration that drops the wrong column, a DELETE without a WHERE,
# an app bug that corrupts rows, "I deleted that recipe by accident". It does NOT cover the disk
# dying, the VPS being destroyed, or the provider losing the machine — in all three the backups
# go with the database. Treat this as an undo button, not as disaster recovery. Shipping a copy
# off-box is a separate, still-missing piece; docs/adr/0004 is the argument for why it matters.
#
# A backup nobody has restored is a guess, so every dump is read back with `pg_restore --list`
# before the run is allowed to succeed, and restore.sh exists to make the drill cheap.
set -euo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/opt/family-manager}"
COMPOSE_FILE="${COMPOSE_FILE:-$DEPLOY_DIR/docker-compose.prod.yml}"
BACKUP_ROOT="${BACKUP_ROOT:-/var/backups/family-manager}"
# Keep two weeks of nightly runs. At ~5 MB a run that is under 100 MB, which is nothing against
# a 20 GB disk — the limit is about noise in the directory, not about space.
KEEP_RUNS="${KEEP_RUNS:-14}"
# Refuse to run when the disk is nearly full. A backup that fills the last gigabyte takes the
# whole box down, which is a worse outcome than a missed nightly run.
MIN_FREE_MB="${MIN_FREE_MB:-2048}"

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[fail]\033[0m %s\n' "$*" >&2; exit 1; }

[[ ${EUID} -eq 0 ]] || die "run this as root"
[[ -r "$COMPOSE_FILE" ]] || die "no compose file at $COMPOSE_FILE"

# Resolve the container through compose rather than hardcoding family-manager-postgres-1: the
# project name follows the directory, so a rename would silently break a hardcoded name.
cd "$DEPLOY_DIR"
PG=$(docker compose -f "$COMPOSE_FILE" ps -q postgres) || true
[[ -n "$PG" ]] || die "the postgres container is not running — nothing to back up"

PGUSER=$(docker exec "$PG" printenv POSTGRES_USER)
[[ -n "$PGUSER" ]] || die "could not read POSTGRES_USER from the container"

free_mb=$(df -Pm "$BACKUP_ROOT" 2>/dev/null | awk 'NR==2 {print $4}' || df -Pm / | awk 'NR==2 {print $4}')
(( free_mb >= MIN_FREE_MB )) || die "only ${free_mb} MB free, need ${MIN_FREE_MB} MB — refusing to run"

# UTC, sortable, and safe in a filename. The sort order is what the retention prune relies on.
STAMP=$(date -u +%Y-%m-%dT%H-%M-%SZ)
RUN_DIR="$BACKUP_ROOT/$STAMP"
mkdir -p "$RUN_DIR"
# Dumps contain every row in the database, including password hashes.
chmod 0700 "$BACKUP_ROOT" "$RUN_DIR"

# A run that dies halfway leaves a directory that looks like a backup but is not one. Clear it,
# so the newest directory present is always a complete run.
cleanup_failed() {
  local code=$?
  if (( code != 0 )); then
    warn "run failed — removing the incomplete $RUN_DIR"
    rm -rf "$RUN_DIR"
  fi
}
trap cleanup_failed EXIT

# Roles and their passwords live in the cluster, not in any single database. Without these a
# restore onto a fresh cluster produces databases whose owner does not exist.
log "Dumping globals (roles, grants)"
docker exec "$PG" pg_dumpall -U "$PGUSER" --globals-only > "$RUN_DIR/globals.sql"
[[ -s "$RUN_DIR/globals.sql" ]] || die "globals dump is empty"

# Every non-template database except the default `postgres`, discovered rather than listed: a
# fifth service added later gets backed up without anyone remembering to edit this script.
mapfile -t DATABASES < <(docker exec "$PG" psql -U "$PGUSER" -d postgres -tAc \
  "SELECT datname FROM pg_database WHERE NOT datistemplate AND datname <> 'postgres' ORDER BY datname")
(( ${#DATABASES[@]} > 0 )) || die "no databases found — that is not a healthy cluster"

for db in "${DATABASES[@]}"; do
  out="$RUN_DIR/$db.dump"
  log "Dumping $db"
  # -Fc: compressed, and restorable one table at a time, which is what you want at 03:00 when
  # only one table is wrong. Plain SQL would be all-or-nothing.
  docker exec "$PG" pg_dump -U "$PGUSER" -Fc --no-owner --no-acl "$db" > "$out"
  # Read it back immediately. pg_restore --list parses the whole archive's table of contents, so
  # a truncated or corrupt dump fails here rather than on the night it is needed.
  docker exec -i "$PG" pg_restore --list < "$out" > "$RUN_DIR/$db.toc" \
    || die "$db dumped but could not be read back — treating the run as failed"
  printf '  %s  %s\n' "$(du -h "$out" | cut -f1)" "$out"
done

# The deployment's own configuration. Not strictly data, but the signing key and the R2 secret
# are the two things that cannot simply be retyped: rotating the signing key logs every device
# out, and Cloudflare shows an R2 secret exactly once. 0600, same as the originals.
if [[ -r "$DEPLOY_DIR/.env" ]]; then
  log "Copying deployment config"
  tar -C "$DEPLOY_DIR" -czf "$RUN_DIR/config.tar.gz" .env secrets 2>/dev/null \
    || warn "config archive incomplete (secrets/ may not exist yet)"
  chmod 0600 "$RUN_DIR/config.tar.gz" 2>/dev/null || true
fi

chmod 0600 "$RUN_DIR"/*.dump "$RUN_DIR/globals.sql" 2>/dev/null || true
ln -sfn "$RUN_DIR" "$BACKUP_ROOT/latest"

trap - EXIT

# Prune oldest-first. Only complete runs are on disk (see cleanup_failed), so counting
# directories counts backups.
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
