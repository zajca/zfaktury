#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Ensure git hooks are set up
git config core.hooksPath scripts

# Ensure placeholder exists for embed directive
mkdir -p "$PROJECT_DIR/web/frontend/build"
if [ ! -f "$PROJECT_DIR/web/frontend/build/index.html" ]; then
    echo '<!DOCTYPE html><html><body>Use Vite dev server</body></html>' > "$PROJECT_DIR/web/frontend/build/index.html"
fi

MAILPIT_CONTAINER="zfaktury-mailpit"

cleanup() {
    echo ""
    echo "==> Stopping all processes..."
    docker rm -f "$MAILPIT_CONTAINER" &>/dev/null || true
    kill 0 2>/dev/null
    wait 2>/dev/null
    echo "==> Done."
}
trap cleanup EXIT

# Start Mailpit via Docker for local email testing (optional)
if command -v docker &>/dev/null; then
    # Remove stale container if exists
    docker rm -f "$MAILPIT_CONTAINER" &>/dev/null || true
    echo "==> Starting Mailpit (SMTP :1025, UI http://localhost:8025)..."
    docker run --rm --name "$MAILPIT_CONTAINER" -p 1025:1025 -p 8025:8025 axllent/mailpit &
else
    echo "==> Docker not found, skipping Mailpit email catcher."
fi

# Start Vite dev server in the background
if [ -f "$PROJECT_DIR/frontend/package.json" ]; then
    echo "==> Starting Vite dev server..."
    cd "$PROJECT_DIR/frontend"
    npm run dev &
    cd "$PROJECT_DIR"
fi

# Ensure local config exists
if [ ! -f "$PROJECT_DIR/config.dev.toml" ]; then
    cp "$PROJECT_DIR/config.dev.dist.toml" "$PROJECT_DIR/config.dev.toml"
    echo "==> Created config.dev.toml from config.dev.dist.toml"
fi

# Start Go server in dev mode
echo "==> Starting Go server in dev mode..."
CGO_ENABLED=0 go run -tags server ./cmd/zfaktury --config "$PROJECT_DIR/config.dev.toml" serve --dev --port 8080

wait
