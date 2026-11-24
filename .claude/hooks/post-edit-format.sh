#!/usr/bin/env bash
# PostToolUse hook (Write|Edit): formats the edited file inside $CLAUDE_PROJECT_DIR.
#   *.go                                -> gofmt -w
#   frontend/src/**/*.{svelte,ts,js}    -> the project's prettier --write
# Files outside the project are ignored. The hook never blocks: problems are
# reported as one warning line on stderr and the exit code stays 0.
# Input: hook JSON on stdin (tool_input.file_path is absolute).

# Upper bound in seconds for one formatter run.
FORMAT_TIMEOUT="${ZFAKTURY_HOOK_FORMAT_TIMEOUT:-30}"

warn() { printf 'post-edit-format: warning: %s\n' "$*" >&2; }

if [ -z "${CLAUDE_PROJECT_DIR:-}" ]; then
  echo "post-edit-format: CLAUDE_PROJECT_DIR is not set" >&2
  exit 1
fi
PROJECT=$(realpath -e -- "$CLAUDE_PROJECT_DIR" 2>/dev/null) || {
  echo "post-edit-format: CLAUDE_PROJECT_DIR is not an existing directory" >&2
  exit 1
}

if ! command -v jq >/dev/null 2>&1; then
  warn "jq not found, skipping format"
  exit 0
fi

file=$(jq -r '.tool_input.file_path // empty' 2>/dev/null)
[ -n "$file" ] || exit 0

# realpath -m resolves symlinks and ".." so a path cannot escape the project.
file=$(realpath -m -- "$file") || exit 0
case "$file" in
  "$PROJECT"/*) ;;
  *) exit 0 ;;
esac
[ -f "$file" ] || exit 0

case "$file" in
  *.go)
    if ! command -v gofmt >/dev/null 2>&1; then
      warn "gofmt not found, skipping $file"
      exit 0
    fi
    timeout "$FORMAT_TIMEOUT" gofmt -w "$file" || warn "gofmt failed on $file"
    ;;
  "$PROJECT"/frontend/src/*.svelte | "$PROJECT"/frontend/src/*.ts | "$PROJECT"/frontend/src/*.js)
    prettier="$PROJECT/frontend/node_modules/.bin/prettier"
    if [ ! -x "$prettier" ]; then
      warn "frontend/node_modules missing (run npm ci), skipping prettier for $file"
      exit 0
    fi
    (cd "$PROJECT/frontend" && timeout "$FORMAT_TIMEOUT" "$prettier" --write "$file" >/dev/null) ||
      warn "prettier failed on $file"
    ;;
esac

exit 0
