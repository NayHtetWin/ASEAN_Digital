---
name: asean-pipeline-testing
description: Write and execute Go unit tests, HTML/RSS mock fixture tests, CSV/JSON schema parity checks, and frontend DOM/data-binding verification for the ASEAN Grant Tracker. Use after any backend or frontend code change, when debugging broken selectors or date regexes, or before committing.
---

# ASEAN Pipeline Testing & QA

Verify backend scrapers, deadline calculations, dual CSV/JSON storage parity, and frontend data-binding after any code change.

## 1. Offline Fixture Testing (No Live Network Flakiness)

- Test scrapers and date parsers in `internal/scraper/scraper_test.go` using local HTML/RSS fixtures in `testdata/` (including DFAT AusConnect, ASEAN Foundation, and FundsforNGOs snippets).
- Never hit live donor servers during unit tests. See [fixture-guidelines.md](./references/fixture-guidelines.md) for fixture layout conventions.

## 2. Table-Driven Go Tests

Write table-driven Go unit tests (`[]struct{ ... }`) covering:
- `extractCloseDate` (multi-format date regexes + `"TBD"` fallback)
- `computeDeadlineStatus` (`< 0d` = `Expired`, `<= 7d` = `Closing Soon (Xd left)`, `> 7d` = `Open (Xd left)`)
- `classifyOpportunity` (`"Grant / CFP"` vs. `"Tender / RFP"` vs. `"Opportunity"`)
- `isRelevant` (ASEAN digital economy, DEFA, and workforce relevance filtering + `force_region`)

## 3. Executable Verification Harness

Run [run_checks.sh](./scripts/run_checks.sh) after any backend or frontend change, when debugging broken selectors/regexes, or before committing:

```bash
.agents/skills/asean-pipeline-testing/scripts/run_checks.sh
```

The script executes:
1. `go vet ./...`
2. `go test -v -race ./...`
3. Schema & dual-storage parity validation between `data/grants.json` and `data/grants.csv` (matching row counts, valid `YYYY-MM-DD` or `TBD` dates, valid enums, zero duplicate `id` hashes)
4. Frontend data-binding verification (`../asean-frontend-dashboard/scripts/verify_web.sh`)
