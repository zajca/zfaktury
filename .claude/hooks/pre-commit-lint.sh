#!/usr/bin/env bash
# Pre-commit lint hook for Claude Code
# Runs full lint suite before git commit commands

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

if [[ "$COMMAND" == git\ commit* ]]; then
  cd /home/zajca/Code/Me/ZFaktury

  # Go lint
  CGO_ENABLED=0 /home/zajca/.go/bin/golangci-lint run ./... 2>&1
  GO_EXIT=$?

  # Frontend lint + type check + format check
  cd frontend
  npx eslint . 2>&1
  ESLINT_EXIT=$?
  npm run check 2>&1
  CHECK_EXIT=$?
  npx prettier --check . 2>&1
  PRETTIER_EXIT=$?

  if [[ $GO_EXIT -ne 0 || $ESLINT_EXIT -ne 0 || $CHECK_EXIT -ne 0 || $PRETTIER_EXIT -ne 0 ]]; then
    echo "Lint check failed. Fix issues before committing."
    exit 1
  fi
fi
