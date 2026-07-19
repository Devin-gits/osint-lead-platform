package companyenrich

import "context"

type fakeProvider struct {
	name   string
	enrich func(ctx context.Context, in Input, merged Fields) (ProviderResult, error)
}

func (f *fakeProvider) Name() string { return f.name }
func (f *fakeProvider) Enrich(ctx context.Context, in Input, merged Fields) (ProviderResult, error) {
	return f.enrich(ctx, in, merged)
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
