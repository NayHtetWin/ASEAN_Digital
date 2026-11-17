package main

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDeadlineMath(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)

	dateCases := []struct {
		name string
		raw  string
		want string
	}{
		{"iso", "Closing Date: 2026-12-01", "2026-12-01"},
		{"slash", "Apply by 26/11/2026", "2026-11-26"},
		{"day_month_year", "Deadline: 5 Dec 2026 at 17:00 ICT", "2026-12-05"},
		{"month_day_year", "Submission due November 30, 2026", "2026-11-30"},
		{"deadline_cue_priority", "Published 2026-11-01. Deadline: 2026-12-05", "2026-12-05"},
		{"missing", "Rolling applications accepted", "TBD"},
	}
	for _, tc := range dateCases {
		t.Run("extract_"+tc.name, func(t *testing.T) {
			if got := extractCloseDate(tc.raw); got != tc.want {
				t.Fatalf("extractCloseDate(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}

	statusCases := []struct {
		name      string
		closeDate string
		want      string
	}{
		{"expired", "2026-11-16", "Expired"},
		{"closing_today", "2026-11-17", "Closing Soon (0d left)"},
		{"closing_5d", "2026-11-22", "Closing Soon (5d left)"},
		{"closing_7d", "2026-11-24", "Closing Soon (7d left)"},
		{"open_8d", "2026-11-25", "Open (8d left)"},
		{"tbd", "TBD", "Open (TBD)"},
		{"empty", "", "Open (TBD)"},
	}
	for _, tc := range statusCases {
		t.Run("status_"+tc.name, func(t *testing.T) {
			if got := computeDeadlineStatus(tc.closeDate, now); got != tc.want {
				t.Fatalf("computeDeadlineStatus(%q) = %q, want %q", tc.closeDate, got, tc.want)
			}
		})
	}
}

func TestFilterAndClassify(t *testing.T) {
	regions := []string{"asean", "southeast asia", "vietnam", "thailand", "indo-pacific"}
	themes := []string{"digital economy", "defa", "workforce", "ict", "msme", "tvet"}
	rfpMarkers := []string{"request for proposal", "rfp", "tender", "rapid assessment", "consultancy"}
	grantMarkers := []string{"grant", "call for proposals", "cfp", "fellowship"}

	relevanceCases := []struct {
		name        string
		title       string
		body        string
		forceRegion bool
		want        bool
	}{
		{"asean_digital", "ASEAN Digital Economy Grant", "Supporting MSMEs", false, true},
		{"vietnam_tvet", "Vietnam TVET Upskilling Call", "Workforce readiness", false, true},
		{"non_asean_rejected", "EU Digital Workforce Fund", "Europe only", false, false},
		{"non_theme_rejected", "ASEAN Clean Water Program", "Sanitation only", true, false},
		{"substring_ict_rejected", "Philippines Country Brief", "Conflict and district relief update", false, false},
		{"force_region_override", "DEFA ICT Rapid Assessment", "Regional study", true, true},
	}
	for _, tc := range relevanceCases {
		t.Run("relevant_"+tc.name, func(t *testing.T) {
			if got := isRelevant(tc.title, tc.body, tc.forceRegion, regions, themes); got != tc.want {
				t.Fatalf("isRelevant(%q, force=%v) = %v, want %v", tc.title, tc.forceRegion, got, tc.want)
			}
		})
	}

	classifyCases := []struct {
		name string
		text string
		want string
	}{
		{"rfp", "DFAT AusConnect Request for Proposal: Rapid Assessment of the ICT Eco-system", "RFP/Tender"},
		{"tender", "Open Tender for National TVET Portal", "RFP/Tender"},
		{"grant", "Call for Proposals: ASEAN Digital Transformation Grant", "Grant"},
		{"fellowship", "Indo-Pacific Digital Governance Fellowship", "Grant"},
		{"unmarked", "Philippines Country Brief November 2026", ""},
	}
	for _, tc := range classifyCases {
		t.Run("classify_"+tc.name, func(t *testing.T) {
			if got := classifyOpportunity(tc.text, rfpMarkers, grantMarkers); got != tc.want {
				t.Fatalf("classifyOpportunity(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}

func TestStorageParity(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)

	manuals := []ManualEntry{
		{
			Title:     "DFAT AusConnect Request for Proposal: Rapid Assessment of the ICT Eco-system",
			Type:      "RFP/Tender",
			Source:    "DFAT AusConnect",
			OpenDate:  "2026-11-01",
			CloseDate: "2026-11-26",
			URL:       "https://ausconnect.dfat.gov.au/",
		},
		{
			Title:     "ASEAN Digital Workforce Grant",
			Type:      "Grant",
			Source:    "ASEAN Foundation",
			OpenDate:  "2026-11-10",
			CloseDate: "2026-11-22",
			URL:       "https://www.aseanfoundation.org/grants/workforce",
		},
	}

	// Include a duplicate in scraped to verify deduplication.
	scraped := []Opportunity{
		{
			ID:        hashID("https://www.aseanfoundation.org/grants/workforce"),
			Type:      "Grant",
			Title:     "ASEAN Digital Workforce Grant",
			Source:    "ASEAN Foundation",
			OpenDate:  "2026-11-10",
			CloseDate: "2026-11-22",
			URL:       "https://www.aseanfoundation.org/grants/workforce",
		},
	}

	merged := mergeAndRefresh(nil, scraped, manuals, now)
	if len(merged) != 2 {
		t.Fatalf("expected 2 deduplicated records, got %d", len(merged))
	}
	// Sorted by open_date descending: 2026-11-10 before 2026-11-01.
	if merged[0].OpenDate != "2026-11-10" || merged[1].OpenDate != "2026-11-01" {
		t.Fatalf("unexpected sort order: %+v", merged)
	}

	if err := writeOutputs(dir, merged); err != nil {
		t.Fatalf("writeOutputs failed: %v", err)
	}

	jsonBytes, err := os.ReadFile(filepath.Join(dir, "grants.json"))
	if err != nil {
		t.Fatalf("read grants.json: %v", err)
	}
	var loadedJSON []Opportunity
	if err := json.Unmarshal(jsonBytes, &loadedJSON); err != nil {
		t.Fatalf("unmarshal grants.json: %v", err)
	}

	jsBytes, err := os.ReadFile(filepath.Join(dir, "grants.js"))
	if err != nil {
		t.Fatalf("read grants.js: %v", err)
	}
	if !strings.HasPrefix(string(jsBytes), "window.GRANTS_DATA = [") {
		t.Fatalf("grants.js missing window.GRANTS_DATA wrapper")
	}

	cf, err := os.Open(filepath.Join(dir, "grants.csv"))
	if err != nil {
		t.Fatalf("open grants.csv: %v", err)
	}
	defer cf.Close()
	rows, err := csv.NewReader(cf).ReadAll()
	if err != nil {
		t.Fatalf("read grants.csv: %v", err)
	}

	if len(loadedJSON) != len(rows)-1 {
		t.Fatalf("parity mismatch: json=%d csv=%d", len(loadedJSON), len(rows)-1)
	}
	if loadedJSON[0].ID != rows[1][0] || loadedJSON[0].DeadlineStatus != rows[1][7] {
		t.Fatalf("field mismatch between JSON and CSV: json=%+v csv=%v", loadedJSON[0], rows[1])
	}
}

func TestParsers(t *testing.T) {
	now := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)
	var cfg Config
	cfg.Keywords.Regions = []string{"asean", "philippines"}
	cfg.Keywords.Themes = []string{"digital economy", "ict"}
	cfg.OpportunityTypes.RFPMarkers = []string{"request for proposal", "rfp"}
	cfg.OpportunityTypes.GrantMarkers = []string{"grant", "call for proposals"}

	rssXML := `<?xml version="1.0" encoding="UTF-8" ?>
<rss version="2.0"><channel><title>Test Feed</title>
  <item>
    <title>ASEAN Digital Economy Grant</title>
    <link>https://example.org/grant-1</link>
    <description>Call for proposals. Deadline: 2026-12-01</description>
    <pubDate>Tue, 10 Nov 2026 00:00:00 GMT</pubDate>
  </item>
  <item>
    <title>Philippines Country Brief</title>
    <link>https://example.org/brief-1</link>
    <description>Conflict and district update without grant marker.</description>
  </item>
</channel></rss>`

	rssItems, err := parseRSS(strings.NewReader(rssXML), SourceConfig{Name: "TestRSS"}, &cfg, now)
	if err != nil {
		t.Fatalf("parseRSS failed: %v", err)
	}
	if len(rssItems) != 1 || rssItems[0].Type != "Grant" || rssItems[0].CloseDate != "2026-12-01" {
		t.Fatalf("unexpected parseRSS output: %+v", rssItems)
	}

	htmlDoc := `<html><body>
  <article class="card">
    <a href="/rfp-ict">Request for Proposal: ICT Assessment</a>
    <p>Apply by 26/11/2026</p>
  </article>
  <article class="card">
    <a href="/other">General News</a>
    <p>No relevant theme</p>
  </article>
</body></html>`

	htmlItems, err := parseHTML(strings.NewReader(htmlDoc), SourceConfig{
		Name:         "TestHTML",
		URL:          "https://portal.example.org/calls",
		ItemSelector: "article.card",
		LinkSelector: "a",
		ForceRegion:  "ASEAN",
	}, &cfg, now)
	if err != nil {
		t.Fatalf("parseHTML failed: %v", err)
	}
	if len(htmlItems) != 1 || htmlItems[0].Type != "RFP/Tender" || htmlItems[0].URL != "https://portal.example.org/rfp-ict" || htmlItems[0].CloseDate != "2026-11-26" {
		t.Fatalf("unexpected parseHTML output: %+v", htmlItems)
	}
}

func TestArchivedSourcesAndFilters(t *testing.T) {
	now := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)
	var cfg Config
	cfg.Keywords.Regions = []string{"asean", "vietnam", "indonesia"}
	cfg.Keywords.Themes = []string{"digital economy", "ict", "workforce", "ai"}
	cfg.Keywords.OffScopeTitle = []string{"laptops", "civil works"}
	cfg.OpportunityTypes.RFPMarkers = []string{"request for proposal", "rfp", "tender", "consultancy"}
	cfg.OpportunityTypes.GrantMarkers = []string{"grant", "call for applications", "fellowship"}

	// 1. HTML enhancements: title_selector, deadline_selector, context_parent_levels, skip_if_text, off_scope_title
	htmlDoc := `<html><body>
  <div class="wrapper">
    <div class="card">
      <h4><a href="/open-call">ASEAN Digital Economy Call for Applications</a></h4>
      <p>Deadline: 2026-12-10</p>
    </div>
  </div>
  <div class="wrapper">
    <div class="card">
      <h4><a href="/closed-call">ASEAN AI Fellowship</a></h4>
      <p>Registration Closed</p>
    </div>
  </div>
  <div class="wrapper">
    <div class="card">
      <h4><a href="/laptops-call">Tender for Supply of Laptops for ICT Center</a></h4>
      <p>Deadline: 2026-12-05</p>
    </div>
  </div>
</body></html>`

	items, err := parseHTML(strings.NewReader(htmlDoc), SourceConfig{
		Name:                "ASEAN Foundation",
		URL:                 "https://aseanfoundation.org/call-for-applications/",
		ItemSelector:        "h4",
		LinkSelector:        "a",
		ContextParentLevels: 2,
		SkipIfText:          "Registration Closed",
		ForceRegion:         "ASEAN",
	}, &cfg, now)
	if err != nil {
		t.Fatalf("parseHTML failed: %v", err)
	}
	if len(items) != 1 || items[0].Title != "ASEAN Digital Economy Call for Applications" || items[0].CloseDate != "2026-12-10" {
		t.Fatalf("unexpected enhanced parseHTML output: %+v", items)
	}

	// 2. Grants.gov mock API
	mockClient := func(body string) *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(body)),
				}, nil
			}),
		}
	}

	ggClient := mockClient(`{"data":{"oppHits":[{"id":"12345","title":"ASEAN Digital Workforce Grant","openDate":"11/01/2026","closeDate":"12/15/2026","docType":"grants"}]}}`)
	ggItems, err := fetchGrantsGov(ggClient, SourceConfig{
		Name:     "Grants.gov",
		URL:      "https://api.grants.gov/v1/api/search2",
		Type:     "grants_gov",
		Keywords: []string{"ASEAN"},
	}, &cfg, now)
	if err != nil || len(ggItems) != 1 || ggItems[0].CloseDate != "2026-12-15" || ggItems[0].Type != "Grant" {
		t.Fatalf("unexpected fetchGrantsGov output: err=%v items=%+v", err, ggItems)
	}

	// 3. EU Portal mock API
	euClient := mockClient(`{"results":[{"url":"https://ec.europa.eu/topic/1","metadata":{"title":["ASEAN Digital Economy Fellowship"],"type":["1"],"startDate":["2026-11-05T00:00:00.000+0000"],"deadlineDate":["2026-12-20T00:00:00.000+0000"]}}]}`)
	euItems, err := fetchEUPortal(euClient, SourceConfig{
		Name:     "EU Portal",
		URL:      "https://api.tech.ec.europa.eu/search-api/prod/rest/search",
		Type:     "eu_portal",
		Keywords: []string{"ASEAN"},
	}, &cfg, now)
	if err != nil || len(euItems) != 1 || euItems[0].CloseDate != "2026-12-20" || euItems[0].Type != "Grant" {
		t.Fatalf("unexpected fetchEUPortal output: err=%v items=%+v", err, euItems)
	}

	// 4. World Bank mock API (filters non-ASEAN country, keeps Vietnam)
	wbClient := mockClient(`{"procnotices":[{"id":"WB1","project_ctry_name":"Viet Nam","bid_description":"Consultancy for Digital Economy Strategy","noticedate":"2026-11-10T00:00:00Z","submission_deadline_date":"2026-12-12T00:00:00Z"},{"id":"WB2","project_ctry_name":"Brazil","bid_description":"Consultancy for Digital Economy Strategy","noticedate":"2026-11-10T00:00:00Z","submission_deadline_date":"2026-12-12T00:00:00Z"}]}`)
	wbItems, err := fetchWorldBank(wbClient, SourceConfig{
		Name:     "World Bank",
		URL:      "https://search.worldbank.org/api/v2/procnotices",
		Type:     "worldbank",
		Keywords: []string{"digital"},
	}, &cfg, now)
	if err != nil || len(wbItems) != 1 || wbItems[0].CloseDate != "2026-12-12" || wbItems[0].Type != "RFP/Tender" {
		t.Fatalf("unexpected fetchWorldBank output: err=%v items=%+v", err, wbItems)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
