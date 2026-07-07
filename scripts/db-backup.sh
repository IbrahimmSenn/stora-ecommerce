#!/usr/bin/env bash
# Dumps the compose Postgres to backups/db-<timestamp>.sql and prints the path.
# Run before every migration so rollback.sh can restore the pre-migration state.
set -euo pipefail
cd "$(dirname "$0")/.."

mkdir -p backups
file="backups/db-$(date +%Y%m%d-%H%M%S).sql"

docker compose exec -T db pg_dump \
  -U "${POSTGRES_USER:-admin}" -d "${POSTGRES_DB:-mystore}" \
  --clean --if-exists > "$file"

echo "$file"
