package companyenrich

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	discolikeDefaultBaseURL = "https://api.discolike.com/v1"
	discolikeAuthHeader     = "x-discolike-key"
	discolikeDefaultTimeout = 30 * time.Second
)

// discolikeProvider is an optional paid adapter to the DiscoLike B2B API.
// It is active only when DISCOLIKE_API_KEY is set.
type discolikeProvider struct {
	name       string
	apiKey     string
	baseURL    string
	timeout    time.Duration
	httpClient *http.Client
}

func newDiscolikeProvider() *discolikeProvider {
	timeout := discolikeDefaultTimeout
	if d := os.Getenv("DISCOLIKE_TIMEOUT"); d != "" {
		if parsed, err := time.ParseDuration(d); err == nil {
			timeout = parsed
		}
	}
	return &discolikeProvider{
		name:       "discolike",
		apiKey:     strings.TrimSpace(os.Getenv("DISCOLIKE_API_KEY")),
		baseURL:    strings.TrimRight(os.Getenv("DISCOLIKE_BASE_URL"), "/"),
		timeout:    timeout,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (p *discolikeProvider) Name() string { return p.name }

func (p *discolikeProvider) Enrich(ctx context.Context, in Input, merged Fields) (ProviderResult, error) {
	if p.apiKey == "" {
		return ProviderResult{
			Status:     "skipped",
			SourceTool: "company-enrich/discolike",
			Fields:     emptyFields(),
			Error:      "DISCOLIKE_API_KEY not set",
		}, nil
	}

	base := p.baseURL
	if base == "" {
		base = discolikeDefaultBaseURL
	}

	out := emptyFields()
	out.Sources = []string{"discolike"}

	// Profile is the core firmographic endpoint.
	profile, err := p.getProfile(ctx, base, in.Domain)
	if err != nil {
		return ProviderResult{
			Status:     "error",
			SourceTool: "company-enrich/discolike",
			Fields:     out,
			Error:      err.Error(),
		}, nil
	}

	out = mapDiscolikeProfile(out, profile)

	// Vendors is a separate tech-stack endpoint; some plans may not have access.
	// We attempt it, but degrade gracefully if it fails or is plan-gated.
	vendors, vErr := p.getVendors(ctx, base, in.Domain)
	if vErr == nil && len(vendors) > 0 {
		out.TechStack = appendUnique(out.TechStack, vendors...)
		out.Sources = appendUnique(out.Sources, "discolike_vendors")
	}

	status := "partial"
	if fieldsSatisfied(out, defaultP0()) {
		status = "ok"
	}

	return ProviderResult{
		Status:     status,
		SourceTool: "company-enrich/discolike",
		Fields:     out,
	}, nil
}

func (p *discolikeProvider) getProfile(ctx context.Context, baseURL, domain string) (map[string]interface{}, error) {
	return p.getJSON(ctx, baseURL+"/profile", url.Values{"domain": {domain}})
}

func (p *discolikeProvider) getVendors(ctx context.Context, baseURL, domain string) ([]string, error) {
	data, err := p.getJSON(ctx, baseURL+"/vendors", url.Values{"domain": {domain}})
	if err != nil {
		return nil, err
	}
	// Response shape: dict of vendor -> details, or list of strings.
	switch v := data["vendors"].(type) {
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out, nil
	case map[string]interface{}:
		out := make([]string, 0, len(v))
		for k := range v {
			if k != "" {
				out = append(out, k)
			}
		}
		sort.Strings(out)
		return out, nil
	case []string:
		return v, nil
	}
	return nil, nil
}

func (p *discolikeProvider) getJSON(ctx context.Context, endpoint string, params url.Values) (map[string]interface{}, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(discolikeAuthHeader, p.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "osint-lead-platform/company-enrich")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("discolike %s returned %d: %s", u.Path, resp.StatusCode, string(body))
	}

	var out map[string]interface{}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func mapDiscolikeProfile(out Fields, profile map[string]interface{}) Fields {
	if v, ok := getString(profile, "domain"); ok {
		out.Domain = v
	}
	if v, ok := getString(profile, "name"); ok {
		out.Name = v
	}
	if out.Website == "" && out.Domain != "" {
		out.Website = "https://" + out.Domain
	}
	if v, ok := getString(profile, "description"); ok {
		out.Description = v
	}
	if v, ok := getString(profile, "employees"); ok {
		out.EmployeeCountRange = v
	}

	if addr, ok := profile["address"].(map[string]interface{}); ok {
		hq := &Headquarters{}
		if v, ok := getString(addr, "country"); ok {
			hq.Country = v
		}
		if v, ok := getString(addr, "state"); ok {
			hq.State = v
		}
		if v, ok := getString(addr, "city"); ok {
			hq.City = v
		}
		if v, ok := getString(addr, "address"); ok {
			hq.Address = v
		}
		if hq.Country != "" || hq.City != "" || hq.Address != "" {
			out.Headquarters = hq
		}
	}

	out.Industry = extractDiscolikeStrings(profile, "industry_groups")

	if socials, ok := profile["social_urls"]; ok {
		out.SocialLinks = map[string]string{}
		switch v := socials.(type) {
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					platform, normalized := normalizeSocialLink(s)
					if platform != "" {
						out.SocialLinks[platform] = normalized
					}
				}
			}
		case map[string]interface{}:
			for platform, val := range v {
				if s, ok := val.(string); ok && s != "" {
					out.SocialLinks[strings.ToLower(platform)] = s
				}
			}
		case []string:
			for _, s := range v {
				platform, normalized := normalizeSocialLink(s)
				if platform != "" {
					out.SocialLinks[platform] = normalized
				}
			}
		}
	}

	return out
}

func extractDiscolikeStrings(data map[string]interface{}, key string) []string {
	raw, ok := data[key]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return v
	case map[string]interface{}:
		// Likely map of name -> confidence. Take top by confidence.
		type pair struct {
			name       string
			confidence float64
		}
		pairs := make([]pair, 0, len(v))
		for k, val := range v {
			var conf float64
			switch n := val.(type) {
			case float64:
				conf = n
			case float32:
				conf = float64(n)
			case int:
				conf = float64(n)
			case string:
				conf, _ = strconv.ParseFloat(n, 64)
			}
			if k != "" {
				pairs = append(pairs, pair{name: k, confidence: conf})
			}
		}
		sort.Slice(pairs, func(i, j int) bool { return pairs[i].confidence > pairs[j].confidence })
		out := make([]string, 0, len(pairs))
		for _, p := range pairs {
			out = append(out, p.name)
		}
		return out
	}
	return nil
}

func getString(data map[string]interface{}, key string) (string, bool) {
	raw, ok := data[key]
	if !ok {
		return "", false
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v), v != ""
	case nil:
		return "", false
	}
	return fmt.Sprintf("%v", raw), true
}
