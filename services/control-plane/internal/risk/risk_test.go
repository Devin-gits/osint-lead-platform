package risk

import (
	"testing"

	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/models"
)

func TestCompute_Unknown(t *testing.T) {
	lead := models.Lead{}
	r := Compute(lead)
	if r.Level != models.RiskUnknown {
		t.Fatalf("expected unknown, got %q", r.Level)
	}
	if r.Score == nil || *r.Score != 0 {
		t.Fatalf("expected score 0, got %v", r.Score)
	}
	if len(r.Factors) != 0 {
		t.Fatalf("expected no factors, got %v", r.Factors)
	}
}

func TestCompute_BandsAndUnknown(t *testing.T) {
	cases := []struct {
		name  string
		lead  models.Lead
		want  *float64
		level string
	}{
		{
			name: "no signals",
			lead: models.Lead{},
			want: ptrf(0),
			level: models.RiskUnknown,
		},
		{
			name: "email ok with valid mx and syntax",
			lead: models.Lead{
				Email: "support@example.com",
				Results: map[string]any{
					"email_validate": map[string]any{
						"status":        "ok",
						"syntax_valid":  true,
						"has_mx_records": true,
					},
				},
			},
			want:  ptrf(0),
			level: models.RiskLow,
		},
		{
			name: "email ok plus company_enrich ok",
			lead: models.Lead{
				Email:   "support@example.com",
				Company: "Example",
				Results: map[string]any{
					"email_validate": map[string]any{
						"status":        "ok",
						"syntax_valid":  true,
						"has_mx_records": true,
					},
					"company_enrich": map[string]any{
						"status": "ok",
					},
				},
			},
			want:  ptrf(0),
			level: models.RiskLow,
		},
		{
			name: "disposable email",
			lead: models.Lead{
				Email: "temp@mailinator.com",
				Results: map[string]any{
					"email_validate": map[string]any{
						"status":        "ok",
						"is_disposable": true,
						"syntax_valid":  true,
						"has_mx_records": true,
					},
				},
			},
			want:  ptrf(5), // +15 disposable -10 green contact
			level: models.RiskLow,
		},
		{
			name: "email_validate error",
			lead: models.Lead{
				Email: "bad@example.com",
				Results: map[string]any{
					"email_validate": map[string]any{
						"status": "error",
					},
				},
			},
			want:  ptrf(40),
			level: models.RiskMedium,
		},
		{
			name: "email not validated",
			lead: models.Lead{
				Email:   "a@example.com",
				Company: "Example",
			},
			want:  ptrf(10),
			level: models.RiskLow,
		},
		{
			name: "phone invalid but parseable",
			lead: models.Lead{
				Phone: "+1 555 444 1212",
				Results: map[string]any{
					"phone_validate": map[string]any{
						"status":          "ok",
						"format_valid":      true,
						"is_valid_number":   false,
					},
				},
			},
			want:  ptrf(15), // invalid +25, minus green contact -10
			level: models.RiskLow,
		},
		{
			name: "high from email error + disposable",
			lead: models.Lead{
				Email: "bad@temp.com",
				Results: map[string]any{
					"email_validate": map[string]any{
						"status":        "error",
						"is_disposable": true,
					},
				},
			},
			want:  ptrf(40), // flag points only counted when status is ok/partial
			level: models.RiskMedium,
		},
		{
			name: "very high from email + phone errors",
			lead: models.Lead{
				Email: "bad@example.com",
				Phone: "bad",
				Results: map[string]any{
					"email_validate": map[string]any{"status": "error"},
					"phone_validate": map[string]any{"status": "error"},
				},
			},
			want:  ptrf(75), // 40 + 35
			level: models.RiskHigh,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := Compute(tc.lead)
			if (r.Score == nil) != (tc.want == nil) {
				t.Fatalf("score nil mismatch: got %v, want %v", r.Score, tc.want)
			}
			if r.Score != nil && tc.want != nil && *r.Score != *tc.want {
				t.Fatalf("score = %v, want %v; factors=%v", *r.Score, *tc.want, r.Factors)
			}
			if r.Level != tc.level {
				t.Fatalf("level = %q, want %q", r.Level, tc.level)
			}
		})
	}
}

func TestCompute_FlagCap(t *testing.T) {
	lead := models.Lead{
		Email: "x@example.com",
		Results: map[string]any{
			"email_validate": map[string]any{
				"status":           "ok",
				"is_disposable":    true,
				"is_role_account":  true,
				"is_free_provider": true,
				"syntax_valid":     true,
				"has_mx_records":    true,
			},
		},
	}
	r := Compute(lead)
	// 15+15+15 capped at 30, then -10 green contact
	want := 20.0
	if r.Score == nil || *r.Score != want {
		t.Fatalf("score = %v, want %v; factors=%v", r.Score, want, r.Factors)
	}
	if r.Level != models.RiskLow {
		t.Fatalf("level = %q, want low", r.Level)
	}
}

func TestCompute_DomainIntelHardFail(t *testing.T) {
	lead := models.Lead{
		Email:  "support@example.com",
		Domain: "example.com",
		Results: map[string]any{
			"email_validate": map[string]any{
				"status":        "ok",
				"syntax_valid":  true,
				"has_mx_records": true,
			},
			"domain_intel": map[string]any{
				"status": "ok",
				"web_check": map[string]any{
					"resolvable": false,
					"ssl": map[string]any{
						"valid": false,
					},
					"http": map[string]any{
						"status_code": 500.0,
					},
				},
			},
		},
	}
	r := Compute(lead)
	// +10 hard fail, -10 green contact
	want := 0.0
	if r.Score == nil || *r.Score != want {
		t.Fatalf("score = %v, want %v; factors=%v", r.Score, want, r.Factors)
	}
	if r.Level != models.RiskLow {
		t.Fatalf("level = %q, want low", r.Level)
	}
}

func TestCompute_ExtractionOkBonus(t *testing.T) {
	lead := models.Lead{
		Email:   "support@example.com",
		Company: "Example",
		Results: map[string]any{
			"email_validate": map[string]any{
				"status":        "ok",
				"syntax_valid":  true,
				"has_mx_records": true,
			},
			"extraction": map[string]any{
				"status": "ok",
			},
		},
	}
	r := Compute(lead)
	// 0 -10 green contact -5 extraction ok
	want := 0.0
	if r.Score == nil || *r.Score != want {
		t.Fatalf("score = %v, want %v; factors=%v", r.Score, want, r.Factors)
	}
}

func ptrf(v float64) *float64 {
	return &v
}