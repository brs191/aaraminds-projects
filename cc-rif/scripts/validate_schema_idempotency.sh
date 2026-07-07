#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

SCHEMA_FILES=(
  "${ROOT_DIR}/data/schema/age_schema.sql"
  "${ROOT_DIR}/data/schema/relational_schema.sql"
)

MIGRATION_FILES=(
  "${ROOT_DIR}/data/migrations/migration_phase2.sql"
  "${ROOT_DIR}/data/migrations/migration_pgvector.sql"
  "${ROOT_DIR}/data/migrations/migration_fts.sql"
)

ALL_FILES=("${SCHEMA_FILES[@]}" "${MIGRATION_FILES[@]}")

usage() {
  cat <<'USAGE'
Usage: scripts/validate_schema_idempotency.sh [--dry-run]

Validates deterministic migration order and idempotency for:
  data/schema/*.sql + data/migrations/*.sql

Modes:
  --dry-run  Static checks only (file presence + idempotency guards)
             Does not require DATABASE_URL.

Without --dry-run, DATABASE_URL must be set and psql must be available.
The script applies schema+migrations twice; second pass proves idempotent re-apply.
USAGE
}

assert_files_exist() {
  local missing=0
  for file in "${ALL_FILES[@]}"; do
    if [[ ! -f "${file}" ]]; then
      echo "[schema-idempotency] missing required SQL file: ${file}" >&2
      missing=1
    fi
  done
  if [[ "${missing}" -ne 0 ]]; then
    exit 1
  fi
}

assert_guard_patterns() {
  local missing=0
  for file in "${ALL_FILES[@]}"; do
    if ! rg -q "IF NOT EXISTS|ADD COLUMN IF NOT EXISTS|CREATE OR REPLACE|DROP TRIGGER IF EXISTS|DO \\\$\\\$" "${file}"; then
      echo "[schema-idempotency] no idempotency guard pattern found: ${file}" >&2
      missing=1
    fi
  done
  if [[ "${missing}" -ne 0 ]]; then
    exit 1
  fi
}

run_pass() {
  local pass_label="$1"
  for file in "${ALL_FILES[@]}"; do
    echo "[schema-idempotency] ${pass_label}: applying $(basename "${file}")"
    psql "${DATABASE_URL}" -X -v ON_ERROR_STOP=1 -f "${file}" >/dev/null
  done
}

main() {
  local dry_run=0

  if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
  fi

  if [[ "${1:-}" == "--dry-run" ]]; then
    dry_run=1
  elif [[ -n "${1:-}" ]]; then
    echo "Unknown argument: ${1}" >&2
    usage
    exit 1
  fi

  assert_files_exist
  assert_guard_patterns

  if [[ "${dry_run}" -eq 1 ]]; then
    echo "[schema-idempotency] dry-run checks passed (files + guard patterns)."
    exit 0
  fi

  if ! command -v psql >/dev/null 2>&1; then
    echo "[schema-idempotency] psql is required for execution mode." >&2
    exit 1
  fi

  if [[ -z "${DATABASE_URL:-}" ]]; then
    echo "[schema-idempotency] DATABASE_URL is required for execution mode." >&2
    echo "[schema-idempotency] Tip: run with --dry-run for static validation only." >&2
    exit 1
  fi

  run_pass "pass-1"
  run_pass "pass-2"

  echo "[schema-idempotency] execution passed: schema + migrations re-applied without errors."
}

main "$@"
