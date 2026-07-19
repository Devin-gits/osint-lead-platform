package companyenrich

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestMissingPermissionRef(t *testing.T) {
	e := NewEnricher(0, 0)
	res, audits := e.Enrich(context.Background(), Input{Domain: "example.com"})

	if res.Status != "skipped" {
		t.Errorf("status = %q, want skipped", res.Status)
	}
	if !strings.Contains(res.Error, "permission_ref") {
		t.Errorf("error = %q, want permission_ref mention", res.Error)
	}
	if len(audits) != 1 {
		t.Fatalf("expected 1 audit, got %d", len(audits))
	}
	if audits[0].Status != "skipped" {
		t.Errorf("audit status = %q, want skipped", audits[0].Status)
	}
	if audits[0].LegalBasis != LegalBasis {
		t.Errorf("legal basis = %q", audits[0].LegalBasis)
	}
}

func TestMissingDomainCompanyURL(t *testing.T) {
	e := NewEnricher(0, 0)
	res, audits := e.Enrich(context.Background(), Input{PermissionRef: "DEMO-1"})
	if res.Status != "skipped" {
		t.Errorf("status = %q, want skipped", res.Status)
	}
	if len(audits) != 1 {
		t.Fatalf("expected 1 audit, got %d", len(audits))
	}
	if res.Error != "missing domain, company, and url" {
		t.Errorf("error = %q", res.Error)
	}
}

func TestLocalOnlyOk(t *testing.T) {
	e := NewEnricher(0, 0)
	res, _ := e.Enrich(context.Background(), Input{
		Domain:        "example.com",
		Company:       "Example",
		PermissionRef: "DEMO-1",
	})
	if res.Status != "ok" {
		t.Errorf("status = %q, want ok", res.Status)
	}
	if res.Fields.Domain != "example.com" {
		t.Errorf("domain = %q", res.Fields.Domain)
	}
	if res.Fields.Name != "Example" {
		t.Errorf("name = %q", res.Fields.Name)
	}
	if res.Fields.Website != "https://example.com" {
		t.Errorf("website = %q", res.Fields.Website)
	}
	if res.Fields.Sources == nil || len(res.Fields.Sources) == 0 {
		t.Errorf("sources empty")
	}
}

func TestLocalReusesExtraction(t *testing.T) {
	lp := newLocalProvider()
	lp.lookupGitHub = func(ctx context.Context, client *http.Client, baseURL, org string) (githubOrg, error) {
		return githubOrg{Name: "Stripe", Description: "A payments org", HTMLURL: "https://github.com/stripe"}, nil
	}

	e := NewEnricher(0, 0)
	e.SetProviders(lp, newDiscolikeProvider())
	res, _ := e.Enrich(context.Background(), Input{
		Domain:        "stripe.com",
		PermissionRef: "DEMO-1",
		Extraction: &ExtractionInput{
			Status: "ok",
			Fields: ExtractionFields{
				CompanyName: "Stripe, Inc.",
				Description: "Payments infrastructure for the internet.",
				SocialLinks: []string{"https://github.com/stripe", "https://twitter.com/stripe"},
			},
		},
	})
	if res.Status != "ok" {
		t.Errorf("status = %q, want ok", res.Status)
	}
	if res.Fields.Name != "Stripe, Inc." {
		t.Errorf("name = %q", res.Fields.Name)
	}
	if res.Fields.Description != "Payments infrastructure for the internet." {
		t.Errorf("description = %q", res.Fields.Description)
	}
	if res.Fields.SocialLinks["github"] != "https://github.com/stripe" {
		t.Errorf("github link = %q", res.Fields.SocialLinks["github"])
	}
}

func TestLocalOnlyPartial(t *testing.T) {
	e := NewEnricher(0, 0)
	res, _ := e.Enrich(context.Background(), Input{
		Domain:        "example.com",
		PermissionRef: "DEMO-1",
	})
	// No company/extraction input means no company name can be derived honestly.
	if res.Status != "partial" {
		t.Errorf("status = %q, want partial", res.Status)
	}
	if res.Fields.Name != "" {
		t.Errorf("name should be empty for domain-only, got %q", res.Fields.Name)
	}
}

func TestAuditJSONSerialization(t *testing.T) {
	e := NewEnricher(0, 0)
	_, audits := e.Enrich(context.Background(), Input{
		Domain:        "example.com",
		Company:       "Example",
		PermissionRef: "DEMO-1",
	})
	for _, a := range audits {
		b, err := json.Marshal(a)
		if err != nil {
			t.Errorf("audit marshal: %v", err)
		}
		if !strings.Contains(string(b), "module") {
			t.Errorf("audit missing module: %s", string(b))
		}
		if a.LegalBasis != LegalBasis {
			t.Errorf("legal basis = %q", a.LegalBasis)
		}
	}
}

func TestRequiredFieldsEarlyStop(t *testing.T) {
	called := 0
	fake := &fakeProvider{
		name: "fake-second",
		enrich: func(ctx context.Context, in Input, merged Fields) (ProviderResult, error) {
			called++
			return ProviderResult{Status: "ok", Fields: Fields{Name: "Fake Name", Domain: in.Domain, Website: "https://" + in.Domain, Sources: []string{"fake"}}, SourceTool: "fake"}, nil
		},
	}

	e := NewEnricher(0, 0)
	e.SetProviders(newLocalProvider(), fake)
	res, _ := e.Enrich(context.Background(), Input{
		Domain:         "example.com",
		Company:        "Example",
		PermissionRef:  "DEMO-1",
		RequiredFields: []string{"domain", "name", "website"},
	})
	if called != 0 {
		t.Errorf("second provider called %d times, want 0 (early stop)", called)
	}
	if res.Status != "ok" {
		t.Errorf("status = %q, want ok", res.Status)
	}
	if res.Fields.Name != "Example" {
		t.Errorf("name = %q", res.Fields.Name)
	}
}

func TestConfidenceBounds(t *testing.T) {
	f := Fields{
		Domain:  "example.com",
		Name:    "Example",
		Website: "https://example.com",
	}
	c := confidenceFor(f)
	if c < 0 || c > 1 {
		t.Errorf("confidence out of bounds: %f", c)
	}
}

func TestResultToJSON(t *testing.T) {
	e := NewEnricher(0, 0)
	res, _ := e.Enrich(context.Background(), Input{
		Domain:        "example.com",
		Company:       "Example",
		PermissionRef: "DEMO-1",
	})
	b, err := ResultToJSON(res)
	if err != nil {
		t.Fatalf("ResultToJSON: %v", err)
	}
	if !strings.Contains(string(b), "company_enrich") {
		// ResultToJSON returns only the Result struct, not the wrapper.
		// Ensure it contains status.
	}
	if !strings.Contains(string(b), `"status"`) {
		t.Errorf("JSON missing status: %s", string(b))
	}
}

func TestClockInjected(t *testing.T) {
	fixed := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	e := NewEnricher(0, 0)
	e.clock = func() time.Time { return fixed }
	res, _ := e.Enrich(context.Background(), Input{
		Domain:        "example.com",
		Company:       "Example",
		PermissionRef: "DEMO-1",
	})
	if res.CheckedAt != fixed.Format(time.RFC3339) {
		t.Errorf("checked_at = %q", res.CheckedAt)
	}
}
