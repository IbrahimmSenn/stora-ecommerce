#!/usr/bin/env bash
# Rolls the deployment back to a previous image, optionally restoring the
# pre-migration database backup taken by deploy.sh.
#
#   scripts/rollback.sh ghcr.io/<owner>/iloveshopping:<prev-sha> [backups/db-....sql]
#
# Without a backup file only the application version is reverted (safe when the
# newer migrations are backward-compatible). With one, the database is restored
# to the pre-migration snapshot — orders placed after that snapshot are lost, so
# prefer `migrate down` for reversible schema changes.
set -euo pipefail
cd "$(dirname "$0")/.."

IMAGE="${1:?usage: rollback.sh <previous-image-ref> [backup.sql]}"
BACKUP="${2:-}"
COMPOSE="docker compose -f docker-compose.yml -f docker-compose.deploy.yml"

if [ -n "$BACKUP" ]; then
  [ -f "$BACKUP" ] || { echo "rollback: backup file $BACKUP not found" >&2; exit 1; }
  echo "==> stopping api before database restore"
  docker compose stop api
  echo "==> restoring database from $BACKUP"
  docker compose exec -T db psql -q \
    -U "${POSTGRES_USER:-admin}" -d "${POSTGRES_DB:-mystore}" < "$BACKUP"
fi

echo "==> starting previous version $IMAGE"
API_IMAGE="$IMAGE" $COMPOSE up -d --no-build --wait --wait-timeout 180 api

echo "==> validating rolled-back version"
scripts/smoke.sh

echo "rollback: $IMAGE is live"
