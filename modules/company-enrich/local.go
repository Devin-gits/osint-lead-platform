package companyenrich

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// localProvider is the default, no-API-key provider. It derives company fields
// from the lead record and any prior extraction/domain-intel results. It may
// optionally call the public GitHub API for an org linked from extraction, but it
// never scrapes LinkedIn, Crunchbase, or job boards.
type localProvider struct {
	name        string
	httpClient  *http.Client
	githubBase  string
	lookupGitHub func(ctx context.Context, client *http.Client, baseURL, org string) (githubOrg, error)
}

func newLocalProvider() *localProvider {
	return &localProvider{
		name:       "local",
		httpClient: &http.Client{Timeout: 10 * time.Second},
		githubBase: "https://api.github.com",
		lookupGitHub: defaultGitHubLookup,
	}
}

func (p *localProvider) Name() string { return p.name }

func (p *localProvider) Enrich(ctx context.Context, in Input, merged Fields) (ProviderResult, error) {
	out := emptyFields()
	out.Sources = []string{"local"}

	domain, company, website := in.Domain, in.Company, in.URL

	// P0 fields from input normalization.
	out.Domain = domain
	out.Name = pickNonEmpty(out.Name, company)
	out.Website = website

	// Extraction fields.
	if in.Extraction != nil {
		if in.Extraction.Fields.CompanyName != "" && out.Name == "" {
			out.Name = in.Extraction.Fields.CompanyName
		}
		if in.Extraction.Fields.Description != "" && out.Description == "" {
			out.Description = in.Extraction.Fields.Description
		}
		if in.Extraction.Fields.Title != "" && out.Name == "" {
			// Title can be a weak fallback for name.
			out.Name = cleanTitle(in.Extraction.Fields.Title)
		}

		// Normalize social links from extraction.social_links.
		if out.SocialLinks == nil {
			out.SocialLinks = map[string]string{}
		}
		for _, link := range in.Extraction.Fields.SocialLinks {
			platform, normalized := normalizeSocialLink(link)
			if platform != "" && normalized != "" {
				if _, ok := out.SocialLinks[platform]; !ok {
					out.SocialLinks[platform] = normalized
				}
			}
		}
	}

	// Domain-intel hints.
	if in.DomainIntel != nil && in.DomainIntel.WebCheck.SSL != nil {
		// SSL subject is a weak company-name hint (often the domain itself).
		if out.Name == "" {
			hint := cleanSSLSubject(in.DomainIntel.WebCheck.SSL.Subject)
			if hint != "" && !strings.Contains(hint, ".") {
				out.Name = hint
			}
		}
	}

	// GitHub public-org lookup.
	if ghOrg := p.extractGitHubOrg(in, out.SocialLinks); ghOrg != "" {
		if gho, err := p.lookupGitHub(ctx, p.httpClient, p.githubBase, ghOrg); err == nil {
			if out.Name == "" && gho.Name != "" {
				out.Name = gho.Name
			}
			if out.Description == "" && gho.Description != "" {
				out.Description = gho.Description
			}
			if out.Website == "" && gho.Blog != "" {
				out.Website = normalizeWebsite(gho.Blog)
			}
			if gho.Location != "" {
				if out.Headquarters == nil {
					out.Headquarters = &Headquarters{}
				}
				out.Headquarters.City = gho.Location
			}
			if _, ok := out.SocialLinks["github"]; !ok && gho.HTMLURL != "" {
				out.SocialLinks["github"] = gho.HTMLURL
			}
			out.Sources = appendUnique(out.Sources, "github_public_api")
		}
	}

	// Final normalization.
	if out.Website == "" && out.Domain != "" {
		out.Website = "https://" + out.Domain
	}
	if out.SocialLinks == nil {
		out.SocialLinks = map[string]string{}
	}

	status := "partial"
	if fieldsSatisfied(out, defaultP0()) {
		status = "ok"
	}

	return ProviderResult{
		Status:     status,
		SourceTool: "company-enrich/local",
		Fields:     out,
	}, nil
}

func (p *localProvider) extractGitHubOrg(in Input, socials map[string]string) string {
	// 1. If extraction social_links contains a github URL, use that org.
	if in.Extraction != nil {
		for _, link := range in.Extraction.Fields.SocialLinks {
			if org := githubOrgFromURL(link); org != "" {
				return org
			}
		}
	}
	if org, ok := socials["github"]; ok {
		return githubOrgFromURL(org)
	}

	// 2. Optional weak guess from the domain root. Disabled by default to avoid
	// surprise network calls; enabled only when explicitly requested.
	if in.Domain != "" && isDomainGuessEnabled() {
		root := strings.Split(in.Domain, ".")[0]
		// Only guess if it looks like a plausible org name (alphanumeric, not generic).
		if len(root) >= 3 && !isGenericDomain(root) {
			return root
		}
	}
	return ""
}

func isDomainGuessEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("COMPANY_ENRICH_GITHUB_DOMAIN_GUESS")))
	return v == "1" || v == "true" || v == "yes"
}

type githubOrg struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Blog        string `json:"blog"`
	Location    string `json:"location"`
	HTMLURL     string `json:"html_url"`
}

func defaultGitHubLookup(ctx context.Context, client *http.Client, baseURL, org string) (githubOrg, error) {
	var out githubOrg
	if client == nil {
		client = http.DefaultClient
	}
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return out, err
	}
	u.Path = "/orgs/" + url.PathEscape(org)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return out, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "osint-lead-platform/company-enrich")

	resp, err := client.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return out, err
	}
	return out, nil
}

func githubOrgFromURL(link string) string {
	link = strings.TrimSpace(link)
	if link == "" {
		return ""
	}
	link = strings.TrimPrefix(link, "http://")
	link = strings.TrimPrefix(link, "https://")
	link = strings.TrimPrefix(link, "www.")
	link = strings.TrimPrefix(link, "github.com/")
	link = strings.Trim(link, "/")
	if strings.Contains(link, "/") {
		parts := strings.Split(link, "/")
		if len(parts) >= 2 {
			link = parts[0] // org; ignore repository path
		}
	}
	if strings.Contains(link, "?") {
		link = link[:strings.Index(link, "?")]
	}
	return link
}

func normalizeSocialLink(link string) (platform, normalized string) {
	link = strings.TrimSpace(link)
	if link == "" {
		return "", ""
	}
	if !strings.Contains(link, "://") {
		link = "https://" + link
	}

	u, err := url.Parse(link)
	if err != nil {
		return "", ""
	}
	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")

	switch {
	case strings.Contains(host, "linkedin.com"):
		return "linkedin", link
	case strings.Contains(host, "twitter.com") || strings.Contains(host, "x.com"):
		return "twitter", link
	case strings.Contains(host, "github.com"):
		return "github", link
	case strings.Contains(host, "facebook.com"):
		return "facebook", link
	case strings.Contains(host, "instagram.com"):
		return "instagram", link
	case strings.Contains(host, "youtube.com"):
		return "youtube", link
	case strings.Contains(host, "tiktok.com"):
		return "tiktok", link
	}
	return "", ""
}

func normalizeWebsite(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	return raw
}

func cleanTitle(title string) string {
	title = strings.TrimSpace(title)
	// "Example Corp | Home" -> "Example Corp"
	if idx := strings.IndexAny(title, "|-–—"); idx > 0 {
		title = strings.TrimSpace(title[:idx])
	}
	return title
}

func cleanSSLSubject(subject string) string {
	subject = strings.TrimSpace(subject)
	// Strip leading "*." and common CN prefixes.
	subject = strings.TrimPrefix(subject, "*.")
	return subject
}

func pickNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func isGenericDomain(root string) bool {
	generic := map[string]bool{
		"www": true, "app": true, "web": true, "mail": true, "api": true,
		"admin": true, "blog": true, "shop": true, "store": true,
	}
	return generic[strings.ToLower(root)]
}
