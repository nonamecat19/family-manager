#!/usr/bin/env bash
# Pushes the dish photos named by import.mjs's image manifest into the recipes bucket.
#
#   R2_ENDPOINT=... R2_ACCESS_KEY_ID=... R2_SECRET_ACCESS_KEY=... R2_BUCKET=... \
#     tools/recipe-import/upload-images.sh out/images.tsv
#
# Or point it at the deployment's env file, which already holds all four:
#
#   tools/recipe-import/upload-images.sh --env-file /opt/family-manager/.env out/images.tsv
#
# The manifest is "<local file>\t<object key>" per line. Keys are <family_id>/<recipe_id>.webp
# — the same shape services/recipes writes when the app uploads a photo — so a later in-app
# upload replaces the seeded object rather than leaving it orphaned in the bucket.
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

# R2_ENDPOINT may carry a /<bucket> suffix (that is the form Cloudflare's dashboard shows).
# The S3 API wants the bare host, with the bucket named separately.
ENDPOINT="${R2_ENDPOINT%/}"
ENDPOINT="${ENDPOINT%/$R2_BUCKET}"

export AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID"
export AWS_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY"
# R2 ignores the region but the SDK refuses to sign without one.
export AWS_DEFAULT_REGION=auto
# R2 does not implement the trailing checksum the newer CLI adds by default.
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
  # One line per 25 so a 457-image run shows progress without 457 lines of noise.
  if [ $((n % 25)) -eq 0 ] || [ "$n" -eq "$total" ]; then
    echo "  uploaded $n/$total"
  fi
done < "$MANIFEST"

if [ "$missing" -gt 0 ]; then
  echo "$missing source file(s) missing — those recipes will have a dead image_url" >&2
  exit 1
fi
echo "done: $n object(s) in $R2_BUCKET"
