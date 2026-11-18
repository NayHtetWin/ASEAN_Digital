# ASEAN Digital Economy & Workforce Opportunity Tracker — Business & Product Specification

## 1. Business Purpose & Problem Statement
- **Objective:** Centralize the discovery, qualification, and deadline tracking of funding and contracting opportunities related to the ASEAN digital economy and workforce planning.
- **Problem Solved:** Relevant opportunities are scattered across multilateral donor portals, government procurement boards (such as DFAT AusConnect), and foundation feeds, leading to missed submission deadlines, duplicated effort, and fragmented tracking.

## 2. Target Domain & Thematic Scope
Every tracked opportunity must satisfy both qualification pillars (excluding off-scope physical goods, civil works, and routine supply contracts):
- **Geographic Scope:** ASEAN member states (Brunei, Cambodia, Indonesia, Laos, Malaysia, Myanmar, Philippines, Singapore, Thailand, Vietnam), Timor-Leste, the Greater Mekong Subregion, or broader Indo-Pacific / Asia-Pacific initiatives impacting Southeast Asia.
- **Thematic Pillars:**
  1. **ASEAN Digital Economy:** Digital Economy Framework Agreement (DEFA), Digital Public Infrastructure (DPI), e-government and GovTech, AI and data governance, cybersecurity and digital trust, ICT ecosystem assessments, cross-border digital trade, and MSME / climate-tech digital inclusion.
  2. **Workforce Planning & Human Capital:** Technical and Vocational Education and Training (TVET), digital upskilling and reskilling, future of work, labor market information systems (LMIS), skills mapping, and inclusive digital literacy.

## 3. Opportunity Categories
The tracker classifies every qualified opportunity into one of two categories:
- **`Grant`:** Donor, foundation, fellowship, challenge fund, or research calls for proposals (CFPs) supporting self-designed programs, capacity building, or policy research.
- **`RFP/Tender`:** Public-sector or multilateral procurement contracts (RFPs, RFQs, EOIs, technical assistance, rapid assessments, feasibility studies) with defined Terms of Reference (ToR).

## 4. Source Cadence & Ingestion Rules
- **Daily Tracking (Fast-Moving Portals):** Procurement boards, multilateral APIs and tender portals, aggregator feeds, and targeted news feeds where submission windows are short.
- **Weekly Tracking (Programmatic Portals):** Foundation grant pages, ASEAN regional bodies, and research institute feeds (included in weekly full syncs).
- **Direct Manual Logging:** Streamlined logging of high-value opportunities from private networks or login-restricted portals (such as DFAT AusConnect) into the same single source of truth.

## 5. Opportunity Lifecycle & Deadline Intelligence
Every tracked opportunity maintains a canonical record and automated urgency status:
- **Core Record Fields:** Unique ID, Category (`Grant` vs. `RFP/Tender`), Title, Source, Open Date, Closing Deadline (`YYYY-MM-DD` or `TBD`), Pipeline Stage, Deadline Status, Direct URL, and optional `summary`.
- **Pipeline Stages (`pipeline_stage`):** `1-Triage`, `2-Go/No-Go`, `3-Partnering`, `4-Drafting`, `5-Submitted`, and `6-Awarded/Expired` (expired records automatically transition to `6-Awarded/Expired`).
- **Urgency & Deadline Rules:**
  - **`Open (Nd left)` / `Open (TBD)`:** More than 7 days remaining before the closing deadline, or `Open (TBD)` when no closing date is specified.
  - **`Closing Soon (Nd left)`:** 7 days or fewer remaining (0–7 days)—highlighted for immediate Go/No-Go action.
  - **`Expired`:** Closing deadline has passed—moved out of active focus while retained in historical records.

## 6. User Workflow, Decision Dashboard & Storage
- **4 Executive KPI Cards:** Instant visibility into Total Active Opportunities, Closing Soon (<= 7 Days), Grants / CFPs, and Tenders / RFPs.
- **Search, Multi-Facet Filtering & Sorting:**
  - Keyword search across opportunity title and source.
  - Filters for Category (`Grant` / `RFP/Tender`), Theme, Source, Status (`Active Only`, `Closing Soon`, `Open`, `Expired`, `All`), and Pipeline Stage, plus a Reset Filters action.
  - Multi-column sorting (soonest/latest deadline, newest/oldest open date, title A–Z, source A–Z) via sort menu or clickable table headers.
- **Views, Inspection & Theme:**
  - **Table View** with configurable pagination (10, 25, or 50 rows per page) and **Board View** organized across the 6 pipeline stages.
  - **Clickable Row Detail Modal** showing full record metadata, optional summary, detected themes, and direct portal link.
  - **Dark Mode** toggle with system-preference detection and saved preference.
- **Portability & Flat-File Storage:**
  - Portable triple flat-file storage (`data/grants.json`, `data/grants.csv`, `data/grants.js`) supporting both static/offline dashboard loading and external reporting, alongside one-click **CSV Export** in the dashboard.
