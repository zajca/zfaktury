#!/usr/bin/env bash
# Post-tool hook: auto-format files after Claude edits them
# Runs gofmt on .go files, prettier on frontend files

INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

if [ -z "$FILE_PATH" ]; then
  exit 0
fi

cd /home/zajca/Code/Me/ZFaktury

case "$FILE_PATH" in
  *.go)
    gofmt -w "$FILE_PATH" 2>/dev/null
    ;;
  */frontend/src/*.svelte|*/frontend/src/*.ts|*/frontend/src/*.js)
    cd frontend
    RELATIVE=${FILE_PATH#*frontend/}
    npx prettier --write "$RELATIVE" 2>/dev/null
    ;;
esac
