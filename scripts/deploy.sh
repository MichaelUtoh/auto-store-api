#!/usr/bin/env bash
# Production deploy script for Hetzner (Docker Compose).
# Run from the server after git clone, or via GitHub Actions SSH.
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BRANCH="${DEPLOY_BRANCH:-main}"
COMPOSE="${COMPOSE:-docker compose}"

cd "$REPO_DIR"

echo "==> Deploying ${REPO_DIR} (branch: ${BRANCH})"

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "error: not a git repository: ${REPO_DIR}" >&2
  exit 1
fi

git fetch origin "${BRANCH}"
git reset --hard "origin/${BRANCH}"

echo "==> Building and restarting api + worker"
${COMPOSE} up -d --build api worker

echo "==> Service status"
${COMPOSE} ps api worker

echo "==> Health check"
for i in 1 2 3 4 5; do
  if curl -fsS http://127.0.0.1:8089/health >/dev/null; then
    echo "API healthy"
    exit 0
  fi
  echo "waiting for API (${i}/5)..."
  sleep 3
done

echo "error: API health check failed" >&2
${COMPOSE} logs --tail 50 api
exit 1
