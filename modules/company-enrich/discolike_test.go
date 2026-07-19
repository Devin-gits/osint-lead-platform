package companyenrich

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscolikeProviderMissingKey(t *testing.T) {
	tp := newDiscolikeProvider()
	tp.apiKey = "" // ensure missing
	res, err := tp.Enrich(context.Background(), Input{Domain: "example.com", PermissionRef: "DEMO-1"}, Fields{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "skipped" {
		t.Errorf("status = %q, want skipped", res.Status)
	}
	if res.Error != "DISCOLIKE_API_KEY not set" {
		t.Errorf("error = %q", res.Error)
	}
}

func TestDiscolikeProviderProfileMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/vendors" {
			http.Error(w, `{"detail":"plan gated"}`, http.StatusForbidden)
			return
		}
		if r.URL.Path != "/profile" {
			http.NotFound(w, r)
			return
		}
		key := r.Header.Get("x-discolike-key")
		if key != "test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		body := map[string]interface{}{
			"domain":      "example.com",
			"name":        "Example Inc",
			"description": "An example company",
			"employees":   "201-500",
			"address": map[string]interface{}{
				"country": "US",
				"state":   "CA",
				"city":    "San Francisco",
			},
			"industry_groups": map[string]interface{}{
				"Software": 0.95,
				"SaaS":     0.80,
			},
			"social_urls": []interface{}{
				"https://linkedin.com/company/example",
				"https://twitter.com/example",
			},
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer server.Close()

	tp := newDiscolikeProvider()
	tp.apiKey = "test-key"
	tp.baseURL = server.URL
	tp.httpClient = server.Client()

	res, err := tp.Enrich(context.Background(), Input{Domain: "example.com", PermissionRef: "DEMO-1"}, Fields{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "ok" {
		t.Errorf("status = %q, want ok", res.Status)
	}
	if res.Fields.Name != "Example Inc" {
		t.Errorf("name = %q", res.Fields.Name)
	}
	if res.Fields.EmployeeCountRange != "201-500" {
		t.Errorf("employee_count_range = %q", res.Fields.EmployeeCountRange)
	}
	if res.Fields.Headquarters == nil || res.Fields.Headquarters.Country != "US" {
		t.Errorf("headquarters = %+v", res.Fields.Headquarters)
	}
	if len(res.Fields.Industry) < 1 || res.Fields.Industry[0] != "Software" {
		t.Errorf("industry = %v", res.Fields.Industry)
	}
	if res.Fields.SocialLinks["linkedin"] == "" {
		t.Errorf("linkedin link missing: %+v", res.Fields.SocialLinks)
	}
}

func TestDiscolikeProviderVendors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/profile":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"domain": "example.com"})
		case "/vendors":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"vendors": []interface{}{"AWS", "React"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tp := newDiscolikeProvider()
	tp.apiKey = "test-key"
	tp.baseURL = server.URL
	tp.httpClient = server.Client()

	res, err := tp.Enrich(context.Background(), Input{Domain: "example.com", PermissionRef: "DEMO-1"}, Fields{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !contains(res.Fields.TechStack, "AWS") || !contains(res.Fields.TechStack, "React") {
		t.Errorf("tech_stack = %v", res.Fields.TechStack)
	}
	if !contains(res.Fields.Sources, "discolike_vendors") {
		t.Errorf("sources missing discolike_vendors: %v", res.Fields.Sources)
	}
}

func TestDiscolikeProviderNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer server.Close()

	tp := newDiscolikeProvider()
	tp.apiKey = "test-key"
	tp.baseURL = server.URL
	tp.httpClient = server.Client()

	res, err := tp.Enrich(context.Background(), Input{Domain: "example.com", PermissionRef: "DEMO-1"}, Fields{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "error" {
		t.Errorf("status = %q, want error", res.Status)
	}
}
