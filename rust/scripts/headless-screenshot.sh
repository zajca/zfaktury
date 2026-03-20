#!/usr/bin/env bash
set -euo pipefail

# Headless GPUI screenshot via cage + grim
# Usage: ./headless-screenshot.sh <app-command> [output.png] [wait-seconds]
#
# Runs the app inside an isolated headless Wayland session (cage compositor)
# and captures a screenshot via grim. No window appears on the user's desktop.
#
# Requirements: cage, grim (provided by nix develop)

APP="${1:?Usage: $0 <app-command> [output.png] [wait-seconds]}"
OUTPUT="${2:-/tmp/zfaktury-screenshot.png}"
WAIT="${3:-3}"

# Verify deps
command -v cage >/dev/null 2>&1 || { echo "ERROR: cage not found. Run from 'nix develop'." >&2; exit 1; }
command -v grim >/dev/null 2>&1 || { echo "ERROR: grim not found. Run from 'nix develop'." >&2; exit 1; }

# Isolated runtime directory -- no interference with user's Hyprland
export XDG_RUNTIME_DIR=$(mktemp -d /tmp/headless-wayland-XXXXXX)
trap 'rm -rf "$XDG_RUNTIME_DIR"' EXIT

# Headless wlroots config
unset WAYLAND_DISPLAY
export WLR_BACKENDS=headless
export WLR_LIBINPUT_NO_DEVICES=1
export WLR_HEADLESS_OUTPUTS=1

# Start cage (auto-exits when the app closes)
cage -- $APP &
CAGE_PID=$!

# Wait for Wayland socket to appear
SOCKET=""
for _ in $(seq 1 40); do
    SOCKET=$(find "$XDG_RUNTIME_DIR" -maxdepth 1 -name 'wayland-*' -not -name '*.lock' 2>/dev/null | head -1 || true)
    [ -n "$SOCKET" ] && break
    sleep 0.25
done

if [ -z "$SOCKET" ]; then
    echo "ERROR: Wayland socket not found after 10s" >&2
    kill "$CAGE_PID" 2>/dev/null || true
    exit 1
fi

export WAYLAND_DISPLAY=$(basename "$SOCKET")
echo "Wayland socket: $WAYLAND_DISPLAY"

# Wait for app to render
echo "Waiting ${WAIT}s for app to render..."
sleep "$WAIT"

# Check if cage is still running
if ! kill -0 "$CAGE_PID" 2>/dev/null; then
    echo "ERROR: cage/app exited before screenshot" >&2
    exit 1
fi

# Take screenshot
mkdir -p "$(dirname "$OUTPUT")"
grim "$OUTPUT"
echo "Screenshot saved: $OUTPUT"

# Cleanup
kill "$CAGE_PID" 2>/dev/null || true
wait "$CAGE_PID" 2>/dev/null || true
