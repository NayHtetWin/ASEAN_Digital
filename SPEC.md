# ASEAN Digital Economy & Workforce Opportunity Tracker — Business & Product Specification

## 1. Business Purpose & Problem Statement
- **Objective:** Centralize the discovery, qualification, and deadline tracking of funding and contracting opportunities related to the ASEAN digital economy and workforce planning.
- **Problem Solved:** Relevant opportunities are scattered across dozens of multilateral donor portals, government procurement boards (such as DFAT AusConnect), and foundation feeds, leading to missed submission deadlines, duplicated effort, and fragmented tracking.

## 2. Target Domain & Thematic Scope
Every tracked opportunity must satisfy both qualification pillars below:
- **Geographic Scope:** ASEAN member states (Brunei, Cambodia, Indonesia, Laos, Malaysia, Myanmar, Philippines, Singapore, Thailand, Vietnam), Timor-Leste, the Greater Mekong Subregion, or broader Indo-Pacific regional initiatives impacting Southeast Asia.
- **Thematic Pillars:**
  1. **ASEAN Digital Economy:** Digital Economy Framework Agreement (DEFA), Digital Public Infrastructure (DPI), e-government and GovTech, ICT ecosystem assessments, cross-border data and trade, AI governance, and MSME digital transformation.
  2. **Workforce Planning & Human Capital:** Technical and Vocational Education and Training (TVET), digital upskilling and reskilling, future of work, labor market information systems, and inclusive digital skills.

## 3. Opportunity Categories We Track
The pipeline distinguishes two primary funding mechanisms:
- **Grants & Calls for Proposals (CFPs):** Donor, foundation, or research funding supporting self-designed programs, capacity building, or policy research.
- **Tenders & Consulting RFPs (RFPs / RFQs / EOIs):** Public-sector or multilateral procurement contracts (such as rapid ICT assessments, feasibility studies, and technical assistance) with defined Terms of Reference (ToR).

## 4. Source Cadence & Ingestion Rules
- **Daily Tracking (Fast-Moving Portals):** Procurement boards, multilateral tender feeds, and aggregator feeds where submission windows are short.
- **Weekly Tracking (Programmatic Portals):** Foundation grant pages, research institute calls for papers, and bilateral fellowship programs.
- **Direct Manual Logging:** A streamlined way to manually log high-value opportunities discovered via private networks or login-restricted portals (such as DFAT AusConnect) so every opportunity lives in a single source of truth.

## 5. Opportunity Lifecycle & Deadline Intelligence
Every tracked opportunity maintains a standardized business record and automated urgency status:
- **Core Business Record:** Opportunity Title, Category (`Grant` vs. `RFP/Tender`), Issuing Organization/Source, Open Date, Closing Deadline (`YYYY-MM-DD` or `TBD`), Urgency Status, Pipeline Stage, and Direct Link.
- **Urgency & Deadline Rules:**
  - **Open:** More than 7 days remaining before the closing deadline (or `TBD`).
  - **Closing Soon (High Priority):** 7 days or fewer remaining—highlighted immediately for Go/No-Go action.
  - **Expired:** Closing deadline has passed—automatically moved out of active focus while retained in historical records.

## 6. User Workflow & Decision Dashboard
- **At-a-Glance Executive Summary:** Instant visibility into Total Active Opportunities, Urgent Deadlines (`Closing Soon`), Active Drafting count, and the split between Grants and Consulting RFPs.
- **Triage & Filtering:** Search by keyword or donor, filter by `Grant` vs. `RFP/Tender` and pipeline stage, and sort by soonest closing deadline to prioritize proposal writing.
- **Portability:** All tracked opportunities remain accessible through a clean visual decision dashboard (table and pipeline board views) as well as a downloadable spreadsheet (CSV) and structured data feed (JSON) for external reporting.
