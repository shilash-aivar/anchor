#!/usr/bin/env bash
# Cursor CLI status line — shows anchor AWS account + EKS cluster (like git branch).
# Add to ~/.cursor/cli-config.json:
#   "statusLine": { "type": "command", "command": "/path/to/anchor/scripts/cursor-statusline.sh" }
set -euo pipefail

ANCHOR="${ANCHOR_BIN:-anchor}"
if ! command -v "$ANCHOR" >/dev/null 2>&1; then
  ANCHOR="${HOME}/.local/bin/anchor"
fi
if ! command -v "$ANCHOR" >/dev/null 2>&1; then
  exit 0
fi

line="$("$ANCHOR" prompt --format plain 2>/dev/null || true)"
if [[ -z "$line" ]]; then
  exit 0
fi

printf '%s' "$line"
