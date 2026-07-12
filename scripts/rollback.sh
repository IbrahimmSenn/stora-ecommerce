#!/usr/bin/env bash
# Rolls the deployment back to a previous image, optionally restoring the
# pre-migration database backup taken by deploy.sh.
#
#   scripts/rollback.sh ghcr.io/<owner>/stora-ecommerce:<prev-sha> [backups/db-....sql]
#
# Without a backup file only the application version is reverted (safe when the
# newer migrations are backward-compatible). With one, the database is restored
# to the pre-migration snapshot — orders placed after that snapshot are lost, so
# prefer `migrate down` for reversible schema changes.
set -euo pipefail
cd "$(dirname "$0")/.."

IMAGE="${1:?usage: rollback.sh <previous-image-ref> [backup.sql]}"
BACKUP="${2:-}"

# Same compose-file/smoke-url resolution as deploy.sh: shell env, then .env,
# then the default pair.
env_get() { sed -n "s/^$1=//p" .env 2>/dev/null | tail -1; }
export COMPOSE_FILE="${COMPOSE_FILE:-$(env_get COMPOSE_FILE)}"
export COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml:docker-compose.deploy.yml}"
SMOKE_URL="${SMOKE_URL:-$(env_get SMOKE_URL)}"

if [ -n "$BACKUP" ]; then
  [ -f "$BACKUP" ] || { echo "rollback: backup file $BACKUP not found" >&2; exit 1; }
  echo "==> stopping api before database restore"
  docker compose stop api
  echo "==> restoring database from $BACKUP"
  docker compose exec -T db psql -q \
    -U "${POSTGRES_USER:-admin}" -d "${POSTGRES_DB:-mystore}" < "$BACKUP"
fi

echo "==> starting previous version $IMAGE"
API_IMAGE="$IMAGE" docker compose up -d --no-build --wait --wait-timeout 180 api

echo "==> validating rolled-back version"
scripts/smoke.sh "${SMOKE_URL:-http://localhost:8080}"

echo "rollback: $IMAGE is live"
