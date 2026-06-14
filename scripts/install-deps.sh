#!/usr/bin/env bash
# Install anchor's recommended CLI dependencies via Homebrew.
set -euo pipefail

if ! command -v brew >/dev/null 2>&1; then
  echo "Homebrew is required. Install from https://brew.sh"
  exit 1
fi

PACKAGES=(awscli kubectl fzf stern k9s helm)

echo "→ Installing anchor dependencies: ${PACKAGES[*]}"
brew install "${PACKAGES[@]}"
echo ""
echo "✓ Dependencies installed. Run: anchor onboard"
