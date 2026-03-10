#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "==> Building frontend..."
cd "$PROJECT_DIR/frontend"
if [ -f "package.json" ]; then
    npm ci
    npm run build
else
    echo "    No frontend package.json found, skipping frontend build."
    # Create placeholder for embed to work
    mkdir -p "$PROJECT_DIR/web/frontend/build"
    echo '<!DOCTYPE html><html><body>Frontend not built</body></html>' > "$PROJECT_DIR/web/frontend/build/index.html"
fi

echo "==> Building Go binary..."
cd "$PROJECT_DIR"
go build -o zfaktury ./cmd/zfaktury

echo "==> Build complete: ./zfaktury"
echo "    Run with: ./zfaktury serve"
