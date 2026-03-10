#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Ensure placeholder exists for embed directive
mkdir -p "$PROJECT_DIR/web/frontend/build"
if [ ! -f "$PROJECT_DIR/web/frontend/build/index.html" ]; then
    echo '<!DOCTYPE html><html><body>Use Vite dev server</body></html>' > "$PROJECT_DIR/web/frontend/build/index.html"
fi

cleanup() {
    echo ""
    echo "==> Stopping all processes..."
    kill 0 2>/dev/null
    wait 2>/dev/null
    echo "==> Done."
}
trap cleanup EXIT

# Start Vite dev server in the background
if [ -f "$PROJECT_DIR/frontend/package.json" ]; then
    echo "==> Starting Vite dev server..."
    cd "$PROJECT_DIR/frontend"
    npm run dev &
    cd "$PROJECT_DIR"
fi

# Start Go server in dev mode
echo "==> Starting Go server in dev mode..."
go run ./cmd/zfaktury serve --dev --port 8080

wait
