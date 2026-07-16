package extraction

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFirecrawlRunner_MissingKey(t *testing.T) {
	t.Setenv(firecrawlAPIKeyEnv, "")
	r := newFirecrawlRunner()
	res, err := r.run(context.Background(), "https://example.com", 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "skipped" {
		t.Errorf("status = %q, want skipped", res.Status)
	}
	if res.Error == "" {
		t.Error("expected error message for missing API key")
	}
}

func TestFirecrawlRunner_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/scrape" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("authorization = %q, want Bearer test-key", auth)
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req["url"] != "https://example.com" {
			t.Errorf("url = %v", req["url"])
		}
		resp := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"markdown": "Acme Inc.\nhello@acme.com Call +1 555-123-4567.",
				"links":    []string{"https://example.com/contact"},
				"metadata": map[string]interface{}{
					"statusCode": 200,
					"sourceURL":  "https://example.com/",
					"title":      "Acme Inc.",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Setenv(firecrawlAPIKeyEnv, "test-key")
	t.Setenv(firecrawlBaseURLEnv, server.URL+"/v1")
	r := newFirecrawlRunner()
	res, err := r.run(context.Background(), "https://example.com", 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "ok" {
		t.Fatalf("status = %q, want ok; error=%q", res.Status, res.Error)
	}
	if res.Fields.CompanyName != "Acme Inc." {
		t.Errorf("company_name = %q", res.Fields.CompanyName)
	}
	foundEmail := false
	for _, e := range res.Fields.Emails {
		if e == "hello@acme.com" {
			foundEmail = true
			break
		}
	}
	if !foundEmail {
		t.Errorf("emails missing hello@acme.com: %v", res.Fields.Emails)
	}
	if res.Metadata.HTTPStatus != 200 {
		t.Errorf("http_status = %d", res.Metadata.HTTPStatus)
	}
}

func TestFirecrawlRunner_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "invalid key"})
	}))
	defer server.Close()

	t.Setenv(firecrawlAPIKeyEnv, "bad-key")
	t.Setenv(firecrawlBaseURLEnv, server.URL+"/v1")
	r := newFirecrawlRunner()
	res, err := r.run(context.Background(), "https://example.com", 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "error" {
		t.Errorf("status = %q, want error", res.Status)
	}
	if res.Error == "" {
		t.Error("expected error message for API failure")
	}
}
