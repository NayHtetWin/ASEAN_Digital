---
name: asean-frontend-dashboard
description: Build, style, and update the frontend dashboard and proposal pipeline UI in web/ consuming data/grants.json. Use when creating or modifying UI components, KPI cards, deadline countdown badges, multi-facet filters, Kanban pipeline boards, or GitHub Pages deployment workflows.
---

# ASEAN Frontend Dashboard

Build, style, and maintain the static/Tailwind web dashboard in `web/` that visualizes opportunities from `data/grants.json`.

## 1. Scope Boundary

- **Owns:** All files inside `web/` (and GitHub Pages deployment workflow configuration when updating frontend publishing).
- **Strict Boundary:** Never modify Go backend files (`cmd/`, `internal/`, `config.yaml`, `data/`).
- **Data Contract:** Read `../data/grants.json` strictly according to [schema.json](../asean-backend-pipeline/references/schema.json) and [ui-contract.md](./references/ui-contract.md).

## 2. Core Views & Components

1. **4 KPI Cards:**
   - **Active Opportunities** (non-expired count)
   - **Closing Soon** (`<= 7 days`)
   - **Grants vs. RFPs Split** (`Grant / CFP` vs. `Tender / RFP`)
   - **Active Drafting** (`pipeline_stage === "4-Drafting"`)

2. **Multi-Facet Filter Bar:**
   - Live search across `title` and `source`
   - Type toggle (`All`, `Grant / CFP`, `Tender / RFP`)
   - Pipeline Stage filter (all 6 `pipeline_stage` values)
   - Sort by `close_date` with soonest closing first and `"TBD"` last

3. **Dual Layout Switcher:**
   - **Table View:** Scannable rows with semantic pill badges (`Closing Soon` in red/amber, `Grant / CFP` in emerald, `Tender / RFP` in indigo).
   - **Kanban Pipeline View:** 6-column board mapped 1:1 to `pipeline_stage` (`1-Triage`, `2-Go/No-Go`, `3-Partnering`, `4-Drafting`, `5-Submitted`, `6-Awarded/Expired`).

## 3. Manual Quick-Add Integration

Include a prominent `+ Add Manual Opportunity` CTA button/link pointing to the repository's GitHub Issue Form (`/issues/new?template=add_opportunity.yml`) so users can log login-walled opportunities (e.g., DFAT AusConnect).

## 4. Verification

After any frontend edit:
1. Run [verify_web.sh](./scripts/verify_web.sh) to verify DOM/data-binding tokens.
2. Run `../asean-pipeline-testing/scripts/run_checks.sh` to verify end-to-end harness health.
