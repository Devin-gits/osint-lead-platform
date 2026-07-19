package companyenrich

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLocalProviderExtractionOnly(t *testing.T) {
	lp := newLocalProvider()
	lp.lookupGitHub = noGitHubLookup

	res, err := lp.Enrich(context.Background(), Input{
		Domain:        "example.com",
		Company:       "Example",
		URL:           "https://example.com",
		PermissionRef: "DEMO-1",
	}, Fields{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "ok" {
		t.Errorf("status = %q, want ok", res.Status)
	}
	if res.Fields.Name != "Example" {
		t.Errorf("name = %q", res.Fields.Name)
	}
	if res.Fields.Website != "https://example.com" {
		t.Errorf("website = %q", res.Fields.Website)
	}
}

func TestLocalProviderFallsBackToHumanizedDomain(t *testing.T) {
	lp := newLocalProvider()
	lp.lookupGitHub = noGitHubLookup

	res, err := lp.Enrich(context.Background(), Input{
		Domain:        "stripe.com",
		PermissionRef: "DEMO-1",
	}, Fields{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "ok" {
		t.Errorf("status = %q, want ok", res.Status)
	}
	if res.Fields.Name != "Stripe" {
		t.Errorf("name = %q, want Stripe", res.Fields.Name)
	}
	if res.Fields.Website != "https://stripe.com" {
		t.Errorf("website = %q", res.Fields.Website)
	}
}

func TestGitHubOrgLookup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orgs/stripe" {
			http.NotFound(w, r)
			return
		}
		body := map[string]interface{}{
			"name":        "Stripe",
			"description": "Payments infrastructure for the internet",
			"blog":        "https://stripe.com",
			"location":    "San Francisco, CA",
			"html_url":    "https://github.com/stripe",
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer server.Close()

	lp := newLocalProvider()
	lp.githubBase = server.URL
	lp.httpClient = server.Client()

	res, err := lp.Enrich(context.Background(), Input{
		Domain:        "stripe.com",
		PermissionRef: "DEMO-1",
		Extraction: &ExtractionInput{
			Status: "ok",
			Fields: ExtractionFields{
				SocialLinks: []string{"https://github.com/stripe"},
			},
		},
	}, Fields{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Fields.Name != "Stripe" {
		t.Errorf("name = %q", res.Fields.Name)
	}
	if res.Fields.Description != "Payments infrastructure for the internet" {
		t.Errorf("description = %q", res.Fields.Description)
	}
	if res.Fields.Headquarters == nil || res.Fields.Headquarters.City != "San Francisco, CA" {
		t.Errorf("headquarters = %+v", res.Fields.Headquarters)
	}
	if res.Fields.Sources[0] != "local" {
		t.Errorf("first source = %q", res.Fields.Sources[0])
	}
	if !contains(res.Fields.Sources, "github_public_api") {
		t.Errorf("expected github_public_api in sources: %v", res.Fields.Sources)
	}
}

func TestGitHubOrgLookup404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	lp := newLocalProvider()
	lp.githubBase = server.URL
	lp.httpClient = server.Client()

	res, err := lp.Enrich(context.Background(), Input{
		Domain:        "unknown.example",
		PermissionRef: "DEMO-1",
		Extraction: &ExtractionInput{
			Status: "ok",
			Fields: ExtractionFields{
				SocialLinks: []string{"https://github.com/unknownexample"},
			},
		},
	}, Fields{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "partial" && res.Status != "ok" {
		t.Errorf("status = %q", res.Status)
	}
}

func TestNormalizeSocialLink(t *testing.T) {
	cases := []struct {
		in       string
		platform string
	}{
		{"https://github.com/stripe", "github"},
		{"https://www.linkedin.com/company/stripe", "linkedin"},
		{"https://twitter.com/stripe", "twitter"},
	}
	for _, c := range cases {
		p, n := normalizeSocialLink(c.in)
		if p != c.platform {
			t.Errorf("normalizeSocialLink(%q) platform = %q, want %q", c.in, p, c.platform)
		}
		if n == "" {
			t.Errorf("normalizeSocialLink(%q) returned empty normalized", c.in)
		}
	}
}

func TestGitHubOrgFromURL(t *testing.T) {
	cases := map[string]string{
		"https://github.com/stripe":         "stripe",
		"https://github.com/stripe/connect": "stripe",
		"github.com/stripe":                 "stripe",
	}
	for in, want := range cases {
		if got := githubOrgFromURL(in); got != want {
			t.Errorf("githubOrgFromURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsGenericDomain(t *testing.T) {
	if !isGenericDomain("www") {
		t.Errorf("www should be generic")
	}
	if isGenericDomain("stripe") {
		t.Errorf("stripe should not be generic")
	}
}

func noGitHubLookup(ctx context.Context, client *http.Client, baseURL, org string) (githubOrg, error) {
	return githubOrg{}, nil
}
