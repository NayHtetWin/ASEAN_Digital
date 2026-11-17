---
name: asean-backend-pipeline
description: Build, extend, and maintain the Go (Golang) backend scraper pipeline (cmd/fetcher/, internal/, config.yaml, data/grants.json, data/grants.csv) and GitHub Actions workflows for tracking ASEAN digital economy, DEFA, and workforce grants/RFPs. Use when adding public sources, writing Go RSS/HTML/chromedp parsers, extracting open/close deadlines, or syncing GitHub Issues.
---

# ASEAN Backend Pipeline

Build and maintain the Go scraper, deadline engine, dual CSV/JSON storage, and GitHub Actions / Issue sync for the ASEAN Digital Economy & Workforce Grant/RFP Tracker.

## 1. Scope Boundary

- **Owns:** `cmd/`, `internal/`, `config.yaml`, `data/`, and `.github/workflows/`.
- **Strict Boundary:** Never create, edit, or delete files in `web/`. Frontend changes belong to `asean-frontend-dashboard`.
- **Data Contract:** Every emitted `Opportunity` record must strictly conform to [schema.json](./references/schema.json).

## 2. Plug-and-Play Source Configuration

All public donor and procurement sources live in `config.yaml` (see [config-spec.md](./references/config-spec.md)).
Each source entry must define:
- `frequency`: `"daily"` or `"weekly"`
- `type`: `"rss"`, `"html"`, or `"chromedp"`
- `item_selector`: CSS selector for each opportunity item (for `html` / `chromedp`)
- `link_selector`: CSS selector for the canonical URL link within the item
- `force_region`: Optional region override (e.g., `"ASEAN"`) when the portal omits explicit regional keywords

## 3. Concurrency & Scraper Safety

- Fetch configured sources concurrently using goroutines and `sync.WaitGroup`.
- Enforce a **15-second timeout** on `http.Client` per source (and context deadlines for headless browser runs) so one slow donor portal never stalls the pipeline.
- **Parser Selection:**
  - `rss`: Parse feeds with `github.com/mmcdole/gofeed`.
  - `html`: Parse static HTML with `github.com/PuerkitoBio/goquery`.
  - `chromedp`: Use `github.com/chromedp/chromedp` only when portals (e.g., DFAT AusConnect, UNGM) block standard HTTP requests or require JavaScript rendering.
- Generate `id` as the first **10 lowercase hex characters** of the MD5 hash of the canonical URL.

## 4. Deadline Engine

- Extract `open_date` (`YYYY-MM-DD`) and `close_date` (`YYYY-MM-DD` or `"TBD"`) using regex extraction paired with multi-layout Go `time.Parse`.
- Recalculate `deadline_status` for **all** active records on every pipeline run relative to the current date:
  - Remaining days `< 0`: `"Expired"` (and transition `pipeline_stage` to `"6-Awarded/Expired"` as appropriate)
  - Remaining days `<= 7` (and `>= 0`): `"Closing Soon (Xd left)"`
  - Remaining days `> 7` (or `"TBD"`): `"Open (Xd left)"` (for `"TBD"`, handle per project convention while matching the schema pattern or keeping `"Open (0d left)"`—prefer explicit numeric days when `close_date` is known)

## 5. Dual Storage Atomicity & GitHub Issue Sync

- Every run must write **identical** records (same count, same IDs, same field values) atomically to:
  1. `data/grants.json` (pretty-printed, indented JSON array)
  2. `data/grants.csv` (RFC 4180 CSV with header row matching `schema.json` field order)
- Sync newly discovered opportunities and newly expired statuses to GitHub Issues using the GitHub CLI/API in `.github/workflows/`.

## 6. Verification

After any backend change:
1. Validate data contract compliance with [verify_schema.sh](./scripts/verify_schema.sh).
2. Run the full test and parity harness in `asean-pipeline-testing` (`../asean-pipeline-testing/scripts/run_checks.sh`).
