---
name: asean-fast-tester
description: Fast, high-value critical-path QA subagent designed for small models. Tests ONLY the 4 highest-risk failure points: (1) Open/Close date regex & deadline status math, (2) ASEAN keyword + Grant/RFP classification logic, (3) CSV-to-JSON schema parity & deduplication, and (4) Frontend data-binding keys. Do NOT test UI styling, third-party libraries, or live external websites.
---

# ASEAN Fast Tester (Critical-Path QA)

Execute ONLY the 4 critical-path checks below. Do NOT explore unrelated files, test UI CSS/styling, test third-party libraries, or make live HTTP requests.

## Strict Rules

1. **NO Live Network Calls:** Never fetch live donor URLs (e.g., DFAT AusConnect, ADB, UNGM). Test ONLY against static string snippets in `cmd/fetcher/main_test.go`.
2. **Test ONLY the Big 4 Value Paths:**
   - **Path A (Deadline Math & Date Extraction):**
     - `extractCloseDate()` must parse `Apply by 26/11/2026`, `Deadline: 5 Dec 2026`, and return `"TBD"` when missing.
     - `computeDeadlineStatus()` must return `"Expired"` (`<0 days`), `"Closing Soon (Xd left)"` (`0-7 days`), and `"Open (Xd left)"` (`>7 days`).
   - **Path B (ASEAN + Grant/RFP Filter):**
     - `isRelevant()` must reject non-ASEAN or non-digital/workforce items unless `force_region: true` is set.
     - `classifyOpportunity()` must distinguish `"Tender / RFP"` (e.g., `"Request for Proposal: Rapid Assessment of ICT"`) from `"Grant / CFP"`.
   - **Path C (CSV & JSON Storage Integrity):**
     - `data/grants.json` and `data/grants.csv` must have identical record counts, zero duplicate `id` hashes, and all 9 required fields: `id`, `type`, `title`, `source`, `open_date`, `close_date`, `pipeline_stage`, `deadline_status`, `url`.
   - **Path D (Frontend Contract Check):**
     - `web/` must reference `grants.json` and bind to exact field names: `close_date`, `deadline_status`, `pipeline_stage`, `type`.

## Execution Protocol

1. Run:
   ```bash
   bash .agents/skills/asean-fast-tester/scripts/verify_critical.sh
   ```
2. If a check fails, inspect ONLY the failing file/line reported.
3. Output a compact `<15 line` Pass/Fail summary with exact `file:line` references for any failure.
