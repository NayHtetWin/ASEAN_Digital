#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
SCHEMA_PATH="${REPO_ROOT}/.agents/skills/asean-backend-pipeline/references/schema.json"
JSON_PATH="${REPO_ROOT}/data/grants.json"
CSV_PATH="${REPO_ROOT}/data/grants.csv"
WEB_VERIFY="${REPO_ROOT}/.agents/skills/asean-frontend-dashboard/scripts/verify_web.sh"

cd "${REPO_ROOT}"

echo "=== 1. Go Vet & Unit Tests ==="
if [[ -f "${REPO_ROOT}/go.mod" ]]; then
  if ! command -v go >/dev/null 2>&1; then
    echo "[FAIL] go.mod found, but 'go' binary is not installed or not on PATH." >&2
    exit 1
  fi
  echo "[RUN] go vet ./..."
  go vet ./...
  echo "[RUN] go test -v -race ./..."
  go test -v -race ./...
  echo "[OK] Go vet and tests passed."
else
  echo "[SKIP] go.mod not found; skipping go vet and go test."
fi

echo ""
echo "=== 2. Schema & CSV/JSON Parity Checks ==="
python3 - "${SCHEMA_PATH}" "${JSON_PATH}" "${CSV_PATH}" <<'PY'
import csv
import json
import os
import re
import sys

schema_path, json_path, csv_path = sys.argv[1], sys.argv[2], sys.argv[3]

if not os.path.exists(schema_path):
    print(f"[FAIL] Missing schema file: {schema_path}", file=sys.stderr)
    sys.exit(1)

with open(schema_path, "r", encoding="utf-8") as f:
    schema = json.load(f)

required_fields = schema.get("required", [])
props = schema.get("properties", {})
expected_fields = [
    "id",
    "type",
    "title",
    "source",
    "open_date",
    "close_date",
    "pipeline_stage",
    "deadline_status",
    "url",
]
if set(required_fields) != set(expected_fields):
    print(f"[FAIL] Schema required fields mismatch: {required_fields}", file=sys.stderr)
    sys.exit(1)

print(f"[OK] Schema contract valid: {schema_path}")

json_exists = os.path.exists(json_path)
csv_exists = os.path.exists(csv_path)

if not json_exists and not csv_exists:
    print("[SKIP] data/grants.json and data/grants.csv not generated yet; skipping parity check.")
    sys.exit(0)

if json_exists != csv_exists:
    print(
        f"[FAIL] Dual storage mismatch: json_exists={json_exists}, csv_exists={csv_exists}",
        file=sys.stderr,
    )
    sys.exit(1)

with open(json_path, "r", encoding="utf-8") as f:
    json_rows = json.load(f)

if not isinstance(json_rows, list):
    print("[FAIL] data/grants.json must contain a JSON array.", file=sys.stderr)
    sys.exit(1)

with open(csv_path, "r", encoding="utf-8", newline="") as f:
    reader = csv.DictReader(f)
    if reader.fieldnames is None:
        print("[FAIL] data/grants.csv is missing a header row.", file=sys.stderr)
        sys.exit(1)
    missing_cols = [col for col in required_fields if col not in reader.fieldnames]
    if missing_cols:
        print(f"[FAIL] data/grants.csv missing columns: {missing_cols}", file=sys.stderr)
        sys.exit(1)
    csv_rows = list(reader)

if len(json_rows) != len(csv_rows):
    print(
        f"[FAIL] Row count mismatch: JSON={len(json_rows)} vs CSV={len(csv_rows)}",
        file=sys.stderr,
    )
    sys.exit(1)

errors = []
json_ids = set()
csv_ids = set()
json_by_id = {}

for idx, rec in enumerate(json_rows):
    if not isinstance(rec, dict):
        errors.append(f"JSON row {idx}: expected object")
        continue
    for field in required_fields:
        if field not in rec:
            errors.append(f"JSON row {idx}: missing field '{field}'")
    rec_id = rec.get("id", "")
    if rec_id in json_ids:
        errors.append(f"JSON row {idx}: duplicate id '{rec_id}'")
    json_ids.add(rec_id)
    json_by_id[rec_id] = rec

    for field, spec in props.items():
        if field not in rec:
            continue
        val = rec[field]
        if not isinstance(val, str):
            errors.append(f"JSON row {idx}.{field}: expected string")
            continue
        if "enum" in spec and val not in spec["enum"]:
            errors.append(f"JSON row {idx}.{field}: '{val}' not in {spec['enum']}")
        if "pattern" in spec and not re.match(spec["pattern"], val):
            errors.append(f"JSON row {idx}.{field}: '{val}' failed pattern {spec['pattern']}")

for idx, rec in enumerate(csv_rows):
    rec_id = rec.get("id", "")
    if rec_id in csv_ids:
        errors.append(f"CSV row {idx}: duplicate id '{rec_id}'")
    csv_ids.add(rec_id)
    if rec_id in json_by_id:
        for field in required_fields:
            if str(json_by_id[rec_id].get(field, "")) != str(rec.get(field, "")):
                errors.append(
                    f"Parity mismatch for id '{rec_id}' field '{field}': "
                    f"JSON={json_by_id[rec_id].get(field)!r} vs CSV={rec.get(field)!r}"
                )

if json_ids != csv_ids:
    errors.append(
        f"ID set mismatch: only_in_json={sorted(json_ids - csv_ids)[:5]}, "
        f"only_in_csv={sorted(csv_ids - json_ids)[:5]}"
    )

if errors:
    print("[FAIL] Schema/parity check failed:", file=sys.stderr)
    for err in errors[:25]:
        print(f"  - {err}", file=sys.stderr)
    sys.exit(1)

print(f"[OK] Parity & schema check passed for {len(json_rows)} records.")
PY

echo ""
echo "=== 3. Frontend Verification ==="
if [[ -x "${WEB_VERIFY}" ]]; then
  "${WEB_VERIFY}"
else
  echo "[SKIP] ${WEB_VERIFY} not executable or missing."
fi

echo ""
echo "[PASS] All harness checks completed."
