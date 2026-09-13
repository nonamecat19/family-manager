#!/usr/bin/env bash
set -euo pipefail

if [ "${1:-}" = "--env-file" ]; then
  [ -n "${2:-}" ] || { echo "--env-file needs a path" >&2; exit 2; }
  # shellcheck disable=SC1090
  set -a; . "$2"; set +a
  shift 2
fi

MANIFEST="${1:-}"
[ -n "$MANIFEST" ] && [ -f "$MANIFEST" ] || {
  echo "usage: $0 [--env-file FILE] <images.tsv>" >&2
  exit 2
}

for v in R2_ENDPOINT R2_ACCESS_KEY_ID R2_SECRET_ACCESS_KEY R2_BUCKET; do
  [ -n "${!v:-}" ] || { echo "$v is not set" >&2; exit 2; }
done
command -v aws >/dev/null || { echo "aws cli not found" >&2; exit 2; }

ENDPOINT="${R2_ENDPOINT%/}"
ENDPOINT="${ENDPOINT%/$R2_BUCKET}"
case "$ENDPOINT" in
  http://*|https://*) ;;
  *) ENDPOINT="https://$ENDPOINT" ;;
esac

export AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID"
export AWS_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY"
export AWS_DEFAULT_REGION=auto
export AWS_REQUEST_CHECKSUM_CALCULATION=when_required
export AWS_RESPONSE_CHECKSUM_VALIDATION=when_required

total=$(wc -l < "$MANIFEST")
n=0
missing=0
while IFS=$'\t' read -r src key; do
  n=$((n + 1))
  if [ ! -f "$src" ]; then
    echo "  MISSING  $src" >&2
    missing=$((missing + 1))
    continue
  fi
  aws s3api put-object \
    --endpoint-url "$ENDPOINT" \
    --bucket "$R2_BUCKET" \
    --key "$key" \
    --body "$src" \
    --content-type image/webp \
    --output text --query 'ETag' >/dev/null
  if [ $((n % 25)) -eq 0 ] || [ "$n" -eq "$total" ]; then
    echo "  uploaded $n/$total"
  fi
done < "$MANIFEST"

if [ "$missing" -gt 0 ]; then
  echo "$missing source file(s) missing — those recipes will have a dead image_url" >&2
  exit 1
fi
echo "done: $n object(s) in $R2_BUCKET"
