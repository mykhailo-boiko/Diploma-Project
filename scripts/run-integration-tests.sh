#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

if [ -z "${ADMIN_PASSWORD:-}" ]; then
    echo "ADMIN_PASSWORD env var required" >&2
    exit 2
fi

echo "[1/3] Verifying stack health..."
if ! curl -fsS "${GATEWAY_URL:-http://localhost:8080}/health" > /dev/null; then
    echo "Gateway is not healthy. Run 'docker compose up -d' first." >&2
    exit 3
fi

if [ ! -d ".venv-itest" ]; then
    echo "[2/3] Creating test virtualenv..."
    python3 -m venv .venv-itest
    .venv-itest/bin/pip install --quiet --upgrade pip
    .venv-itest/bin/pip install --quiet pytest httpx websockets
else
    echo "[2/3] Using existing .venv-itest"
fi

echo "[3/3] Running integration tests..."
.venv-itest/bin/python -m pytest tests/integration -v "$@"
