package extraction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	firecrawlAPIKeyEnv  = "FIRECRAWL_API_KEY"
	firecrawlBaseURLEnv = "FIRECRAWL_BASE_URL"
)

var defaultFirecrawlBaseURL = "https://api.firecrawl.dev/v1"

type firecrawlRunner struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func newFirecrawlRunner() *firecrawlRunner {
	base := os.Getenv(firecrawlBaseURLEnv)
	if base == "" {
		base = defaultFirecrawlBaseURL
	}
	return &firecrawlRunner{
		apiKey:  os.Getenv(firecrawlAPIKeyEnv),
		baseURL: strings.TrimRight(base, "/"),
		client:  &http.Client{Timeout: 45 * time.Second},
	}
}

func (f *firecrawlRunner) sourceTool() string {
	return SourceToolFirecrawl
}

func (f *firecrawlRunner) run(ctx context.Context, url string, timeout time.Duration) (Result, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	if f.apiKey == "" {
		return Result{
			Status:    "skipped",
			URL:       url,
			SourceTool: SourceToolFirecrawl,
			CheckedAt: now,
			Metadata:  Metadata{Backend: BackendFirecrawl, LegalBasis: LegalBasis},
			Error:     fmt.Sprintf("%s is not set; Firecrawl adapter is optional — see README", firecrawlAPIKeyEnv),
		}, nil
	}

	payload := map[string]interface{}{
		"url":             url,
		"formats":         []string{"markdown"},
		"onlyMainContent": true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return Result{
			Status:    "error",
			URL:       url,
			SourceTool: SourceToolFirecrawl,
			CheckedAt: now,
			Metadata:  Metadata{Backend: BackendFirecrawl, LegalBasis: LegalBasis},
			Error:     fmt.Sprintf("marshal request: %v", err),
		}, nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", f.baseURL+"/scrape", bytes.NewReader(body))
	if err != nil {
		return Result{
			Status:    "error",
			URL:       url,
			SourceTool: SourceToolFirecrawl,
			CheckedAt: now,
			Metadata:  Metadata{Backend: BackendFirecrawl, LegalBasis: LegalBasis},
			Error:     fmt.Sprintf("build request: %v", err),
		}, nil
	}
	req.Header.Set("Authorization", "Bearer "+f.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return Result{
			Status:    "error",
			URL:       url,
			SourceTool: SourceToolFirecrawl,
			CheckedAt: now,
			Metadata:  Metadata{Backend: BackendFirecrawl, LegalBasis: LegalBasis},
			Error:     fmt.Sprintf("firecrawl request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return Result{
			Status:    "error",
			URL:       url,
			SourceTool: SourceToolFirecrawl,
			CheckedAt: now,
			Metadata:  Metadata{Backend: BackendFirecrawl, LegalBasis: LegalBasis, HTTPStatus: resp.StatusCode},
			Error:     fmt.Sprintf("read firecrawl response: %v", err),
		}, nil
	}

	var parsed firecrawlResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return Result{
			Status:    "error",
			URL:       url,
			SourceTool: SourceToolFirecrawl,
			CheckedAt: now,
			Metadata:  Metadata{Backend: BackendFirecrawl, LegalBasis: LegalBasis, HTTPStatus: resp.StatusCode},
			Error:     fmt.Sprintf("parse firecrawl response: %v", err),
		}, nil
	}

	if !parsed.Success {
		errMsg := parsed.Error
		if errMsg == "" {
			errMsg = "firecrawl returned failure"
		}
		return Result{
			Status:    "error",
			URL:       url,
			SourceTool: SourceToolFirecrawl,
			CheckedAt: now,
			Metadata:  Metadata{Backend: BackendFirecrawl, LegalBasis: LegalBasis, HTTPStatus: resp.StatusCode},
			Error:     errMsg,
		}, nil
	}

	markdown := parsed.Data.Markdown
	finalURL := parsed.Data.Metadata.SourceURL
	if finalURL == "" {
		finalURL = url
	}
	status := parsed.Data.Metadata.StatusCode
	if status == 0 {
		status = resp.StatusCode
	}

	links := []string{}
	if parsed.Data.Links != nil {
		links = parsed.Data.Links
	}

	fields := extractFieldsFromText(markdown, links, finalURL)

	res := Result{
		Status:     "ok",
		URL:        url,
		FinalURL:   finalURL,
		SourceTool: SourceToolFirecrawl,
		Confidence: confidenceFor(fields),
		Fields:     fields,
		Metadata: Metadata{
			Backend:    BackendFirecrawl,
			LegalBasis: LegalBasis,
			HTTPStatus: status,
			RawBytes:   len(markdown),
		},
		CheckedAt: now,
	}
	if len(markdown) > MaxRawMarkdown {
		res.RawMarkdown = markdown[:MaxRawMarkdown]
		res.Metadata.Truncated = true
		res.Metadata.RawBytes = MaxRawMarkdown
	} else {
		res.RawMarkdown = markdown
		res.Metadata.RawBytes = len(markdown)
	}
	return res, nil
}

type firecrawlResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Markdown string `json:"markdown"`
		Links    []string `json:"links"`
		Metadata struct {
			StatusCode int    `json:"statusCode"`
			SourceURL  string `json:"sourceURL"`
			Title      string `json:"title"`
			Description string `json:"description"`
		} `json:"metadata"`
	} `json:"data"`
	Error string `json:"error"`
}

var (
	emailRe    = regexp.MustCompile(`(?i)([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)
	phoneRe    = regexp.MustCompile(`(?i)(?:\+?\d{1,3}[-.\s]?)?(?:\(?\d{2,4}\)?[-.\s]?)?\d{2,4}[-.\s]?\d{2,4}(?:[-.\s]?\d{2,9})?`)
	socialRe   = regexp.MustCompile(`(?i)https?://(?:www\.)?(twitter\.com|x\.com|facebook\.com|linkedin\.com|instagram\.com|youtube\.com|tiktok\.com|github\.com|gitlab\.com)/[^\s\"]+`)
	contactRe  = regexp.MustCompile(`(?i)(/contact|/about|/support|/help|/careers|/jobs)`)
)

func extractFieldsFromText(markdown string, links []string, baseURL string) Fields {
	text := markdown
	emails := unique(emailRe.FindAllString(text, -1))

	rawPhones := phoneRe.FindAllString(text, -1)
	var phones []string
	for _, p := range rawPhones {
		if clean := normalizePhone(p); clean != "" {
			phones = append(phones, clean)
		}
	}
	phones = unique(phones)

	socialLinks := unique(socialRe.FindAllString(text, -1))
	contactURLs := []string{}
	for _, l := range links {
		if contactRe.MatchString(l) {
			contactURLs = append(contactURLs, l)
		}
	}
	contactURLs = unique(contactURLs)

	title := ""
	description := ""
	companyName := ""

	// Firecrawl metadata isn't available here in raw markdown; use heuristics.
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if title == "" && !strings.HasPrefix(line, "#") && len(line) < 120 {
			title = line
			continue
		}
		if description == "" && len(line) > 40 && len(line) < 500 {
			description = line
			break
		}
	}
	if title != "" {
		companyName = strings.Split(title, "|")[0]
		companyName = strings.Split(companyName, "-")[0]
		companyName = strings.TrimSpace(companyName)
	}

	return Fields{
		CompanyName: companyName,
		Emails:      emails,
		Phones:      phones,
		SocialLinks: socialLinks,
		ContactURLs: contactURLs,
		Title:       title,
		Description: truncate(description, 1000),
	}
}

func normalizePhone(raw string) string {
	digits := regexp.MustCompile(`\D`).ReplaceAllString(raw, "")
	if len(digits) < 7 || len(digits) > 15 {
		return ""
	}
	if len(uniqueChars(digits)) <= 3 {
		return ""
	}
	return strings.TrimSpace(raw)
}

func uniqueChars(s string) []rune {
	m := map[rune]struct{}{}
	for _, r := range s {
		m[r] = struct{}{}
	}
	out := make([]rune, 0, len(m))
	for r := range m {
		out = append(out, r)
	}
	return out
}

func unique(ss []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, s := range ss {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
