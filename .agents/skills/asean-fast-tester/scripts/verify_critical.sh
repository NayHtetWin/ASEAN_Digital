#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
cd "${REPO_ROOT}"

echo "=== 1. Go Vet & Critical Tests ==="
if [[ -f "${REPO_ROOT}/go.mod" ]]; then
  if ! command -v go >/dev/null 2>&1; then
    echo "[FAIL] go.mod exists but 'go' binary not found on PATH." >&2
    exit 1
  fi
  go vet ./...
  go test -short -v ./cmd/fetcher -run "TestCritical"
  echo "[OK] Go vet and TestCritical passed."
else
  echo "[SKIP] go.mod not found; skipping go vet and go test."
fi

echo "=== 2. Config & CSV/JSON Parity ==="
if [[ -f "${REPO_ROOT}/config.yaml" ]]; then
  if grep -q "sources:" "${REPO_ROOT}/config.yaml"; then
    echo "[OK] config.yaml contains 'sources:'."
  else
    echo "[FAIL] config.yaml missing 'sources:' key." >&2
    exit 1
  fi
else
  echo "[SKIP] config.yaml not found."
fi

JSON_FILE="${REPO_ROOT}/data/grants.json"
CSV_FILE="${REPO_ROOT}/data/grants.csv"
if [[ -f "${JSON_FILE}" && -f "${CSV_FILE}" ]]; then
  json_count="$(jq 'length' "${JSON_FILE}")"
  csv_count="$(awk 'NR>1 && NF>0 {c++} END {print c+0}' "${CSV_FILE}")"
  if [[ "${json_count}" -ne "${csv_count}" ]]; then
    echo "[FAIL] Row count mismatch: grants.json=${json_count}, grants.csv=${csv_count}" >&2
    exit 1
  fi
  dup_count="$(jq '[.[].id] | length - (unique | length)' "${JSON_FILE}")"
  if [[ "${dup_count}" -ne 0 ]]; then
    echo "[FAIL] Duplicate id hashes found in grants.json: ${dup_count}" >&2
    exit 1
  fi
  missing_fields="$(jq '[.[] | select(.id == null or .type == null or .title == null or .source == null or .open_date == null or .close_date == null or .pipeline_stage == null or .deadline_status == null or .url == null)] | length' "${JSON_FILE}")"
  if [[ "${missing_fields}" -ne 0 ]]; then
    echo "[FAIL] ${missing_fields} record(s) missing required schema fields in grants.json." >&2
    exit 1
  fi
  echo "[OK] Dual storage parity verified (${json_count} records, 0 duplicates)."
elif [[ -f "${JSON_FILE}" || -f "${CSV_FILE}" ]]; then
  echo "[FAIL] Only one of data/grants.json or data/grants.csv exists." >&2
  exit 1
else
  echo "[SKIP] data/grants.json and data/grants.csv not found."
fi

echo "=== 3. Frontend Data-Binding Keys ==="
if [[ -d "${REPO_ROOT}/web" ]]; then
  for key in grants.json close_date deadline_status pipeline_stage type; do
    if ! grep -rq "${key}" "${REPO_ROOT}/web"; then
      echo "[FAIL] web/ missing reference to '${key}'." >&2
      exit 1
    fi
  done
  echo "[OK] web/ references grants.json, close_date, deadline_status, pipeline_stage, type."
else
  echo "[SKIP] web/ directory not found."
fi

echo "[PASS] verify_critical.sh completed."
