# Offline Fixture & Table-Driven Testing Guidelines

Unit tests must **never** make live HTTP requests to donor portals. All scraper, selector, classification, and date-parsing tests run offline against deterministic fixtures in `testdata/`.

## Recommended `testdata/` Layout

```text
internal/scraper/testdata/
├── dfat_ausconnect.html
├── asean_foundation.html
└── fundsforngos.rss
```

- **DFAT AusConnect (`dfat_ausconnect.html`)**: Representative DOM snippet matching `item_selector` and `link_selector`, including varied close-date formats (`DD MMM YYYY`, `YYYY-MM-DD`, or `TBD`).
- **ASEAN Foundation (`asean_foundation.html`)**: Static HTML grant cards with CFP titles, open/close dates, and relative links.
- **FundsforNGOs (`fundsforngos.rss`)**: RSS/Atom XML feed with `<item>`/`<entry>` elements, `<pubDate>`, and digital/workforce vs. non-relevant entries to test `isRelevant`.

## Required Table-Driven Test Functions

In `internal/scraper/scraper_test.go` (or corresponding package tests), include table-driven tests (`[]struct{ name string; ... }`) for:
1. `TestExtractCloseDate` (`extractCloseDate`): ISO dates, day-month-year formats, US month-day-year formats, `"TBD"` / missing dates.
2. `TestComputeDeadlineStatus` (`computeDeadlineStatus`): `< 0` days (`Expired`), `0..7` days (`Closing Soon (Xd left)`), `> 7` days (`Open (Xd left)`), and `"TBD"`.
3. `TestClassifyOpportunity` (`classifyOpportunity`): `"Grant / CFP"` vs. `"Tender / RFP"` vs. `"Opportunity"` keyword classification.
4. `TestIsRelevant` (`isRelevant`): ASEAN digital economy, DEFA, and workforce keywords vs. out-of-scope regions/topics, plus `force_region` overrides.
