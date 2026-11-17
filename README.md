# ASEAN Digital Economy & Workforce Grant/RFP Tracker

A lean, serverless tracker for discovering, qualifying, and monitoring funding calls, grants, and consulting RFPs across the **ASEAN Digital Economy** (DEFA, DPI, GovTech, ICT, AI/data governance, MSME transformation) and **Workforce Planning** (TVET, upskilling, future of work, labor market systems).

## Architecture

- **`config.yaml`** — Keywords (`regions`, `themes`), classification markers (`rfp_markers`, `grant_markers`), RSS/HTML `sources` (`daily` / `weekly`), and `manual_entries` for login-walled portals (e.g., DFAT AusConnect).
- **`main.go`** — Single-file Go pipeline with `-schedule [daily|weekly|all]`, concurrent scraping (`15s` timeout via `gofeed` and `goquery`), qualification & classification (`Grant` vs. `RFP/Tender`), UTC deadline status calculation, and triple flat-file output.
- **`data/`** — Synchronized, deduplicated outputs sorted by `open_date` descending:
  - `data/grants.json` (formatted JSON array)
  - `data/grants.csv` (spreadsheet/GitHub-ready CSV)
  - `data/grants.js` (`window.GRANTS_DATA` wrapper for zero-server `file://` loading)
- **`web/index.html`** — Serverless Tailwind + vanilla JS executive dashboard with 4 KPI cards, live search, category/status filters, and deadline sorting. Works via direct `file://` double-click or GitHub Pages.
- **`.github/workflows/sync.yml`** — Automated daily (`Mon–Sat`), weekly (`Sun`), and manual (`workflow_dispatch`) sync committing updated `data/` files.

## Quick Start

### 1. Run the Tracker
```bash
go run main.go -schedule all
```
Options for `-schedule`: `daily`, `weekly`, or `all` (default).

### 2. Open the Dashboard
Open `web/index.html` directly in your browser (double-click or `open web/index.html`) — no local HTTP server required.

### 3. Run Unit Tests
```bash
go vet ./...
go test -v ./...
```

### 4. Add a Manual Opportunity
Append an entry under `manual_entries` in `config.yaml` with `title`, `type` (`Grant` or `RFP/Tender`), `source`, `open_date`, `close_date` (`YYYY-MM-DD` or `TBD`), and `url`, then re-run `go run main.go -schedule all`.

## Curated Sources

### Active Automated Sources (`config.yaml`)
All active sources are verified publicly accessible (HTTP 200) without login walls or anti-bot blocks:

| Source | Type | Schedule | URL | Notes |
| :--- | :--- | :--- | :--- | :--- |
| **FundsforNGOs** | RSS | `daily` | https://www2.fundsforngos.org/feed/ | Global & Asia-Pacific NGO grants, fellowships, CFPs |
| **ReliefWeb** | RSS | `daily` | https://reliefweb.int/updates/rss.xml | Humanitarian & development updates across ASEAN |
| **UNDP Procurement Notices** | HTML | `daily` | https://procurement-notices.undp.org/ | UN RFPs, RFQs, and ICT/digital consultancies |
| **ASEAN Secretariat** | RSS | `weekly` | https://asean.org/feed/ | Official ASEAN announcements (`force_region: ASEAN`) |
| **ASEAN Foundation** | HTML | `weekly` | https://www.aseanfoundation.org/ | Regional digital & youth grants (`force_region: ASEAN`) |

### Manual / Login-Walled Sources (`manual_entries`)
Portals that require authentication, JavaScript rendering, or block automated scrapers are tracked via `manual_entries` in `config.yaml`:
- **DFAT AusConnect** (`https://ausconnect.dfat.gov.au/`) — Australian government procurement & ICT rapid assessments
- **UNGM (UN Global Marketplace)** (`https://www.ungm.org/`) — Filtered UN agency tenders for Asia-Pacific
- **ADB Business Center / ADBI** (`https://www.adb.org/`) — Returns HTTP 403 to automated HTTP clients
- **DevelopmentAid** (`https://www.developmentaid.org/`) — Returns HTTP 403 to automated RSS clients

