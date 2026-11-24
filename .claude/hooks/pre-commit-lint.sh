#!/usr/bin/env bash
# PreToolUse hook (Bash): runs CI's lint checks before a `git commit`.
#
# The Bash input is first stripped of heredoc bodies and quoted strings, then split
# on `&&`, `||`, `;`, `|`, `&` and newlines into simple commands. A commit is a
# simple command `git [global options] commit`. `cd <dir>` and `git -C <dir>`
# change the tree the commit targets; the starting directory is `cwd` from the
# hook JSON. The tree is linted only when it lies inside $CLAUDE_PROJECT_DIR,
# otherwise the hook does nothing.
#
# Commit forms that are NOT detected (the commit then goes through unlinted):
# commands inside quotes or heredocs (`bash -c "git commit"`, `eval`, `xargs`),
# `command git` / `env git` / `sudo git`, shell aliases and functions, a path with
# spaces after `cd` or `git -C`, `git --git-dir/--work-tree` pointing at another
# tree, and a `cd` made inside a subshell that leaks into later commands.
#
# Checks (same as the CI Go and Frontend jobs):
#   golangci-lint run --build-tags server
#   npm run lint | npm run check | npm run format:check   (in frontend/)
#
# Exit codes: 2 blocks the commit with the reason on stderr (a failed check).
# 0 lets it through. A missing tool or frontend/node_modules or a timed-out check
# does not block: the warning goes to stderr and, because Claude Code shows stderr
# of an exit-0 hook only in the debug log, also to stdout as a `systemMessage`
# JSON object, which the user sees.
# 1 (non-blocking) reports a broken setup such as an unset CLAUDE_PROJECT_DIR.

# Default upper bound in seconds for each single check.
DEFAULT_CHECK_TIMEOUT=240
# Lines of a failed check's output shown in the block reason.
REASON_LINES=40

warnings=()
warn() {
  printf 'pre-commit-lint: warning: %s\n' "$*" >&2
  warnings+=("pre-commit-lint: $*")
}

# finish <rc>: prints the collected warnings as a systemMessage, then exits.
finish() {
  if [ "${#warnings[@]}" -gt 0 ]; then
    if command -v jq >/dev/null 2>&1; then
      jq -n --arg m "$(printf '%s\n' "${warnings[@]}")" '{systemMessage: $m}'
    else
      printf '{"systemMessage":"pre-commit-lint: jq not found, lint skipped"}\n'
    fi
  fi
  exit "$1"
}

CHECK_TIMEOUT="${ZFAKTURY_HOOK_CHECK_TIMEOUT:-$DEFAULT_CHECK_TIMEOUT}"
if ! [[ $CHECK_TIMEOUT =~ ^[1-9][0-9]*$ ]]; then
  warn "ZFAKTURY_HOOK_CHECK_TIMEOUT='$CHECK_TIMEOUT' is not a positive integer, using $DEFAULT_CHECK_TIMEOUT"
  CHECK_TIMEOUT=$DEFAULT_CHECK_TIMEOUT
fi

if [ -z "${CLAUDE_PROJECT_DIR:-}" ]; then
  echo "pre-commit-lint: CLAUDE_PROJECT_DIR is not set" >&2
  exit 1
fi
PROJECT=$(realpath -e -- "$CLAUDE_PROJECT_DIR" 2>/dev/null) || {
  echo "pre-commit-lint: CLAUDE_PROJECT_DIR is not an existing directory" >&2
  exit 1
}

if ! command -v jq >/dev/null 2>&1; then
  warn "jq not found, skipping lint"
  finish 0
fi

input=$(cat)
command_line=$(jq -r '.tool_input.command // empty' <<<"$input" 2>/dev/null)
cwd=$(jq -r '.cwd // empty' <<<"$input" 2>/dev/null)
[ -n "$command_line" ] || exit 0
[ -n "$cwd" ] || cwd="$PROJECT"

# Cheap exit for the vast majority of Bash calls.
case "$command_line" in
  *commit*) ;;
  *) exit 0 ;;
esac

# resolve_dir <base> <path>: prints the absolute path, or nothing when the path
# cannot be known statically (variables, ~, command substitution, "cd -").
resolve_dir() {
  local base=$1 path=$2
  case "$path" in
    '' | - | '~'* | *'$'* | *'`'*) return 0 ;;
    /*) realpath -m -- "$path" ;;
    *) [ -n "$base" ] && realpath -m -- "$base/$path" ;;
  esac
}

targets=()
cur=$cwd
# 1. awk drops heredoc bodies (`<<[-]WORD` up to the line holding WORD; `<<<` is
#    a here-string, not a heredoc).
# 2. awk walks the quotes in pairs: a quoted string that holds anything beyond
#    path characters becomes Q; a plain quoted word such as "sub" is unquoted so it
#    stays usable as a `cd`/`-C` argument.
# 3. sed splits on command separators.
segments=$(printf '%s\n' "$command_line" |
  awk '
    skip { t = $0; sub(/^\t+/, "", t); if (t == tag) skip = 0; next }
    {
      print
      if (match($0, /<<-?[ \t]*["\047]?[A-Za-z_][A-Za-z0-9_]*/) && substr($0, RSTART - 1, 1) != "<") {
        tag = substr($0, RSTART, RLENGTH)
        sub(/^<<-?[ \t]*["\047]?/, "", tag)
        skip = 1
      }
    }' |
  awk -v RS='\001' '
    {
      s = $0; n = length(s); out = ""; i = 1
      while (i <= n) {
        c = substr(s, i, 1)
        if (c == "\"" || c == "\047") {
          j = index(substr(s, i + 1), c)
          if (j == 0) { out = out substr(s, i + 1); break }
          body = substr(s, i + 1, j - 1)
          out = out (body ~ /^[A-Za-z0-9_.\/-]*$/ ? body : "Q")
          i = i + j + 1
        } else { out = out c; i++ }
      }
      printf "%s", out
    }' |
  sed -E 's/(&&|\|\||;|\||&)/\n/g')
while IFS= read -r seg; do
  seg=${seg//[()]/ }
  read -r -a w <<<"$seg"
  n=${#w[@]}
  i=0
  while [ "$i" -lt "$n" ] && [[ ${w[i]} =~ ^[A-Za-z_][A-Za-z0-9_]*= ]]; do i=$((i + 1)); done
  [ "$i" -lt "$n" ] || continue
  case "${w[i]}" in
    cd)
      j=$((i + 1))
      while [ "$j" -lt "$n" ] && [[ ${w[j]} == -[LPe@] || ${w[j]} == -- ]]; do j=$((j + 1)); done
      cur=$(resolve_dir "$cur" "${w[j]:-}")
      ;;
    git)
      gdir=$cur
      j=$((i + 1))
      while [ "$j" -lt "$n" ]; do
        case "${w[j]}" in
          -C)
            gdir=$(resolve_dir "$gdir" "${w[j + 1]:-}")
            j=$((j + 2))
            ;;
          -c | --git-dir | --work-tree | --namespace | --super-prefix | --config-env) j=$((j + 2)) ;;
          -*) j=$((j + 1)) ;;
          *) break ;;
        esac
      done
      if [ "$j" -lt "$n" ] && [ "${w[j]}" = commit ] && [ -n "$gdir" ]; then
        targets+=("$gdir")
      fi
      ;;
  esac
done <<<"$segments"

[ "${#targets[@]}" -gt 0 ] || exit 0

failed=()
reasons=""

# run_check <name> <dir> <command...>: records a failure, warns on a timeout.
run_check() {
  local name=$1 dir=$2 out rc
  shift 2
  out=$(cd "$dir" && timeout "$CHECK_TIMEOUT" "$@" 2>&1)
  rc=$?
  if [ "$rc" -eq 124 ]; then
    warn "$name timed out after ${CHECK_TIMEOUT}s, not blocking"
  elif [ "$rc" -ne 0 ]; then
    failed+=("$name")
    reasons+=$'\n'"--- $name (exit $rc) ---"$'\n'"$(printf '%s\n' "$out" | tail -n "$REASON_LINES")"
  fi
}

lint_tree() {
  local root=$1
  if [ -f "$root/go.mod" ]; then
    if command -v golangci-lint >/dev/null 2>&1; then
      run_check "golangci-lint" "$root" golangci-lint run --build-tags server
    else
      warn "golangci-lint not found in PATH, skipping Go lint"
    fi
  fi
  if [ -f "$root/frontend/package.json" ]; then
    if ! command -v npm >/dev/null 2>&1; then
      warn "npm not found in PATH, skipping frontend lint"
    elif [ ! -d "$root/frontend/node_modules" ]; then
      warn "frontend/node_modules missing (run npm ci), skipping frontend lint"
    else
      run_check "npm run lint" "$root/frontend" npm run lint
      run_check "npm run check" "$root/frontend" npm run check
      run_check "npm run format:check" "$root/frontend" npm run format:check
    fi
  fi
}

declare -A seen=()
for t in "${targets[@]}"; do
  tree=$(realpath -m -- "$t")
  case "$tree" in
    "$PROJECT" | "$PROJECT"/*) ;;
    *) continue ;;
  esac
  [ -d "$tree" ] || continue
  top=$(git -C "$tree" rev-parse --show-toplevel 2>/dev/null) || continue
  top=$(realpath -- "$top") || continue
  case "$top" in
    "$PROJECT" | "$PROJECT"/*) root=$top ;;
    *) root=$PROJECT ;;
  esac
  [ -z "${seen[$root]:-}" ] || continue
  seen[$root]=1
  lint_tree "$root"
done

if [ "${#failed[@]}" -gt 0 ]; then
  {
    echo "Commit blocked: lint check failed (${failed[*]}). Fix the issues and commit again."
    printf '%s\n' "$reasons"
  } >&2
  exit 2
fi
finish 0
