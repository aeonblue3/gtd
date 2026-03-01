#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="${HOME}/.local/bin"

echo "Building gtd binary..."
mkdir -p "${PROJECT_ROOT}/bin"
go build -o "${PROJECT_ROOT}/bin/gtd" "${PROJECT_ROOT}/cmd"

echo "Installing to ${BIN_DIR}/gtd"
mkdir -p "${BIN_DIR}"
cp "${PROJECT_ROOT}/bin/gtd" "${BIN_DIR}/gtd"
chmod +x "${BIN_DIR}/gtd"

if ! echo "${PATH}" | tr ':' '\n' | rg -q "^${BIN_DIR}$"; then
  echo "Add ${BIN_DIR} to your PATH to run 'gtd' globally."
fi

echo "Running first-time bootstrap..."
GTD_DATA_PATH="${HOME}/.gtd" "${BIN_DIR}/gtd" sync --init || true
echo "Installed. Try: gtd help"
