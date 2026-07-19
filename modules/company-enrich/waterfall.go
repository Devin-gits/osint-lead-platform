package companyenrich

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// runWaterfall executes providers in order, merges results with gap-fill only,
// and short-circuits once requiredFields are satisfied.
func runWaterfall(ctx context.Context, providers []Provider, in Input, requiredFields []string, clock func() time.Time, subject Subject, permissionRef string) (Fields, []AuditRecord, int64, string) {
	merged := emptyFields()
	var audits []AuditRecord
	var firstError string

	for _, p := range providers {
		pstart := clock()
		pres, err := p.Enrich(ctx, in, merged)
		dur := clock().Sub(pstart).Milliseconds()

		audit := newAudit(
			"company-enrich",
			p.Name(),
			permissionRef,
			subject,
			pres.Status,
			pres.Error,
			dur,
			pstart.UTC(),
		)
		audits = append(audits, audit)

		if err != nil {
			if firstError == "" {
				firstError = err.Error()
			}
			continue
		}

		mergeFields(&merged, pres.Fields)

		if fieldsSatisfied(merged, requiredFields) {
			break
		}
	}

	errMsg := ""
	if !fieldsSatisfied(merged, defaultP0()) && !hasUsefulData(merged) {
		if firstError != "" {
			errMsg = firstError
		} else {
			errMsg = "no useful enrichment data returned"
		}
	}

	return merged, audits, 0, errMsg
}

// mergeFields merges src into dst, filling only empty fields and appending sources.
func mergeFields(dst *Fields, src Fields) {
	if dst.Domain == "" && src.Domain != "" {
		dst.Domain = src.Domain
	}
	if dst.Name == "" && src.Name != "" {
		dst.Name = src.Name
	}
	if dst.LegalName == "" && src.LegalName != "" {
		dst.LegalName = src.LegalName
	}
	if dst.Website == "" && src.Website != "" {
		dst.Website = src.Website
	}
	if dst.Description == "" && src.Description != "" {
		dst.Description = src.Description
	}
	if dst.Founded == nil && src.Founded != nil {
		v := *src.Founded
		dst.Founded = &v
	}
	if dst.EmployeeCount == nil && src.EmployeeCount != nil {
		v := *src.EmployeeCount
		dst.EmployeeCount = &v
	}
	if dst.EmployeeCountRange == "" && src.EmployeeCountRange != "" {
		dst.EmployeeCountRange = src.EmployeeCountRange
	}
	if dst.Headquarters == nil && src.Headquarters != nil {
		h := *src.Headquarters
		dst.Headquarters = &h
	}

	dst.Industry = appendUnique(dst.Industry, src.Industry...)
	dst.TechStack = appendUnique(dst.TechStack, src.TechStack...)
	dst.Sources = appendUnique(dst.Sources, src.Sources...)

	if dst.SocialLinks == nil {
		dst.SocialLinks = map[string]string{}
	}
	for k, v := range src.SocialLinks {
		if _, ok := dst.SocialLinks[k]; !ok && v != "" {
			dst.SocialLinks[k] = v
		}
	}
}

// fieldsSatisfied reports whether all required fields have non-empty values.
func fieldsSatisfied(f Fields, required []string) bool {
	for _, r := range required {
		switch r {
		case "domain":
			if f.Domain == "" {
				return false
			}
		case "name":
			if f.Name == "" {
				return false
			}
		case "website":
			if f.Website == "" {
				return false
			}
		case "description":
			if f.Description == "" {
				return false
			}
		case "legal_name":
			if f.LegalName == "" {
				return false
			}
		case "founded":
			if f.Founded == nil {
				return false
			}
		case "employee_count":
			if f.EmployeeCount == nil {
				return false
			}
		case "employee_count_range":
			if f.EmployeeCountRange == "" {
				return false
			}
		case "headquarters":
			if f.Headquarters == nil {
				return false
			}
		case "industry":
			if len(f.Industry) == 0 {
				return false
			}
		case "tech_stack":
			if len(f.TechStack) == 0 {
				return false
			}
		case "social_links":
			if len(f.SocialLinks) == 0 {
				return false
			}
		}
	}
	return true
}

// hasUsefulData returns true if at least one enrichment field is present.
func hasUsefulData(f Fields) bool {
	return f.Domain != "" || f.Name != "" || f.Website != "" || f.Description != "" ||
		f.LegalName != "" || f.Founded != nil || f.EmployeeCount != nil ||
		f.EmployeeCountRange != "" || f.Headquarters != nil ||
		len(f.Industry) > 0 || len(f.TechStack) > 0 || len(f.SocialLinks) > 0
}

// confidenceFor returns a simple 0.0–1.0 score based on populated field categories.
func confidenceFor(f Fields) float64 {
	score := 0.0
	if f.Domain != "" {
		score += 0.1
	}
	if f.Name != "" {
		score += 0.2
	}
	if f.Website != "" {
		score += 0.1
	}
	if f.Description != "" {
		score += 0.15
	}
	if f.LegalName != "" {
		score += 0.1
	}
	if f.Founded != nil {
		score += 0.1
	}
	if f.EmployeeCount != nil || f.EmployeeCountRange != "" {
		score += 0.1
	}
	if f.Headquarters != nil && f.Headquarters.Country != "" {
		score += 0.1
	}
	if len(f.Industry) > 0 {
		score += 0.05
	}
	if len(f.TechStack) > 0 {
		score += 0.05
	}
	if score > 1.0 {
		score = 1.0
	}
	return score
}

func sourceToolFor(sources []string) string {
	if len(sources) == 0 {
		return "company-enrich"
	}
	return fmt.Sprintf("company-enrich/%s", strings.Join(sources, "+"))
}

func appendUnique(dst []string, src ...string) []string {
	seen := make(map[string]struct{}, len(dst))
	for _, d := range dst {
		seen[d] = struct{}{}
	}
	for _, s := range src {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; !ok {
			dst = append(dst, s)
			seen[s] = struct{}{}
		}
	}
	return dst
}
