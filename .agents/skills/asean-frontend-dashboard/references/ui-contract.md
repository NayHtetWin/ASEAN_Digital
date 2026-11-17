# Frontend UI & Data-Binding Contract

The `web/` dashboard is a static/Tailwind web application consuming `../data/grants.json` (relative to `web/index.html` or served via GitHub Pages).

## Data Source & Schema

All rendered records must match [schema.json](../../asean-backend-pipeline/references/schema.json).

## KPI Cards (4 Required)

1. **Active Opportunities**: Count of records where `deadline_status !== "Expired"` (or `pipeline_stage !== "6-Awarded/Expired"`).
2. **Closing Soon (<= 7d)**: Count of records where `deadline_status` starts with `"Closing Soon"`.
3. **Grants vs. RFPs Split**: Count breakdown of `"Grant / CFP"` vs. `"Tender / RFP"`.
4. **Active Drafting**: Count of records where `pipeline_stage === "4-Drafting"`.

## Filter & Sort Behavior

- **Search**: Case-insensitive substring match across `title` and `source`.
- **Type Toggle**: `All` | `Grant / CFP` | `Tender / RFP`.
- **Pipeline Stage Filter**: `All` or any of the 6 `pipeline_stage` enum values.
- **Default Sort**: Ascending by `close_date` (soonest closing first); `"TBD"` always sorts last.

## Semantic Pill Badges

- `Closing Soon (...)`: Red/amber badge (`bg-red-100 text-red-800` or `bg-amber-100 text-amber-800`)
- `Grant / CFP`: Emerald badge (`bg-emerald-100 text-emerald-800`)
- `Tender / RFP`: Indigo badge (`bg-indigo-100 text-indigo-800`)

## Kanban Pipeline Columns (6 Required)

1. `1-Triage`
2. `2-Go/No-Go`
3. `3-Partnering`
4. `4-Drafting`
5. `5-Submitted`
6. `6-Awarded/Expired`

## Manual Quick-Add CTA

- Button/link labeled `+ Add Manual Opportunity`
- Target: `/issues/new?template=add_opportunity.yml` (relative to the repository's GitHub URL)
