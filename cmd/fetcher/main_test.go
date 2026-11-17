package main

import (
	"crypto/md5"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

type opportunity struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	Source         string `json:"source"`
	OpenDate       string `json:"open_date"`
	CloseDate      string `json:"close_date"`
	PipelineStage  string `json:"pipeline_stage"`
	DeadlineStatus string `json:"deadline_status"`
	URL            string `json:"url"`
}

// Reference helpers for standalone critical-path verification.
// Move or bind to main.go implementations once cmd/fetcher/main.go is created.
var dateRegex = regexp.MustCompile(`(\d{4}-\d{2}-\d{2}|\d{1,2}/\d{1,2}/\d{4}|\d{1,2}\s+[A-Za-z]{3,9}\s+\d{4})`)

func extractCloseDate(text string) string {
	m := dateRegex.FindString(text)
	if m == "" {
		return "TBD"
	}
	layouts := []string{"2006-01-02", "02/01/2006", "2/1/2006", "2 Jan 2006", "02 Jan 2006", "2 January 2006"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, m); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return "TBD"
}

func computeDeadlineStatus(closeDate string, now time.Time) string {
	if closeDate == "" || closeDate == "TBD" {
		return "Open (0d left)"
	}
	t, err := time.Parse("2006-01-02", closeDate)
	if err != nil {
		return "Open (0d left)"
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	days := int(t.Sub(today).Hours() / 24)
	switch {
	case days < 0:
		return "Expired"
	case days <= 7:
		return fmt.Sprintf("Closing Soon (%dd left)", days)
	default:
		return fmt.Sprintf("Open (%dd left)", days)
	}
}

func isRelevant(title, body string, forceRegion bool) bool {
	haystack := strings.ToLower(title + " " + body)
	hasTopic := false
	for _, kw := range []string{"digital", "ict", "defa", "workforce", "tech", "ai", "cyber"} {
		if strings.Contains(haystack, kw) {
			hasTopic = true
			break
		}
	}
	if !hasTopic {
		return false
	}
	if forceRegion {
		return true
	}
	for _, region := range []string{"asean", "southeast asia", "indonesia", "vietnam", "thailand", "malaysia", "philippines", "singapore", "cambodia", "laos", "myanmar", "brunei"} {
		if strings.Contains(haystack, region) {
			return true
		}
	}
	return false
}

func classifyOpportunity(title string) string {
	lower := strings.ToLower(title)
	for _, kw := range []string{"rfp", "request for proposal", "tender", "procurement", "rfq", "consultancy"} {
		if strings.Contains(lower, kw) {
			return "Tender / RFP"
		}
	}
	for _, kw := range []string{"grant", "cfp", "call for proposal", "fellowship", "fund"} {
		if strings.Contains(lower, kw) {
			return "Grant / CFP"
		}
	}
	return "Opportunity"
}

func TestCriticalDateAndDeadline(t *testing.T) {
	now := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)

	dateCases := []struct {
		name string
		raw  string
		want string
	}{
		{"slash_format", "Apply by 26/11/2026 for consideration", "2026-11-26"},
		{"short_month", "Deadline: 5 Dec 2026 at 17:00 ICT", "2026-12-05"},
		{"iso_format", "Closing date 2026-12-15", "2026-12-15"},
		{"missing_date", "Rolling applications accepted", "TBD"},
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
		{"closing_in_5d", "2026-11-22", "Closing Soon (5d left)"},
		{"closing_in_7d", "2026-11-24", "Closing Soon (7d left)"},
		{"open_10d", "2026-11-27", "Open (10d left)"},
	}
	for _, tc := range statusCases {
		t.Run("status_"+tc.name, func(t *testing.T) {
			if got := computeDeadlineStatus(tc.closeDate, now); got != tc.want {
				t.Fatalf("computeDeadlineStatus(%q) = %q, want %q", tc.closeDate, got, tc.want)
			}
		})
	}
}

func TestCriticalClassificationAndRelevance(t *testing.T) {
	relevanceCases := []struct {
		name        string
		title       string
		body        string
		forceRegion bool
		want        bool
	}{
		{"asean_digital_grant", "ASEAN Digital Economy Grant", "Supporting MSMEs", false, true},
		{"non_asean_rejected", "EU Digital Workforce Fund", "Europe only", false, false},
		{"non_topic_rejected", "ASEAN Clean Water Initiative", "Sanitation", true, false},
		{"force_region_override", "DEFA ICT Capacity Building", "Regional program", true, true},
	}
	for _, tc := range relevanceCases {
		t.Run("relevant_"+tc.name, func(t *testing.T) {
			if got := isRelevant(tc.title, tc.body, tc.forceRegion); got != tc.want {
				t.Fatalf("isRelevant(%q, force=%v) = %v, want %v", tc.title, tc.forceRegion, got, tc.want)
			}
		})
	}

	classifyCases := []struct {
		name  string
		title string
		want  string
	}{
		{"rfp_ict", "Request for Proposal: Rapid Assessment of ICT", "Tender / RFP"},
		{"tender_label", "Open Tender for Digital Skills Portal", "Tender / RFP"},
		{"grant_cfp", "Call for Proposals: ASEAN Workforce Innovation Grant", "Grant / CFP"},
	}
	for _, tc := range classifyCases {
		t.Run("classify_"+tc.name, func(t *testing.T) {
			if got := classifyOpportunity(tc.title); got != tc.want {
				t.Fatalf("classifyOpportunity(%q) = %q, want %q", tc.title, got, tc.want)
			}
		})
	}
}

func TestCriticalDualStorageParity(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "grants.json")
	csvPath := filepath.Join(dir, "grants.csv")

	sum := md5.Sum([]byte("https://example.org/asean-digital-grant"))
	id := fmt.Sprintf("%x", sum)[:10]

	records := []opportunity{
		{
			ID:             id,
			Type:           "Grant / CFP",
			Title:          "ASEAN Digital Workforce Grant",
			Source:         "ASEAN Foundation",
			OpenDate:       "2026-11-01",
			CloseDate:      "2026-11-24",
			PipelineStage:  "1-Triage",
			DeadlineStatus: "Closing Soon (7d left)",
			URL:            "https://example.org/asean-digital-grant",
		},
	}

	rawJSON, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	if err := os.WriteFile(jsonPath, rawJSON, 0o644); err != nil {
		t.Fatalf("write json: %v", err)
	}

	f, err := os.Create(csvPath)
	if err != nil {
		t.Fatalf("create csv: %v", err)
	}
	w := csv.NewWriter(f)
	_ = w.Write([]string{"id", "type", "title", "source", "open_date", "close_date", "pipeline_stage", "deadline_status", "url"})
	for _, r := range records {
		_ = w.Write([]string{r.ID, r.Type, r.Title, r.Source, r.OpenDate, r.CloseDate, r.PipelineStage, r.DeadlineStatus, r.URL})
	}
	w.Flush()
	_ = f.Close()

	var loadedJSON []opportunity
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read json: %v", err)
	}
	if err := json.Unmarshal(data, &loadedJSON); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}

	cf, err := os.Open(csvPath)
	if err != nil {
		t.Fatalf("open csv: %v", err)
	}
	defer cf.Close()
	rows, err := csv.NewReader(cf).ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	if len(loadedJSON) != len(rows)-1 {
		t.Fatalf("parity mismatch: json=%d, csv=%d", len(loadedJSON), len(rows)-1)
	}
	if len(loadedJSON[0].ID) != 10 {
		t.Fatalf("expected 10-char MD5 id, got %q", loadedJSON[0].ID)
	}
	if loadedJSON[0].ID != rows[1][0] || loadedJSON[0].CloseDate != rows[1][5] {
		t.Fatalf("field mismatch between JSON and CSV: json=%+v csv=%v", loadedJSON[0], rows[1])
	}
}
