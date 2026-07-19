package risk

import (
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/models"
)

const (
	EmailValidateErrorPoints       = 40
	EmailValidateDisposablePoints  = 15
	EmailValidateRolePoints        = 15
	EmailValidateFreeMailPoints    = 15
	EmailValidateNotRunPoints      = 10
	EmailFlagMaxPoints             = 30

	PhoneValidateErrorPoints       = 35
	PhoneValidateInvalidPoints     = 25
	PhoneValidateNotRunPoints      = 10

	DomainIntelErrorPoints         = 20
	DomainIntelHardFailPoints      = 10

	SocialFootprintErrorPoints     = 10
	CompanyOrExtractionErrorPoints = 10

	ContactChecksGreenBonus        = -10
	CompanyEnrichOrExtractionOkBonus = -5

	ScoreLowMax    = 33
	ScoreMediumMax = 66
)

type Factor struct {
	Name    string `json:"name"`
	Points  int    `json:"points"`
	Message string `json:"message"`
}

type Report struct {
	Score   *float64 `json:"score"`
	Level   string   `json:"level"`
	Factors []Factor `json:"factors"`
}

func Compute(lead models.Lead) Report {
	score := 0
	factors := []Factor{}
	applicable := false

	if lead.Email != "" {
		applicable = true
		if email, ok := lead.Results["email_validate"].(map[string]any); ok {
			status, _ := email["status"].(string)
			if status == "error" {
				factors = append(factors, Factor{Name: "email_validate_error", Points: EmailValidateErrorPoints, Message: "email_validate status is error"})
				score += EmailValidateErrorPoints
			}
			if status != "error" {
				flagPoints := 0
				if b, ok := email["is_disposable"].(bool); ok && b {
					factors = append(factors, Factor{Name: "email_disposable", Points: EmailValidateDisposablePoints, Message: "disposable email domain"})
					flagPoints += EmailValidateDisposablePoints
				}
				if b, ok := email["is_role_account"].(bool); ok && b {
					factors = append(factors, Factor{Name: "email_role_account", Points: EmailValidateRolePoints, Message: "role account email"})
					flagPoints += EmailValidateRolePoints
				}
				if b, ok := email["is_free_provider"].(bool); ok && b {
					factors = append(factors, Factor{Name: "email_free_provider", Points: EmailValidateFreeMailPoints, Message: "free email provider"})
					flagPoints += EmailValidateFreeMailPoints
				}
				if flagPoints > EmailFlagMaxPoints {
					flagPoints = EmailFlagMaxPoints
				}
				score += flagPoints
			}
			if status != "ok" && status != "error" && status != "partial" {
				factors = append(factors, Factor{Name: "email_not_validated", Points: EmailValidateNotRunPoints, Message: "email present but email_validate not run"})
				score += EmailValidateNotRunPoints
			}
		} else {
			factors = append(factors, Factor{Name: "email_not_validated", Points: EmailValidateNotRunPoints, Message: "email present but email_validate not run"})
			score += EmailValidateNotRunPoints
		}
	}

	if lead.Phone != "" {
		applicable = true
		if phone, ok := lead.Results["phone_validate"].(map[string]any); ok {
			status, _ := phone["status"].(string)
			if status == "error" {
				factors = append(factors, Factor{Name: "phone_validate_error", Points: PhoneValidateErrorPoints, Message: "phone_validate status is error"})
				score += PhoneValidateErrorPoints
			}
			if status == "ok" {
				invalid := false
				if b, ok := phone["is_valid_number"].(bool); ok && !b {
					invalid = true
				}
				if b, ok := phone["is_possible"].(bool); ok && !b {
					invalid = true
				}
				if invalid {
					factors = append(factors, Factor{Name: "phone_invalid_or_impossible", Points: PhoneValidateInvalidPoints, Message: "phone parsed but marked invalid or impossible"})
					score += PhoneValidateInvalidPoints
				}
			}
			if status != "ok" && status != "error" && status != "partial" {
				factors = append(factors, Factor{Name: "phone_not_validated", Points: PhoneValidateNotRunPoints, Message: "phone present but phone_validate not run"})
				score += PhoneValidateNotRunPoints
			}
		} else {
			factors = append(factors, Factor{Name: "phone_not_validated", Points: PhoneValidateNotRunPoints, Message: "phone present but phone_validate not run"})
			score += PhoneValidateNotRunPoints
		}
	}

	if lead.Domain != "" || lead.URL != "" {
		applicable = true
		if di, ok := lead.Results["domain_intel"].(map[string]any); ok {
			status, _ := di["status"].(string)
			if status == "error" {
				factors = append(factors, Factor{Name: "domain_intel_error", Points: DomainIntelErrorPoints, Message: "domain_intel status is error"})
				score += DomainIntelErrorPoints
			}
			if status == "ok" || status == "partial" {
				hardFail := false
				if b, ok := di["resolvable"].(bool); ok && !b {
					hardFail = true
				}
				if wc, ok := di["web_check"].(map[string]any); ok {
					if b, ok := wc["resolvable"].(bool); ok && !b {
						hardFail = true
					}
					if ssl, ok := wc["ssl"].(map[string]any); ok {
						if b, ok := ssl["valid"].(bool); ok && !b {
							hardFail = true
						}
					}
					if http, ok := wc["http"].(map[string]any); ok {
						if code, ok := http["status_code"].(float64); ok && code >= 400 {
							hardFail = true
						}
					}
				}
				if hardFail {
					factors = append(factors, Factor{Name: "domain_intel_hard_fail", Points: DomainIntelHardFailPoints, Message: "domain resolution, SSL, or HTTP hard fail"})
					score += DomainIntelHardFailPoints
				}
			}
		}
	}

	if sf, ok := lead.Results["social_footprint"].(map[string]any); ok {
		applicable = true
		status, _ := sf["status"].(string)
		if status == "error" {
			factors = append(factors, Factor{Name: "social_footprint_error", Points: SocialFootprintErrorPoints, Message: "social_footprint status is error"})
			score += SocialFootprintErrorPoints
		}
	}

	if ce, ok := lead.Results["company_enrich"].(map[string]any); ok {
		applicable = true
		status, _ := ce["status"].(string)
		if status == "error" {
			factors = append(factors, Factor{Name: "company_enrich_error", Points: CompanyOrExtractionErrorPoints, Message: "company_enrich status is error"})
			score += CompanyOrExtractionErrorPoints
		}
	}

	if ex, ok := lead.Results["extraction"].(map[string]any); ok {
		applicable = true
		status, _ := ex["status"].(string)
		if status == "error" {
			factors = append(factors, Factor{Name: "extraction_error", Points: CompanyOrExtractionErrorPoints, Message: "extraction status is error"})
			score += CompanyOrExtractionErrorPoints
		}
	}

	if companyEnrichOk(lead.Results) || extractionOk(lead.Results) {
		factors = append(factors, Factor{Name: "company_context_ok", Points: CompanyEnrichOrExtractionOkBonus, Message: "company_enrich or extraction completed successfully"})
		score += CompanyEnrichOrExtractionOkBonus
	}

	if contactChecksGreen(lead) {
		factors = append(factors, Factor{Name: "contact_validated", Points: ContactChecksGreenBonus, Message: "required contact channel validated"})
		score += ContactChecksGreenBonus
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	level := models.RiskUnknown
	var scorePtr *float64
	if applicable || len(lead.Results) > 0 {
		s := float64(score)
		scorePtr = &s
		level = levelFromScore(score)
	} else {
		s := float64(0)
		scorePtr = &s
	}

	return Report{Score: scorePtr, Level: level, Factors: factors}
}

func levelFromScore(score int) string {
	if score <= ScoreLowMax {
		return models.RiskLow
	}
	if score <= ScoreMediumMax {
		return models.RiskMedium
	}
	return models.RiskHigh
}

func contactChecksGreen(lead models.Lead) bool {
	hasEmail := lead.Email != ""
	hasPhone := lead.Phone != ""
	if hasEmail {
		if email, ok := lead.Results["email_validate"].(map[string]any); ok {
			if status, _ := email["status"].(string); status == "ok" {
				return true
			}
		}
	}
	if !hasEmail && hasPhone {
		if phone, ok := lead.Results["phone_validate"].(map[string]any); ok {
			if status, _ := phone["status"].(string); status == "ok" {
				return true
			}
		}
	}
	return false
}

func companyEnrichOk(results map[string]any) bool {
	ce, ok := results["company_enrich"].(map[string]any)
	if !ok {
		return false
	}
	status, _ := ce["status"].(string)
	return status == "ok"
}

func extractionOk(results map[string]any) bool {
	ex, ok := results["extraction"].(map[string]any)
	if !ok {
		return false
	}
	status, _ := ex["status"].(string)
	return status == "ok"
}
