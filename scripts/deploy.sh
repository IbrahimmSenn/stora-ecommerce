#!/usr/bin/env bash
# Deploys a prebuilt image to the docker-compose environment on this host.
#
#   scripts/deploy.sh ghcr.io/<owner>/iloveshopping:<sha>
#
# Steps: pull image -> pre-migration DB backup (if db is running) -> apply
# migrations -> start services -> smoke-test. On smoke failure the script exits
# non-zero; run scripts/rollback.sh <previous-image> [backup.sql] to revert.
#
# Works the same on a laptop, a self-hosted CI runner, or a VPS — the target is
# wherever this script runs. Requires: docker compose v2, an .env file next to
# docker-compose.yml (see .env.example).
set -euo pipefail
cd "$(dirname "$0")/.."

IMAGE="${1:?usage: deploy.sh <image-ref>}"
COMPOSE="docker compose -f docker-compose.yml -f docker-compose.deploy.yml"

[ -f .env ] || { echo "deploy: .env missing — copy .env.example and fill it in" >&2; exit 1; }

echo "==> pulling $IMAGE"
docker pull "$IMAGE" || echo "    (pull failed — assuming image is already loaded locally)"

echo "==> recording current version for rollback"
current=$(docker inspect --format '{{index .Config.Image}}' "$(docker compose ps -q api 2>/dev/null)" 2>/dev/null || true)
[ -n "$current" ] && echo "    previous image: $current"

backup=""
if docker compose ps db --status running -q 2>/dev/null | grep -q .; then
  echo "==> backing up database before migration"
  backup=$(scripts/db-backup.sh)
  echo "    backup: $backup"
fi

echo "==> deploying (migrations run via the compose migrate service)"
API_IMAGE="$IMAGE" $COMPOSE up -d --no-build --wait --wait-timeout 180

echo "==> post-deploy validation"
if ! scripts/smoke.sh; then
  echo "deploy: smoke tests FAILED for $IMAGE" >&2
  [ -n "$current" ] && echo "rollback: scripts/rollback.sh $current ${backup:-}" >&2
  exit 1
fi

echo "deploy: $IMAGE is live"
if [ -n "$backup" ]; then
  echo "deploy: pre-migration backup at $backup (kept for rollback)"
fi
