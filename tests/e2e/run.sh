#!/usr/bin/env bash
# ==============================================================================
# GoRoute E2E Test Suite Runner
# Opaque-box Parity & Verification Test Suite (Tiers 1-4)
# ==============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

cd "${PROJECT_ROOT}"

echo "======================================================================"
echo "⚡ GoRoute Opaque-Box E2E Test Suite"
echo "   Directory: ${PROJECT_ROOT}"
echo "   Date:      $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
echo "======================================================================"

# Pre-compile the GoRoute gateway binary once to optimize test execution
TMP_BIN=""
if [ -z "${GOROUTE_BIN:-}" ]; then
  TMP_BIN=$(mktemp /tmp/goroute-e2e-bin-XXXXXX)
  echo "🔨 Compiling GoRoute daemon to ${TMP_BIN}..."
  go build -o "${TMP_BIN}" ./cmd/goroute
  export GOROUTE_BIN="${TMP_BIN}"
  trap 'rm -f "${TMP_BIN}"' EXIT
else
  echo "ℹ Using existing GOROUTE_BIN=${GOROUTE_BIN}"
fi

START_TIME=$(date +%s)

echo "🚀 Executing E2E tests in ./tests/e2e/..."
go test -v -tags=e2e ./tests/e2e/... "$@"

DURATION=$(( $(date +%s) - START_TIME ))
echo "======================================================================"
echo "✅ All E2E test suites passed cleanly in ${DURATION}s."
echo "======================================================================"
