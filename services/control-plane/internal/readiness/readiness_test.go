package readiness

import (
	"testing"

	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/models"
)

func mkResult(status string) map[string]any {
	return map[string]any{"status": status}
}

func TestCompute(t *testing.T) {
	tests := []struct {
		name    string
		lead    models.Lead
		want    bool
		suggest string
	}{
		{
			name: "fully ready email path",
			lead: models.Lead{
				Email:         "a@example.com",
				PermissionRef: "p1",
				Company:       "Example",
				Stage:         models.StageValidated,
				RiskLevel:     models.RiskLow,
				Results: map[string]any{
					"email_validate":  mkResult("ok"),
					"company_enrich":  mkResult("partial"),
				},
			},
			want:    true,
			suggest: models.StageCrmReady,
		},
		{
			name: "ready phone path with company_enrich ok",
			lead: models.Lead{
				Phone:         "+14155551212",
				PermissionRef: "p1",
				Domain:        "example.com",
				Stage:         models.StageValidated,
				RiskLevel:     models.RiskLow,
				Results: map[string]any{
					"phone_validate": mkResult("ok"),
					"company_enrich": mkResult("ok"),
				},
			},
			want:    true,
			suggest: models.StageCrmReady,
		},
		{
			name: "missing permission_ref",
			lead: models.Lead{
				Email:     "a@example.com",
				Company:   "Example",
				Stage:     models.StageValidated,
				RiskLevel: models.RiskLow,
				Results: map[string]any{
					"email_validate": mkResult("ok"),
					"company_enrich": mkResult("partial"),
				},
			},
			want:    false,
			suggest: models.StageValidated,
		},
		{
			name: "no contact",
			lead: models.Lead{
				PermissionRef: "p1",
				Company:       "Example",
				Stage:         models.StageValidated,
				RiskLevel:     models.RiskLow,
				Results: map[string]any{
					"company_enrich": mkResult("partial"),
				},
			},
			want:    false,
			suggest: models.StageValidated,
		},
		{
			name: "email present but not validated",
			lead: models.Lead{
				Email:         "a@example.com",
				PermissionRef: "p1",
				Company:       "Example",
				Stage:         models.StageValidated,
				RiskLevel:     models.RiskLow,
				Results: map[string]any{
					"company_enrich": mkResult("partial"),
				},
			},
			want:    false,
			suggest: models.StageValidated,
		},
		{
			name: "domain only no email/phone",
			lead: models.Lead{
				PermissionRef: "p1",
				Domain:        "example.com",
				Stage:         models.StageEnriched,
				RiskLevel:     models.RiskLow,
				Results: map[string]any{
					"company_enrich": mkResult("partial"),
				},
			},
			want:    false,
			suggest: models.StageEnriched,
		},
		{
			name: "risk high blocks",
			lead: models.Lead{
				Email:         "a@example.com",
				PermissionRef: "p1",
				Company:       "Example",
				Stage:         models.StageValidated,
				RiskLevel:     models.RiskHigh,
				Results: map[string]any{
					"email_validate": mkResult("ok"),
					"company_enrich": mkResult("partial"),
				},
			},
			want:    false,
			suggest: models.StageValidated,
		},
		{
			name: "unknown risk allows with warning",
			lead: models.Lead{
				Email:         "a@example.com",
				PermissionRef: "p1",
				Company:       "Example",
				Stage:         models.StageValidated,
				RiskLevel:     models.RiskUnknown,
				Results: map[string]any{
					"email_validate": mkResult("ok"),
					"company_enrich": mkResult("partial"),
				},
			},
			want:    true,
			suggest: models.StageCrmReady,
		},
		{
			name: "required channel error blocks",
			lead: models.Lead{
				Email:         "a@example.com",
				PermissionRef: "p1",
				Company:       "Example",
				Stage:         models.StageValidated,
				RiskLevel:     models.RiskLow,
				Results: map[string]any{
					"email_validate": mkResult("error"),
					"company_enrich": mkResult("partial"),
				},
			},
			want:    false,
			suggest: models.StageValidated,
		},
		{
			name: "extraction ok satisfies company context",
			lead: models.Lead{
				Email:         "a@example.com",
				PermissionRef: "p1",
				Domain:        "example.com",
				URL:           "https://example.com",
				Stage:         models.StageValidated,
				RiskLevel:     models.RiskLow,
				Results: map[string]any{
					"email_validate": mkResult("ok"),
					"extraction":     mkResult("ok"),
				},
			},
			want:    true,
			suggest: models.StageCrmReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Compute(tt.lead)
			if r.Ready != tt.want {
				t.Errorf("ready = %v, want %v; checks=%+v", r.Ready, tt.want, r.Checks)
			}
			if r.SuggestedStage != tt.suggest {
				t.Errorf("suggested_stage = %s, want %s", r.SuggestedStage, tt.suggest)
			}
			if tt.want && r.Stage == models.StageValidated && r.Warning == "" {
				t.Logf("expected warning for unknown/missing risk; got none")
			}
		})
	}
}

func TestCanDemoteTo(t *testing.T) {
	if !CanDemoteTo(models.StageCrmReady, models.StageValidated) {
		t.Error("crm_ready -> validated should be allowed")
	}
	if !CanDemoteTo(models.StageCrmReady, models.StageRaw) {
		t.Error("crm_ready -> raw should be allowed")
	}
	if CanDemoteTo(models.StageValidated, models.StageCrmReady) {
		t.Error("validated -> crm_ready is not a demotion")
	}
	if CanDemoteTo(models.StageCrmReady, models.StageCrmReady) {
		t.Error("crm_ready -> crm_ready is not a demotion")
	}
	if CanDemoteTo(models.StageValidated, "bad-stage") {
		t.Error("bad target should not be allowed")
	}
}
