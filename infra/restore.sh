#!/usr/bin/env bash
#
# Restores one database from a backup.sh dump. Installed at /opt/family-manager/restore.sh.
#
#   ./restore.sh recipes                       # from the newest backup
#   ./restore.sh recipes 2026-08-25T03-00-00Z  # from a specific run
#   ./restore.sh --drill recipes               # rehearsal: restore into a scratch database
#
# --drill is the mode to run regularly. It restores into <db>_restore_drill, compares table
# counts against the live database, prints the result and drops the scratch copy — proving the
# dump is restorable without touching anything anyone is using. A backup nobody has restored is
# a guess; docs/adr/0004 argues that an untested restore is worse than no backup at all, because
# it buys confidence that is not there.
#
# Without --drill this is DESTRUCTIVE: the target database's contents are dropped and replaced.
# It asks for the database name back before doing it.
set -euo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/opt/family-manager}"
COMPOSE_FILE="${COMPOSE_FILE:-$DEPLOY_DIR/docker-compose.prod.yml}"
BACKUP_ROOT="${BACKUP_ROOT:-/var/backups/family-manager}"

log()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[fail]\033[0m %s\n' "$*" >&2; exit 1; }

DRILL=0
if [[ ${1:-} == "--drill" ]]; then DRILL=1; shift; fi

DB="${1:-}"
RUN="${2:-latest}"
[[ -n "$DB" ]] || die "usage: $0 [--drill] <database> [backup-run]"
[[ ${EUID} -eq 0 ]] || die "run this as root"

RUN_DIR="$BACKUP_ROOT/$RUN"
DUMP="$RUN_DIR/$DB.dump"
[[ -r "$DUMP" ]] || die "no dump at $DUMP (available runs: $(find "$BACKUP_ROOT" -mindepth 1 -maxdepth 1 -type d -printf '%f ' | sort))"

cd "$DEPLOY_DIR"
PG=$(docker compose -f "$COMPOSE_FILE" ps -q postgres) || true
[[ -n "$PG" ]] || die "the postgres container is not running"
PGUSER=$(docker exec "$PG" printenv POSTGRES_USER)

# No -i. A query needs nothing on stdin, and an interactive docker exec inherits the caller's
# stdin — inside a `while read` loop it swallows the rest of the list being iterated, so the
# loop runs exactly once. That is how the first version of the drill below "passed" after
# comparing a single table.
psql_q() { docker exec "$PG" psql -U "$PGUSER" -d "$1" -tAc "$2"; }

if (( DRILL )); then
  TARGET="${DB}_restore_drill"
  log "Drill: restoring $DB from $RUN into $TARGET"
  # Drop any leftover from an interrupted previous drill before recreating.
  psql_q postgres "DROP DATABASE IF EXISTS \"$TARGET\"" >/dev/null
  psql_q postgres "CREATE DATABASE \"$TARGET\"" >/dev/null
  # A drill that leaves a scratch database behind is a drill that quietly costs disk forever.
  trap 'docker exec -i "$PG" psql -U "$PGUSER" -d postgres -tAc "DROP DATABASE IF EXISTS \"'"$TARGET"'\"" >/dev/null 2>&1 || true' EXIT

  docker exec -i "$PG" pg_restore -U "$PGUSER" -d "$TARGET" --no-owner --no-acl < "$DUMP" \
    || die "restore failed — this backup would not have worked"

  log "Comparing $TARGET against the live $DB"
  fail=0
  # Read the list fully before iterating, so nothing inside the loop can interfere with the
  # stream it is reading from.
  mapfile -t tables < <(psql_q "$DB" "SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename")
  (( ${#tables[@]} > 0 )) || die "the live $DB has no tables to compare against"
  for table in "${tables[@]}"; do
    [[ -n "$table" ]] || continue
    live=$(psql_q "$DB" "SELECT count(*) FROM \"$table\"")
    restored=$(psql_q "$TARGET" "SELECT count(*) FROM \"$table\"")
    if [[ "$live" == "$restored" ]]; then
      printf '    %-24s %8s rows  ok\n' "$table" "$restored"
    else
      # Not automatically a bug: rows written after the dump was taken legitimately differ.
      # It is flagged rather than failed so a human reads it.
      printf '    %-24s live %s / restored %s  \033[1;33mDIFFERS\033[0m\n' "$table" "$live" "$restored"
      fail=1
    fi
  done

  if (( fail )); then
    warn "some counts differ. Expected if the database has been written to since the dump;"
    warn "investigate if it has not."
  else
    log "Drill passed: ${#tables[@]} table(s) restored with the same row count."
  fi
  exit 0
fi

# --- the real thing --------------------------------------------------------------------------

warn "This REPLACES the contents of the live '$DB' database with the copy from $RUN."
warn "Stop the services that write to it first, or they will write into a half-restored database:"
warn "  docker compose -f $COMPOSE_FILE stop $DB"
printf '  Type the database name to confirm: '
read -r confirm
[[ "$confirm" == "$DB" ]] || die "aborted"

# --clean --if-exists drops each object before recreating it, so this works on a database that
# already has data. --single-transaction makes it all-or-nothing: a failure halfway leaves the
# database as it was rather than half replaced.
log "Restoring $DB from $RUN"
docker exec -i "$PG" pg_restore -U "$PGUSER" -d "$DB" \
  --clean --if-exists --no-owner --no-acl --single-transaction < "$DUMP" \
  || die "restore failed — the database was rolled back to its previous contents"

log "Restored. Start the services again and check the data before trusting it."
