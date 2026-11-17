#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
WEB_DIR="${REPO_ROOT}/web"

if [[ ! -d "${WEB_DIR}" ]]; then
  echo "[SKIP] ${WEB_DIR} does not exist yet."
  exit 0
fi

INDEX_HTML="${WEB_DIR}/index.html"
if [[ ! -f "${INDEX_HTML}" ]]; then
  echo "[FAIL] Missing ${INDEX_HTML}" >&2
  exit 1
fi

python3 - "${WEB_DIR}" <<'PY'
import os
import sys

web_dir = sys.argv[1]
bundle_text = []
for root, _, files in os.walk(web_dir):
    for name in sorted(files):
        if name.endswith((".html", ".js", ".mjs", ".ts")):
            path = os.path.join(root, name)
            with open(path, "r", encoding="utf-8", errors="ignore") as f:
                bundle_text.append(f.read())

combined = "\n".join(bundle_text)

required_tokens = [
    "grants.json",
    "add_opportunity.yml",
    "1-Triage",
    "2-Go/No-Go",
    "3-Partnering",
    "4-Drafting",
    "5-Submitted",
    "6-Awarded/Expired",
]

missing = [tok for tok in required_tokens if tok not in combined]
if missing:
    print(f"[FAIL] web/ is missing expected UI/data bindings: {missing}", file=sys.stderr)
    sys.exit(1)

print(f"[OK] web/ verified: all {len(required_tokens)} data-binding and pipeline tokens present.")
PY
