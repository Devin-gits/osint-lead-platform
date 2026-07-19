// Package readiness implements the deterministic crm_ready promotion gate.
package readiness

import (
	"fmt"
	"strings"

	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/models"
)

// Check is one readiness assertion.
type Check struct {
	Name     string `json:"name"`
	Pass     bool   `json:"pass"`
	Message  string `json:"message"`
	Required bool   `json:"required"`
}

// Report is the complete readiness evaluation for a lead.
type Report struct {
	Ready          bool    `json:"ready"`
	Stage          string  `json:"stage"`
	SuggestedStage string  `json:"suggested_stage"`
	Warning        string  `json:"warning,omitempty"`
	Checks         []Check `json:"checks"`
}

// stageOrder maps each stage to its numeric order.
var stageOrder = map[string]int{
	models.StageRaw:       0,
	models.StageEnriched:  1,
	models.StageValidated: 2,
	models.StageCrmReady:  3,
}

// allowedDemoteTargets are the stages a lead can be demoted to from crm_ready.
var allowedDemoteTargets = map[string]bool{
	models.StageRaw:       true,
	models.StageEnriched:  true,
	models.StageValidated: true,
}

// Compute returns a readiness report for the given lead.
func Compute(lead models.Lead) Report {
	checks := []Check{}

	// 1. permission_ref
	permCheck := Check{
		Name:     "permission_ref",
		Required: true,
		Pass:     strings.TrimSpace(lead.PermissionRef) != "",
	}
	if permCheck.Pass {
		permCheck.Message = "permission_ref is set"
	} else {
		permCheck.Message = "permission_ref is required"
	}
	checks = append(checks, permCheck)

	// 2. identity contact
	hasEmail := strings.TrimSpace(lead.Email) != ""
	hasPhone := strings.TrimSpace(lead.Phone) != ""
	identityCheck := Check{
		Name:     "identity_contact",
		Required: true,
		Pass:     hasEmail || hasPhone,
	}
	if identityCheck.Pass {
		if hasEmail {
			identityCheck.Message = "email is present"
		} else {
			identityCheck.Message = "phone is present"
		}
	} else {
		identityCheck.Message = "at least one of email or phone is required"
	}
	checks = append(checks, identityCheck)

	// 3. contact validation
	requiredChannel := "email_validate"
	if !hasEmail && hasPhone {
		requiredChannel = "phone_validate"
	}
	requiredStatus := moduleStatus(lead.Results, requiredChannel)
	contactCheck := Check{
		Name:     requiredChannel,
		Required: true,
		Pass:     requiredStatus == "ok",
	}
	if contactCheck.Pass {
		contactCheck.Message = fmt.Sprintf("%s status is ok", requiredChannel)
	} else {
		contactCheck.Message = fmt.Sprintf("%s status is '%s', expected ok", requiredChannel, requiredStatus)
	}
	checks = append(checks, contactCheck)

	// 6. no required-channel error (if required channel was run and errored)
	errorCheck := Check{
		Name:     "no_required_channel_error",
		Required: true,
		Pass:     requiredStatus != "error",
	}
	if errorCheck.Pass {
		errorCheck.Message = "no errors on required channels"
	} else {
		errorCheck.Message = fmt.Sprintf("%s last run reported error", requiredChannel)
	}
	checks = append(checks, errorCheck)

	// 4. company context
	hasCompanyInput := strings.TrimSpace(lead.Company) != "" || strings.TrimSpace(lead.Domain) != ""
	companyStatuses := []struct {
		key    string
		accept []string
	}{
		{"company_enrich", []string{"ok", "partial"}},
		{"extraction", []string{"ok"}},
		{"domain_intel", []string{"ok"}},
	}
	companyOk := false
	which := ""
	for _, cs := range companyStatuses {
		st := moduleStatus(lead.Results, cs.key)
		for _, a := range cs.accept {
			if st == a {
				companyOk = true
				which = fmt.Sprintf("%s %s", cs.key, st)
				break
			}
		}
		if companyOk {
			break
		}
	}
	companyCheck := Check{
		Name:     "company_context",
		Required: true,
		Pass:     hasCompanyInput && companyOk,
	}
	if companyCheck.Pass {
		companyCheck.Message = fmt.Sprintf("company context present (%s)", which)
	} else {
		if !hasCompanyInput {
			companyCheck.Message = "company or domain is required"
		} else if !companyOk {
			companyCheck.Message = "company_enrich ok/partial, extraction ok, or domain_intel ok is required"
		}
	}
	checks = append(checks, companyCheck)

	// 5. risk ceiling
	riskCheck := Check{
		Name:     "risk_ceiling",
		Required: true,
		Pass:     lead.RiskLevel != models.RiskHigh,
	}
	if riskCheck.Pass {
		riskCheck.Message = "risk_level is not high"
	} else {
		riskCheck.Message = "risk_level is high"
	}
	checks = append(checks, riskCheck)

	ready := true
	for _, c := range checks {
		if c.Required && !c.Pass {
			ready = false
			break
		}
	}

	warning := ""
	if ready && (lead.RiskLevel == models.RiskUnknown || strings.TrimSpace(lead.RiskLevel) == "") {
		warning = "risk_level is unknown; promotion is allowed but should be reviewed"
	}

	suggested := suggestedStage(lead.Stage, ready)

	return Report{
		Ready:          ready,
		Stage:          lead.Stage,
		SuggestedStage: suggested,
		Warning:        warning,
		Checks:         checks,
	}
}

// IsCrmReady is a convenience wrapper that returns true when the report says
// the lead can be promoted to crm_ready.
func IsCrmReady(lead models.Lead) bool {
	return Compute(lead).Ready
}

// CanDemoteTo reports whether the current stage can be demoted to target.
func CanDemoteTo(current, target string) bool {
	if !allowedDemoteTargets[target] {
		return false
	}
	curOrder, ok := stageOrder[current]
	if !ok {
		return false
	}
	targetOrder, ok := stageOrder[target]
	if !ok {
		return false
	}
	return targetOrder < curOrder
}

// moduleStatus extracts the status from a namespaced module result.
func moduleStatus(results map[string]any, key string) string {
	v, ok := results[key]
	if !ok {
		return "not_run"
	}
	m, ok := v.(map[string]any)
	if !ok {
		return "not_run"
	}
	s, _ := m["status"].(string)
	if s == "" {
		return "not_run"
	}
	return s
}

// suggestedStage returns the highest stage the lead should be in based on
// readiness. It never demotes from module-achieved stages; if ready it returns
// crm_ready, otherwise it returns the current stage (or the highest earlier
// stage when the current stage is crm_ready and readiness is lost).
func suggestedStage(current string, ready bool) string {
	if ready {
		return models.StageCrmReady
	}
	curOrder, ok := stageOrder[current]
	if !ok {
		return models.StageRaw
	}
	// If current is crm_ready but no longer ready, suggest validated.
	if curOrder > stageOrder[models.StageValidated] {
		return models.StageValidated
	}
	return current
}
