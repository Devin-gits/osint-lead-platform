package companyenrich

import "context"

// Provider is the interface implemented by enrichment sources.
type Provider interface {
	// Name returns the provider identifier used in audit records.
	Name() string

	// Enrich attempts to enrich the company described by in, given the fields
	// already collected from previous providers (merged). It must never panic;
	// operational failures are reported via ProviderResult.Status/Error.
	Enrich(ctx context.Context, in Input, merged Fields) (ProviderResult, error)
}

// ProviderResult is the output of one provider.
type ProviderResult struct {
	Status     string
	SourceTool string
	Fields     Fields
	Error      string
}
