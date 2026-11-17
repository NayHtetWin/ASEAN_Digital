#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
SCHEMA_PATH="${SCRIPT_DIR}/../references/schema.json"
JSON_PATH="${REPO_ROOT}/data/grants.json"

python3 - "${SCHEMA_PATH}" "${JSON_PATH}" <<'PY'
import json
import os
import re
import sys

schema_path, json_path = sys.argv[1], sys.argv[2]

with open(schema_path, "r", encoding="utf-8") as f:
    schema = json.load(f)

required_fields = schema["required"]
props = schema["properties"]
print(f"[OK] Loaded schema ({len(required_fields)} required fields): {schema_path}")

if not os.path.exists(json_path):
    print(f"[SKIP] {json_path} does not exist yet.")
    sys.exit(0)

with open(json_path, "r", encoding="utf-8") as f:
    records = json.load(f)

if not isinstance(records, list):
    print(f"[FAIL] {json_path} must be a JSON array.", file=sys.stderr)
    sys.exit(1)

errors = []
seen_ids = set()
for idx, rec in enumerate(records):
    if not isinstance(rec, dict):
        errors.append(f"Row {idx}: expected object, got {type(rec).__name__}")
        continue
    missing = [k for k in required_fields if k not in rec]
    if missing:
        errors.append(f"Row {idx}: missing required fields {missing}")
    extra = [k for k in rec if k not in props]
    if extra:
        errors.append(f"Row {idx}: unexpected fields {extra}")

    rec_id = rec.get("id", "")
    if rec_id in seen_ids:
        errors.append(f"Row {idx}: duplicate id '{rec_id}'")
    seen_ids.add(rec_id)

    for field, spec in props.items():
        if field not in rec:
            continue
        val = rec[field]
        if not isinstance(val, str):
            errors.append(f"Row {idx}.{field}: expected string")
            continue
        if "minLength" in spec and len(val) < spec["minLength"]:
            errors.append(f"Row {idx}.{field}: value too short")
        if "enum" in spec and val not in spec["enum"]:
            errors.append(f"Row {idx}.{field}: '{val}' not in {spec['enum']}")
        if "pattern" in spec and not re.match(spec["pattern"], val):
            errors.append(f"Row {idx}.{field}: '{val}' failed pattern {spec['pattern']}")

if errors:
    print("[FAIL] Schema validation errors:", file=sys.stderr)
    for err in errors[:25]:
        print(f"  - {err}", file=sys.stderr)
    sys.exit(1)

print(f"[OK] {len(records)} records in {json_path} conform to schema.json")
PY
