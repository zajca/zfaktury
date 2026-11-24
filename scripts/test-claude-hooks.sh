#!/usr/bin/env bash
# Tests the Claude Code hooks in .claude/hooks by feeding them sample hook JSON on
# stdin. Every case runs against a temporary fake project (a copy of the hooks, a
# git repo, stub golangci-lint/npm/prettier), never against the real lint.
# Usage: scripts/test-claude-hooks.sh        (exit 0 when all cases pass)
set -u

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
HOOKS_SRC="$REPO_ROOT/.claude/hooks"
BASH_BIN=$(command -v bash)

T=$(mktemp -d)
trap 'rm -rf "$T"' EXIT
PROJ="$T/proj"
OUTSIDE="$T/outside"
STUBS="$T/stubs"
BASE="$T/basebin"
CALLS="$T/calls.log"

# Directory with only the tools the hooks need, so "tool missing" is reproducible.
mkdir -p "$BASE" "$STUBS" "$PROJ/.claude" "$OUTSIDE"
for tool in bash env awk jq git timeout realpath sed tail tr cat dirname gofmt sleep; do
  p=$(command -v "$tool") && ln -s "$p" "$BASE/$tool"
done

cp -r "$HOOKS_SRC" "$PROJ/.claude/hooks"
for d in "$PROJ" "$OUTSIDE"; do
  git init -q "$d"
  : >"$d/go.mod"
done
mkdir -p "$PROJ/sub" "$PROJ/frontend/src" "$PROJ/frontend/node_modules/.bin"
echo '{}' >"$PROJ/frontend/package.json"

# Stubs: exit codes come from FAKE_* variables, every call is logged.
printf '%s\n' '#!/usr/bin/env bash' \
  'echo "golangci-lint $*" >>"$CALLS"' \
  '[ -z "${FAKE_SLEEP:-}" ] || sleep "$FAKE_SLEEP"' \
  '[ "${FAKE_GO_RC:-0}" -eq 0 ] || echo "golangci boom" >&2' \
  'exit "${FAKE_GO_RC:-0}"' >"$STUBS/golangci-lint"
printf '%s\n' '#!/usr/bin/env bash' \
  'echo "npm $*" >>"$CALLS"' \
  '[ "${FAKE_NPM_FAIL:-}" != "$2" ] || { echo "npm $2 boom" >&2; exit 1; }' \
  'exit 0' >"$STUBS/npm"
printf '%s\n' '#!/usr/bin/env bash' \
  'echo "prettier $*" >>"$CALLS"' \
  'exit 0' >"$PROJ/frontend/node_modules/.bin/prettier"
chmod +x "$STUBS/golangci-lint" "$STUBS/npm" "$PROJ/frontend/node_modules/.bin/prettier"

pass=0
fail=0
ERR_FILE="$T/stderr.txt"
OUT_FILE="$T/stdout.txt"

# check <name> <hook> <stdin json> <expected rc> <stderr regex or ""> [VAR=val ...]
check() {
  local name=$1 hook=$2 json=$3 want=$4 pattern=$5 rc err
  shift 5
  : >"$CALLS"
  printf '%s' "$json" | env -i PATH="$BASE:$STUBS" CALLS="$CALLS" "$@" \
    "$BASH_BIN" "$PROJ/.claude/hooks/$hook" 2>"$ERR_FILE" >"$OUT_FILE"
  rc=$?
  err=$(cat "$ERR_FILE")
  local out msg
  out=$(cat "$OUT_FILE")
  msg=$(jq -r '.systemMessage // empty' <<<"$out" 2>/dev/null)
  if [ "$rc" -ne "$want" ]; then
    echo "FAIL $name: rc=$rc want=$want stderr=$err"
    fail=$((fail + 1))
  elif [ -n "$pattern" ] && ! grep -Eq -- "$pattern" <<<"$err"; then
    echo "FAIL $name: stderr does not match /$pattern/: $err"
    fail=$((fail + 1))
  elif [ "$rc" -eq 0 ] && [ -n "$pattern" ] && ! grep -Eq -- "$pattern" <<<"$msg"; then
    # A warning on an exit-0 hook must also reach the user as a systemMessage.
    echo "FAIL $name: stdout has no systemMessage matching /$pattern/: $out"
    fail=$((fail + 1))
  elif [ "$rc" -eq 0 ] && [ -z "$pattern" ] && [ -n "$out" ]; then
    echo "FAIL $name: unexpected stdout: $out"
    fail=$((fail + 1))
  else
    echo "ok   $name"
    pass=$((pass + 1))
  fi
}

# expect_calls <name> <regex>: the stub call log must (not) match.
expect_called() {
  if grep -Eq -- "$2" "$CALLS"; then echo "ok   $1"; pass=$((pass + 1)); else
    echo "FAIL $1: calls log lacks /$2/: $(cat "$CALLS")"
    fail=$((fail + 1))
  fi
}
expect_not_called() {
  if [ ! -s "$CALLS" ]; then echo "ok   $1"; pass=$((pass + 1)); else
    echo "FAIL $1: unexpected calls: $(cat "$CALLS")"
    fail=$((fail + 1))
  fi
}

bash_json() { jq -n --arg c "$1" --arg d "$2" '{hook_event_name:"PreToolUse",tool_name:"Bash",cwd:$d,tool_input:{command:$c}}'; }
edit_json() { jq -n --arg f "$1" --arg d "$PROJ" '{hook_event_name:"PostToolUse",tool_name:"Edit",cwd:$d,tool_input:{file_path:$f}}'; }

PD="CLAUDE_PROJECT_DIR=$PROJ"
LINT=pre-commit-lint.sh
FMT=post-edit-format.sh

# --- pre-commit-lint.sh ---
check "lint passes on commit" $LINT "$(bash_json 'git commit -m x' "$PROJ")" 0 "" "$PD"
expect_called "lint ran all four checks" 'npm run format:check'
check "go lint failure blocks" $LINT "$(bash_json 'git commit -m x' "$PROJ")" 2 'blocked.*golangci-lint' "$PD" FAKE_GO_RC=1
check "npm lint failure blocks" $LINT "$(bash_json 'git commit -m x' "$PROJ")" 2 'npm run lint' "$PD" FAKE_NPM_FAIL=lint
check "npm check failure blocks" $LINT "$(bash_json 'git commit -m x' "$PROJ")" 2 'npm run check' "$PD" FAKE_NPM_FAIL=check
check "format:check failure blocks" $LINT "$(bash_json 'git commit -m x' "$PROJ")" 2 'npm run format:check' "$PD" FAKE_NPM_FAIL=format:check
check "go lint uses the server build tag" $LINT "$(bash_json 'git commit -m x' "$PROJ")" 0 "" "$PD"
expect_called "golangci-lint got --build-tags server" 'golangci-lint run --build-tags server'
check "cd && git commit blocks" $LINT "$(bash_json 'cd sub && git commit -m x' "$PROJ")" 2 'blocked' "$PD" FAKE_GO_RC=1
check "semicolon git commit blocks" $LINT "$(bash_json 'git add -A; git commit -m x' "$PROJ")" 2 'blocked' "$PD" FAKE_GO_RC=1
check "env prefix git commit blocks" $LINT "$(bash_json 'GIT_AUTHOR_NAME=a git commit -m x' "$PROJ")" 2 'blocked' "$PD" FAKE_GO_RC=1
check "git -C dir commit blocks" $LINT "$(bash_json 'git -C sub commit -m x' "$PROJ")" 2 'blocked' "$PD" FAKE_GO_RC=1
check "git -c k=v commit blocks" $LINT "$(bash_json 'git -c user.name=a commit -m x' "$PROJ")" 2 'blocked' "$PD" FAKE_GO_RC=1
check "commit lints the cwd tree" $LINT "$(bash_json 'git commit -m x' "$PROJ/sub")" 2 'blocked' "$PD" FAKE_GO_RC=1
check "non-commit command ignored" $LINT "$(bash_json 'ls -la' "$PROJ")" 0 "" "$PD" FAKE_GO_RC=1
expect_not_called "non-commit command runs no check"
check "git status mentioning commit ignored" $LINT "$(bash_json 'git log --grep commit' "$PROJ")" 0 "" "$PD" FAKE_GO_RC=1
expect_not_called "git log runs no check"
check "echo of git commit ignored" $LINT "$(bash_json 'echo git commit' "$PROJ")" 0 "" "$PD" FAKE_GO_RC=1
check "separator inside double quotes ignored" $LINT "$(bash_json 'echo "note; git commit -m y"' "$PROJ")" 0 "" "$PD" FAKE_GO_RC=1
check "&& inside quoted PR body ignored" $LINT "$(bash_json 'gh pr create --body "fix && git commit hooks"' "$PROJ")" 0 "" "$PD" FAKE_GO_RC=1
check "separator inside single quotes ignored" $LINT "$(bash_json "echo 'a; git commit -m y'" "$PROJ")" 0 "" "$PD" FAKE_GO_RC=1
check "heredoc body mentioning git commit ignored" $LINT "$(bash_json $'cat > notes.txt <<EOF\ngit commit -m y\nEOF' "$PROJ")" 0 "" "$PD" FAKE_GO_RC=1
check "quoted heredoc body mentioning git commit ignored" $LINT "$(bash_json $'cat > notes.txt <<\'EOF\'\nline\ngit commit -m y\nEOF\nls' "$PROJ")" 0 "" "$PD" FAKE_GO_RC=1
expect_not_called "quoted/heredoc text runs no check"
check "git commit with heredoc message still blocks" $LINT "$(bash_json $'git commit -m "$(cat <<\'EOF\'\nfix; things && more\nEOF\n)"' "$PROJ")" 2 'blocked' "$PD" FAKE_GO_RC=1
check "git commit with quoted multi-line message blocks" $LINT "$(bash_json $'git commit -m "a; b\nc && d"' "$PROJ")" 2 'blocked' "$PD" FAKE_GO_RC=1
check "quoted -C path still followed" $LINT "$(bash_json 'git -C "sub" commit -m "x; y"' "$PROJ")" 2 'blocked' "$PD" FAKE_GO_RC=1
check "commit after heredoc command blocks" $LINT "$(bash_json $'cat > f <<EOF\nx\nEOF\ngit commit -m z' "$PROJ")" 2 'blocked' "$PD" FAKE_GO_RC=1
check "here-string is not a heredoc" $LINT "$(bash_json $'cat <<< word\ngit commit -m z' "$PROJ")" 2 'blocked' "$PD" FAKE_GO_RC=1
check "invalid check timeout warns and falls back" $LINT "$(bash_json 'git commit -m x' "$PROJ")" 0 'not a positive integer' "$PD" ZFAKTURY_HOOK_CHECK_TIMEOUT=abc
check "commit in cwd outside the project does nothing" $LINT "$(bash_json 'git commit -m x' "$OUTSIDE")" 0 "" "$PD" FAKE_GO_RC=1
expect_not_called "outside cwd runs no check"
check "git -C outside path does nothing" $LINT "$(bash_json "git -C $OUTSIDE commit -m x" "$PROJ")" 0 "" "$PD" FAKE_GO_RC=1
expect_not_called "git -C outside runs no check"
check "cd outside && commit does nothing" $LINT "$(bash_json "cd $OUTSIDE && git commit -m x" "$PROJ")" 0 "" "$PD" FAKE_GO_RC=1
expect_not_called "cd outside runs no check"
check "relative .. escape does nothing" $LINT "$(bash_json 'git -C ../outside commit -m x' "$PROJ")" 0 "" "$PD" FAKE_GO_RC=1
expect_not_called "relative escape runs no check"
check "unset CLAUDE_PROJECT_DIR fails clearly" $LINT "$(bash_json 'git commit -m x' "$PROJ")" 1 'CLAUDE_PROJECT_DIR is not set'
check "missing golangci-lint warns, no block" $LINT "$(bash_json 'git commit -m x' "$PROJ")" 0 'golangci-lint not found' "$PD" PATH="$BASE" FAKE_NPM_FAIL=lint
check "missing npm warns, no block" $LINT "$(bash_json 'git commit -m x' "$PROJ")" 0 'npm not found' "$PD" PATH="$BASE" FAKE_GO_RC=1
check "tool timeout warns, no block" $LINT "$(bash_json 'git commit -m x' "$PROJ")" 0 'timed out' "$PD" FAKE_SLEEP=5 ZFAKTURY_HOOK_CHECK_TIMEOUT=1
mv "$PROJ/frontend/node_modules" "$T/node_modules.off"
check "missing node_modules warns, no block" $LINT "$(bash_json 'git commit -m x' "$PROJ")" 0 'node_modules missing' "$PD" FAKE_NPM_FAIL=lint
mv "$T/node_modules.off" "$PROJ/frontend/node_modules"

# --- post-edit-format.sh ---
printf 'package main\nfunc  main( ) {}\n' >"$PROJ/a.go"
check "gofmt formats a go file" $FMT "$(edit_json "$PROJ/a.go")" 0 "" "$PD"
if grep -q 'func main() {}' "$PROJ/a.go"; then echo "ok   go file was formatted"; pass=$((pass + 1)); else
  echo "FAIL go file was not formatted: $(cat "$PROJ/a.go")"
  fail=$((fail + 1))
fi
printf 'package main\nfunc  main( ) {}\n' >"$OUTSIDE/b.go"
check "go file outside the project ignored" $FMT "$(edit_json "$OUTSIDE/b.go")" 0 "" "$PD"
if grep -q 'func  main( )' "$OUTSIDE/b.go"; then echo "ok   outside go file untouched"; pass=$((pass + 1)); else
  echo "FAIL outside go file was modified"
  fail=$((fail + 1))
fi
echo '<p></p>' >"$PROJ/frontend/src/A.svelte"
check "prettier runs on frontend source" $FMT "$(edit_json "$PROJ/frontend/src/A.svelte")" 0 "" "$PD"
expect_called "prettier was called for the svelte file" 'prettier --write .*A\.svelte'
echo 'x' >"$OUTSIDE/c.ts"
check "frontend-like file outside the project ignored" $FMT "$(edit_json "$OUTSIDE/c.ts")" 0 "" "$PD"
expect_not_called "outside ts file runs no formatter"
check "dotdot path escaping the project ignored" $FMT "$(edit_json "$PROJ/../outside/b.go")" 0 "" "$PD"
if grep -q 'func  main( )' "$OUTSIDE/b.go"; then echo "ok   dotdot escape left the outside file untouched"; pass=$((pass + 1)); else
  echo "FAIL dotdot escape modified the outside file"
  fail=$((fail + 1))
fi
ln -s "$OUTSIDE/b.go" "$PROJ/link.go"
check "symlink escaping the project ignored" $FMT "$(edit_json "$PROJ/link.go")" 0 "" "$PD"
if grep -q 'func  main( )' "$OUTSIDE/b.go"; then echo "ok   symlink escape left the outside file untouched"; pass=$((pass + 1)); else
  echo "FAIL symlink escape modified the outside file"
  fail=$((fail + 1))
fi
check "invalid format timeout warns and falls back" $FMT "$(edit_json "$PROJ/a.go")" 0 'not a positive integer' "$PD" ZFAKTURY_HOOK_FORMAT_TIMEOUT=0
check "unset CLAUDE_PROJECT_DIR fails clearly (format)" $FMT "$(edit_json "$PROJ/a.go")" 1 'CLAUDE_PROJECT_DIR is not set'
mv "$PROJ/frontend/node_modules" "$T/node_modules.off"
check "missing prettier warns, no block" $FMT "$(edit_json "$PROJ/frontend/src/A.svelte")" 0 'node_modules missing' "$PD"
mv "$T/node_modules.off" "$PROJ/frontend/node_modules"

echo "passed=$pass failed=$fail"
[ "$fail" -eq 0 ]
