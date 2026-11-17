package main

import (
	"bytes"
	"crypto/md5"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
	"gopkg.in/yaml.v3"
)

const (
	httpTimeout = 15 * time.Second
	userAgent   = "ASEAN-Digital-Tracker/1.0 (+https://github.com)"
	dataDir     = "data"
)

// Config represents the root config.yaml structure.
type Config struct {
	Keywords struct {
		Regions       []string `yaml:"regions"`
		Themes        []string `yaml:"themes"`
		OffScopeTitle []string `yaml:"off_scope_title"`
	} `yaml:"keywords"`
	OpportunityTypes struct {
		RFPMarkers   []string `yaml:"rfp_markers"`
		GrantMarkers []string `yaml:"grant_markers"`
	} `yaml:"opportunity_types"`
	Sources       []SourceConfig `yaml:"sources"`
	ManualEntries []ManualEntry  `yaml:"manual_entries"`
}

// SourceConfig defines a single RSS, HTML, or API source to scrape.
type SourceConfig struct {
	Name                string   `yaml:"name"`
	URL                 string   `yaml:"url"`
	Frequency           string   `yaml:"frequency"`
	Type                string   `yaml:"type"`
	ItemSelector        string   `yaml:"item_selector"`
	LinkSelector        string   `yaml:"link_selector"`
	TitleSelector       string   `yaml:"title_selector"`
	DeadlineSelector    string   `yaml:"deadline_selector"`
	ContextParentLevels int      `yaml:"context_parent_levels"`
	SkipIfText          string   `yaml:"skip_if_text"`
	ForceRegion         string   `yaml:"force_region"`
	Keywords            []string `yaml:"keywords"`
	Statuses            string   `yaml:"statuses"`
}

// ManualEntry defines a manually logged opportunity in config.yaml.
type ManualEntry struct {
	Title     string `yaml:"title"`
	Type      string `yaml:"type"`
	Source    string `yaml:"source"`
	OpenDate  string `yaml:"open_date"`
	CloseDate string `yaml:"close_date"`
	URL       string `yaml:"url"`
}

// Opportunity is the canonical record persisted to grants.json, grants.csv, and grants.js.
type Opportunity struct {
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

var dateRegex = regexp.MustCompile(`(\d{4}-\d{2}-\d{2}|\d{1,2}/\d{1,2}/\d{4}|[A-Za-z]{3,9}\s+\d{1,2},\s+\d{4}|\d{1,2}\s+[A-Za-z]{3,9}\s+\d{4})`)

var dateLayouts = []string{
	"2006-01-02",
	"02/01/2006",
	"2/1/2006",
	"02 Jan 2006",
	"2 Jan 2006",
	"02 January 2006",
	"2 January 2006",
	"January 2, 2006",
	"Jan 2, 2006",
}

func main() {
	schedule := flag.String("schedule", "all", "Schedule filter: daily, weekly, or all")
	configPath := flag.String("config", "config.yaml", "Path to config.yaml")
	flag.Parse()

	sched := strings.ToLower(strings.TrimSpace(*schedule))
	if sched != "daily" && sched != "weekly" && sched != "all" {
		log.Fatalf("invalid -schedule %q: must be daily, weekly, or all", *schedule)
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load %s: %v", *configPath, err)
	}

	now := time.Now().UTC()
	existing, err := loadExistingJSON(filepath.Join(dataDir, "grants.json"))
	if err != nil {
		log.Printf("warning: could not read existing grants.json: %v", err)
	}

	scraped := fetchAllSources(cfg, sched, now)
	merged := mergeAndRefresh(existing, scraped, cfg.ManualEntries, now)

	if err := writeOutputs(dataDir, merged); err != nil {
		log.Fatalf("failed to write outputs: %v", err)
	}
	log.Printf("sync complete (%s): %d total opportunities saved to %s/", sched, len(merged), dataDir)
}

func loadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func loadExistingJSON(path string) ([]Opportunity, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, nil
	}
	var records []Opportunity
	if err := json.Unmarshal(raw, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func fetchAllSources(cfg *Config, schedule string, now time.Time) []Opportunity {
	client := &http.Client{Timeout: httpTimeout}
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []Opportunity
	)

	for _, src := range cfg.Sources {
		if schedule != "all" && !strings.EqualFold(src.Frequency, schedule) {
			continue
		}
		wg.Add(1)
		go func(s SourceConfig) {
			defer wg.Done()
			items, err := scrapeSource(client, s, cfg, now)
			if err != nil {
				log.Printf("warning: source %q (%s) failed: %v", s.Name, s.URL, err)
				return
			}
			log.Printf("source %q: matched %d opportunity(ies)", s.Name, len(items))
			mu.Lock()
			results = append(results, items...)
			mu.Unlock()
		}(src)
	}
	wg.Wait()
	return results
}

func scrapeSource(client *http.Client, src SourceConfig, cfg *Config, now time.Time) ([]Opportunity, error) {
	switch strings.ToLower(src.Type) {
	case "grants_gov":
		return fetchGrantsGov(client, src, cfg, now)
	case "eu_portal":
		return fetchEUPortal(client, src, cfg, now)
	case "worldbank":
		return fetchWorldBank(client, src, cfg, now)
	}

	req, err := http.NewRequest(http.MethodGet, src.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}

	switch strings.ToLower(src.Type) {
	case "rss":
		return parseRSS(resp.Body, src, cfg, now)
	case "html":
		return parseHTML(resp.Body, src, cfg, now)
	default:
		return nil, fmt.Errorf("unsupported source type %q", src.Type)
	}
}

func parseRSS(r io.Reader, src SourceConfig, cfg *Config, now time.Time) ([]Opportunity, error) {
	feed, err := gofeed.NewParser().Parse(r)
	if err != nil {
		return nil, err
	}
	var out []Opportunity
	for _, item := range feed.Items {
		title := strings.TrimSpace(item.Title)
		link := strings.TrimSpace(item.Link)
		if title == "" || link == "" {
			continue
		}
		if isOffScopeTitle(title, cfg.Keywords.OffScopeTitle) {
			continue
		}
		body := item.Description + " " + item.Content
		if !isRelevant(title, body, src.ForceRegion != "", cfg.Keywords.Regions, cfg.Keywords.Themes) {
			continue
		}
		openDate := now.Format("2006-01-02")
		if item.PublishedParsed != nil {
			openDate = item.PublishedParsed.UTC().Format("2006-01-02")
		}
		closeDate := extractCloseDate(title + " " + body)
		oppType := classifyOpportunity(title+" "+body, cfg.OpportunityTypes.RFPMarkers, cfg.OpportunityTypes.GrantMarkers)
		if oppType == "" {
			continue
		}
		out = append(out, Opportunity{
			ID:             hashID(link),
			Type:           oppType,
			Title:          title,
			Source:         src.Name,
			OpenDate:       openDate,
			CloseDate:      closeDate,
			PipelineStage:  "1-Triage",
			DeadlineStatus: computeDeadlineStatus(closeDate, now),
			URL:            link,
		})
	}
	return out, nil
}

func parseHTML(r io.Reader, src SourceConfig, cfg *Config, now time.Time) ([]Opportunity, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}
	itemSel := src.ItemSelector
	if itemSel == "" {
		itemSel = "article, li"
	}
	linkSel := src.LinkSelector
	if linkSel == "" {
		linkSel = "a"
	}

	var out []Opportunity
	seen := make(map[string]struct{})
	doc.Find(itemSel).Each(func(_ int, sel *goquery.Selection) {
		var a *goquery.Selection
		if sel.Is("a") && (src.LinkSelector == "" || src.LinkSelector == "a") {
			a = sel
		} else {
			a = sel.Find(linkSel).First()
			if a.Length() == 0 && sel.Is("a") {
				a = sel
			}
		}
		href, exists := a.Attr("href")
		if !exists || strings.TrimSpace(href) == "" {
			return
		}
		resolvedURL := resolveURL(src.URL, strings.TrimSpace(href))
		if resolvedURL == "" {
			return
		}
		if _, dup := seen[resolvedURL]; dup {
			return
		}
		seen[resolvedURL] = struct{}{}

		title := ""
		if src.TitleSelector != "" {
			title = strings.Join(strings.Fields(sel.Find(src.TitleSelector).First().Text()), " ")
		}
		if title == "" {
			title = strings.Join(strings.Fields(a.Text()), " ")
		}

		ctx := sel
		for i := 0; i < src.ContextParentLevels; i++ {
			if p := ctx.Parent(); p.Length() > 0 {
				ctx = p
			}
		}
		ctxText := strings.Join(strings.Fields(ctx.Text()), " ")
		if title == "" {
			title = ctxText
		}
		if src.SkipIfText != "" && strings.Contains(strings.ToLower(ctxText), strings.ToLower(src.SkipIfText)) {
			return
		}
		if title == "" || isOffScopeTitle(title, cfg.Keywords.OffScopeTitle) {
			return
		}
		if !isRelevant(title, ctxText, src.ForceRegion != "", cfg.Keywords.Regions, cfg.Keywords.Themes) {
			return
		}

		closeDate := "TBD"
		if src.DeadlineSelector != "" {
			if dtText := strings.Join(strings.Fields(ctx.Find(src.DeadlineSelector).First().Text()), " "); dtText != "" {
				closeDate = extractCloseDate(dtText)
			}
		}
		if closeDate == "TBD" {
			closeDate = extractCloseDate(ctxText)
		}

		oppType := classifyOpportunity(title+" "+ctxText, cfg.OpportunityTypes.RFPMarkers, cfg.OpportunityTypes.GrantMarkers)
		if oppType == "" {
			return
		}
		out = append(out, Opportunity{
			ID:             hashID(resolvedURL),
			Type:           oppType,
			Title:          title,
			Source:         src.Name,
			OpenDate:       now.Format("2006-01-02"),
			CloseDate:      closeDate,
			PipelineStage:  "1-Triage",
			DeadlineStatus: computeDeadlineStatus(closeDate, now),
			URL:            resolvedURL,
		})
	})
	return out, nil
}

func resolveURL(baseRaw, href string) string {
	base, err := url.Parse(baseRaw)
	if err != nil {
		return ""
	}
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(ref)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}
	return resolved.String()
}

// containsKeyword checks for a whole-word/phrase match (case-insensitive) to avoid
// false positives like "ict" matching inside "conflict" or "district".
func containsKeyword(haystackLower, kw string) bool {
	k := strings.ToLower(strings.TrimSpace(kw))
	if k == "" {
		return false
	}
	re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(k) + `\b`)
	return re.MatchString(haystackLower)
}

// isRelevant checks if the combined text matches (Region OR forceRegion) AND Theme.
func isRelevant(title, body string, forceRegion bool, regions, themes []string) bool {
	haystack := strings.ToLower(title + " " + body)
	hasTheme := false
	for _, t := range themes {
		if containsKeyword(haystack, t) {
			hasTheme = true
			break
		}
	}
	if !hasTheme {
		return false
	}
	if forceRegion {
		return true
	}
	for _, r := range regions {
		if containsKeyword(haystack, r) {
			return true
		}
	}
	return false
}

// classifyOpportunity categorizes text into "RFP/Tender", "Grant", or "" if no marker matches.
func classifyOpportunity(text string, rfpMarkers, grantMarkers []string) string {
	lower := strings.ToLower(text)
	for _, m := range rfpMarkers {
		if containsKeyword(lower, m) {
			return "RFP/Tender"
		}
	}
	for _, m := range grantMarkers {
		if containsKeyword(lower, m) {
			return "Grant"
		}
	}
	return ""
}

var deadlineCueRegex = regexp.MustCompile(`(?i)(?:deadline|closing\s*date|close\s*date|closes|apply\s*by|due(?:\s*date)?)\s*[:\-]?\s*` + dateRegex.String())

// extractCloseDate parses a recognized deadline date into YYYY-MM-DD, or "TBD".
// It prefers dates near explicit deadline cues before falling back to the first date in text.
func extractCloseDate(text string) string {
	candidate := ""
	if cueMatches := deadlineCueRegex.FindStringSubmatch(text); len(cueMatches) > 1 {
		candidate = cueMatches[1]
	} else {
		candidate = dateRegex.FindString(text)
	}
	if candidate == "" {
		return "TBD"
	}
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, candidate); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return "TBD"
}

// computeDeadlineStatus computes status relative to today's UTC date.
func computeDeadlineStatus(closeDate string, now time.Time) string {
	if closeDate == "" || strings.EqualFold(closeDate, "TBD") {
		return "Open (TBD)"
	}
	t, err := time.Parse("2006-01-02", closeDate)
	if err != nil {
		return "Open (TBD)"
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

func hashID(rawURL string) string {
	sum := md5.Sum([]byte(strings.TrimSpace(rawURL)))
	return fmt.Sprintf("%x", sum)[:10]
}

func mergeAndRefresh(existing, scraped []Opportunity, manuals []ManualEntry, now time.Time) []Opportunity {
	byID := make(map[string]Opportunity)

	for _, rec := range existing {
		if rec.ID == "" && rec.URL != "" {
			rec.ID = hashID(rec.URL)
		}
		if rec.ID != "" {
			byID[rec.ID] = rec
		}
	}

	for _, rec := range scraped {
		if prev, ok := byID[rec.ID]; ok {
			if prev.PipelineStage != "" {
				rec.PipelineStage = prev.PipelineStage
			}
			if prev.OpenDate != "" {
				rec.OpenDate = prev.OpenDate
			}
		}
		byID[rec.ID] = rec
	}

	for _, m := range manuals {
		if strings.TrimSpace(m.URL) == "" || strings.TrimSpace(m.Title) == "" {
			continue
		}
		id := hashID(m.URL)
		stage := "1-Triage"
		if prev, ok := byID[id]; ok && prev.PipelineStage != "" {
			stage = prev.PipelineStage
		}
		openDate := strings.TrimSpace(m.OpenDate)
		if openDate == "" {
			openDate = now.Format("2006-01-02")
		}
		closeDate := strings.TrimSpace(m.CloseDate)
		if closeDate == "" {
			closeDate = "TBD"
		}
		oppType := strings.TrimSpace(m.Type)
		if oppType != "RFP/Tender" && oppType != "Grant" {
			oppType = "Grant"
		}
		byID[id] = Opportunity{
			ID:            id,
			Type:          oppType,
			Title:         strings.TrimSpace(m.Title),
			Source:        strings.TrimSpace(m.Source),
			OpenDate:      openDate,
			CloseDate:     closeDate,
			PipelineStage: stage,
			URL:           strings.TrimSpace(m.URL),
		}
	}

	out := make([]Opportunity, 0, len(byID))
	for _, rec := range byID {
		rec.DeadlineStatus = computeDeadlineStatus(rec.CloseDate, now)
		if rec.DeadlineStatus == "Expired" {
			rec.PipelineStage = "6-Awarded/Expired"
		} else if rec.PipelineStage == "" {
			rec.PipelineStage = "1-Triage"
		}
		out = append(out, rec)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].OpenDate != out[j].OpenDate {
			return out[i].OpenDate > out[j].OpenDate
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func writeOutputs(dir string, records []Opportunity) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	rawJSON, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	rawJSON = append(rawJSON, '\n')

	if err := os.WriteFile(filepath.Join(dir, "grants.json"), rawJSON, 0o644); err != nil {
		return err
	}

	jsPayload := append([]byte("window.GRANTS_DATA = "), rawJSON[:len(rawJSON)-1]...)
	jsPayload = append(jsPayload, []byte(";\n")...)
	if err := os.WriteFile(filepath.Join(dir, "grants.js"), jsPayload, 0o644); err != nil {
		return err
	}

	csvPath := filepath.Join(dir, "grants.csv")
	f, err := os.Create(csvPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write([]string{"id", "type", "title", "source", "open_date", "close_date", "pipeline_stage", "deadline_status", "url"}); err != nil {
		return err
	}
	for _, r := range records {
		if err := w.Write([]string{r.ID, r.Type, r.Title, r.Source, r.OpenDate, r.CloseDate, r.PipelineStage, r.DeadlineStatus, r.URL}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func isOffScopeTitle(title string, offScopeTerms []string) bool {
	lower := strings.ToLower(title)
	for _, term := range offScopeTerms {
		if containsKeyword(lower, term) {
			return true
		}
	}
	return false
}

func parseAPIDate(raw, fallback string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return fallback
	}
	if len(s) >= 10 {
		if _, err := time.Parse("2006-01-02", s[:10]); err == nil {
			return s[:10]
		}
	}
	for _, layout := range []string{"01/02/2006", "1/2/2006", "2006-01-02", "Jan 2, 2006", "January 2, 2006", "02 Jan 2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return fallback
}

func fetchGrantsGov(client *http.Client, src SourceConfig, cfg *Config, now time.Time) ([]Opportunity, error) {
	endpoint := src.URL
	if endpoint == "" {
		endpoint = "https://api.grants.gov/v1/api/search2"
	}
	keywords := src.Keywords
	if len(keywords) == 0 {
		keywords = []string{"ASEAN"}
	}
	statuses := src.Statuses
	if statuses == "" {
		statuses = "forecasted|posted"
	}

	var out []Opportunity
	seen := make(map[string]struct{})
	for _, kw := range keywords {
		payload, err := json.Marshal(map[string]any{
			"keyword":     kw,
			"oppStatuses": statuses,
			"rows":        100,
		})
		if err != nil {
			continue
		}
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", userAgent)

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("warning: grants_gov (%s) failed: %v", kw, err)
			continue
		}
		var parsed struct {
			Data struct {
				OppHits []struct {
					ID         string   `json:"id"`
					Title      string   `json:"title"`
					Agency     string   `json:"agency"`
					AgencyName string   `json:"agencyName"`
					OpenDate   string   `json:"openDate"`
					CloseDate  string   `json:"closeDate"`
					DocType    string   `json:"docType"`
					CFDAList   []string `json:"cfdaList"`
				} `json:"oppHits"`
			} `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&parsed)
		resp.Body.Close()
		if err != nil {
			continue
		}

		for _, h := range parsed.Data.OppHits {
			title := strings.TrimSpace(h.Title)
			if title == "" || h.ID == "" {
				continue
			}
			link := fmt.Sprintf("https://www.grants.gov/search-results-detail/%s", h.ID)
			if _, dup := seen[link]; dup {
				continue
			}
			seen[link] = struct{}{}

			if isOffScopeTitle(title, cfg.Keywords.OffScopeTitle) {
				continue
			}
			body := fmt.Sprintf("%s %s %s", kw, h.DocType, strings.Join(h.CFDAList, " "))
			if !isRelevant(title, body, src.ForceRegion != "", cfg.Keywords.Regions, cfg.Keywords.Themes) {
				continue
			}
			oppType := classifyOpportunity(title+" "+body, cfg.OpportunityTypes.RFPMarkers, cfg.OpportunityTypes.GrantMarkers)
			if oppType == "" {
				oppType = "Grant"
			}
			closeDate := parseAPIDate(h.CloseDate, "TBD")
			openDate := parseAPIDate(h.OpenDate, now.Format("2006-01-02"))
			out = append(out, Opportunity{
				ID:             hashID(link),
				Type:           oppType,
				Title:          title,
				Source:         src.Name,
				OpenDate:       openDate,
				CloseDate:      closeDate,
				PipelineStage:  "1-Triage",
				DeadlineStatus: computeDeadlineStatus(closeDate, now),
				URL:            link,
			})
		}
	}
	return out, nil
}

func fetchEUPortal(client *http.Client, src SourceConfig, cfg *Config, now time.Time) ([]Opportunity, error) {
	endpoint := src.URL
	if endpoint == "" {
		endpoint = "https://api.tech.ec.europa.eu/search-api/prod/rest/search"
	}
	keywords := src.Keywords
	if len(keywords) == 0 {
		keywords = []string{"ASEAN"}
	}

	queryJSON, _ := json.Marshal(map[string]any{
		"bool": map[string]any{
			"must": []any{
				map[string]any{"terms": map[string]any{"type": []string{"0", "1", "2", "8"}}},
				map[string]any{"terms": map[string]any{"status": []string{"31094501", "31094502"}}},
			},
		},
	})
	langJSON := `["en"]`

	var out []Opportunity
	seen := make(map[string]struct{})
	for _, kw := range keywords {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		_ = mw.WriteField("query", string(queryJSON))
		_ = mw.WriteField("languages", langJSON)
		_ = mw.Close()

		u, err := url.Parse(endpoint)
		if err != nil {
			continue
		}
		q := u.Query()
		q.Set("apiKey", "SEDIA")
		q.Set("text", kw)
		q.Set("pageSize", "100")
		q.Set("pageNumber", "1")
		u.RawQuery = q.Encode()

		req, err := http.NewRequest(http.MethodPost, u.String(), &buf)
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.Header.Set("User-Agent", userAgent)

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("warning: eu_portal (%s) failed: %v", kw, err)
			continue
		}
		var parsed struct {
			Results []struct {
				URL      string              `json:"url"`
				Summary  string              `json:"summary"`
				Content  string              `json:"content"`
				Metadata map[string][]string `json:"metadata"`
			} `json:"results"`
		}
		err = json.NewDecoder(resp.Body).Decode(&parsed)
		resp.Body.Close()
		if err != nil {
			continue
		}

		firstMeta := func(md map[string][]string, key string) string {
			if vals := md[key]; len(vals) > 0 {
				return strings.TrimSpace(vals[0])
			}
			return ""
		}

		for _, r := range parsed.Results {
			ident := firstMeta(r.Metadata, "identifier")
			if ident == "" {
				ident = firstMeta(r.Metadata, "callIdentifier")
			}
			title := firstMeta(r.Metadata, "title")
			if title == "" {
				title = strings.TrimSpace(r.Summary)
			}
			if title == "" {
				title = strings.TrimSpace(r.Content)
			}
			link := strings.TrimSpace(r.URL)
			if link == "" && ident != "" {
				link = "https://ec.europa.eu/info/funding-tenders/opportunities/portal/screen/opportunities/topic-details/" + strings.ToLower(ident)
			}
			if title == "" || link == "" {
				continue
			}
			if _, dup := seen[link]; dup {
				continue
			}
			seen[link] = struct{}{}

			if isOffScopeTitle(title, cfg.Keywords.OffScopeTitle) {
				continue
			}
			body := firstMeta(r.Metadata, "descriptionByte") + " " + firstMeta(r.Metadata, "callTitle") + " " + r.Summary + " " + kw
			if !isRelevant(title, body, src.ForceRegion != "", cfg.Keywords.Regions, cfg.Keywords.Themes) {
				continue
			}
			oppType := "Grant"
			if firstMeta(r.Metadata, "type") == "0" {
				oppType = "RFP/Tender"
			} else if classified := classifyOpportunity(title+" "+body, cfg.OpportunityTypes.RFPMarkers, cfg.OpportunityTypes.GrantMarkers); classified != "" {
				oppType = classified
			}
			closeDate := parseAPIDate(firstMeta(r.Metadata, "deadlineDate"), "TBD")
			openDate := parseAPIDate(firstMeta(r.Metadata, "startDate"), now.Format("2006-01-02"))
			out = append(out, Opportunity{
				ID:             hashID(link),
				Type:           oppType,
				Title:          title,
				Source:         src.Name,
				OpenDate:       openDate,
				CloseDate:      closeDate,
				PipelineStage:  "1-Triage",
				DeadlineStatus: computeDeadlineStatus(closeDate, now),
				URL:            link,
			})
		}
	}
	return out, nil
}

var worldBankASEANCountries = map[string]bool{
	"indonesia":                        true,
	"philippines":                      true,
	"viet nam":                         true,
	"vietnam":                          true,
	"cambodia":                         true,
	"lao people's democratic republic": true,
	"lao pdr":                          true,
	"myanmar":                          true,
	"thailand":                         true,
	"malaysia":                         true,
	"timor-leste":                      true,
	"singapore":                        true,
	"brunei darussalam":                true,
	"east asia and pacific":            true,
}

func fetchWorldBank(client *http.Client, src SourceConfig, cfg *Config, now time.Time) ([]Opportunity, error) {
	endpoint := src.URL
	if endpoint == "" {
		endpoint = "https://search.worldbank.org/api/v2/procnotices"
	}
	keywords := src.Keywords
	if len(keywords) == 0 {
		keywords = []string{"digital"}
	}

	var out []Opportunity
	seen := make(map[string]struct{})
	for _, kw := range keywords {
		u, err := url.Parse(endpoint)
		if err != nil {
			continue
		}
		q := u.Query()
		q.Set("format", "json")
		q.Set("rows", "200")
		q.Set("qterm", kw)
		q.Set("srt", "noticedate")
		q.Set("order", "desc")
		u.RawQuery = q.Encode()

		req, err := http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("warning: worldbank (%s) failed: %v", kw, err)
			continue
		}
		var parsed struct {
			ProcNotices []struct {
				ID                     string `json:"id"`
				ProjectCountryName     string `json:"project_ctry_name"`
				BidDescription         string `json:"bid_description"`
				ProjectName            string `json:"project_name"`
				NoticeType             string `json:"notice_type"`
				ProcurementGroup       string `json:"procurement_group"`
				NoticeText             string `json:"notice_text"`
				NoticeDate             string `json:"noticedate"`
				SubmissionDeadlineDate string `json:"submission_deadline_date"`
			} `json:"procnotices"`
		}
		err = json.NewDecoder(resp.Body).Decode(&parsed)
		resp.Body.Close()
		if err != nil {
			continue
		}

		for _, n := range parsed.ProcNotices {
			if strings.EqualFold(strings.TrimSpace(n.NoticeType), "Contract Award") {
				continue
			}
			ctry := strings.ToLower(strings.TrimSpace(n.ProjectCountryName))
			if !worldBankASEANCountries[ctry] {
				continue
			}
			title := strings.TrimSpace(n.BidDescription)
			if title == "" {
				title = strings.TrimSpace(n.ProjectName)
			}
			if title == "" || n.ID == "" {
				continue
			}
			link := fmt.Sprintf("https://projects.worldbank.org/en/projects-operations/procurement-detail/%s", n.ID)
			if _, dup := seen[link]; dup {
				continue
			}
			seen[link] = struct{}{}

			if isOffScopeTitle(title, cfg.Keywords.OffScopeTitle) {
				continue
			}
			body := fmt.Sprintf("%s %s %s", n.NoticeType, n.ProcurementGroup, n.NoticeText)
			if !isRelevant(title, body, true, cfg.Keywords.Regions, cfg.Keywords.Themes) {
				continue
			}
			oppType := classifyOpportunity(title+" "+body, cfg.OpportunityTypes.RFPMarkers, cfg.OpportunityTypes.GrantMarkers)
			if oppType == "" {
				oppType = "RFP/Tender"
			}
			closeDate := parseAPIDate(n.SubmissionDeadlineDate, "TBD")
			openDate := parseAPIDate(n.NoticeDate, now.Format("2006-01-02"))
			out = append(out, Opportunity{
				ID:             hashID(link),
				Type:           oppType,
				Title:          title,
				Source:         src.Name,
				OpenDate:       openDate,
				CloseDate:      closeDate,
				PipelineStage:  "1-Triage",
				DeadlineStatus: computeDeadlineStatus(closeDate, now),
				URL:            link,
			})
		}
	}
	return out, nil
}
