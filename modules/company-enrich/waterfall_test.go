package companyenrich

import (
	"context"
	"testing"
	"time"
)

func TestRunWaterfallFillsGaps(t *testing.T) {
	p1 := &fakeProvider{
		name: "p1",
		enrich: func(ctx context.Context, in Input, merged Fields) (ProviderResult, error) {
			return ProviderResult{
				Status: "partial",
				Fields: Fields{
					Domain:  in.Domain,
					Name:    in.Company,
					Website: "https://" + in.Domain,
					Sources: []string{"p1"},
				},
			}, nil
		},
	}
	p2 := &fakeProvider{
		name: "p2",
		enrich: func(ctx context.Context, in Input, merged Fields) (ProviderResult, error) {
			return ProviderResult{
				Status: "ok",
				Fields: Fields{
					Industry: []string{"Software"},
					Sources:  []string{"p2"},
				},
			}, nil
		},
	}

	merged, audits, _, errMsg := runWaterfall(
		context.Background(),
		[]Provider{p1, p2},
		Input{Domain: "example.com", Company: "Example", PermissionRef: "DEMO-1"},
		[]string{"domain", "name", "website", "industry"},
		time.Now,
		Subject{Domain: "example.com"},
		"DEMO-1",
	)
	if errMsg != "" {
		t.Errorf("unexpected errMsg: %s", errMsg)
	}
	if merged.Domain != "example.com" || merged.Name != "Example" {
		t.Errorf("merged P0 missing: %+v", merged)
	}
	if !contains(merged.Industry, "Software") {
		t.Errorf("industry not merged: %v", merged.Industry)
	}
	if len(audits) != 2 {
		t.Errorf("expected 2 audits, got %d", len(audits))
	}
}

func TestRunWaterfallEarlyStopOnP0(t *testing.T) {
	called := 0
	p1 := &fakeProvider{
		name: "p1",
		enrich: func(ctx context.Context, in Input, merged Fields) (ProviderResult, error) {
			return ProviderResult{
				Status: "ok",
				Fields: Fields{
					Domain:  in.Domain,
					Name:    in.Company,
					Website: "https://" + in.Domain,
					Sources: []string{"p1"},
				},
			}, nil
		},
	}
	p2 := &fakeProvider{
		name: "p2",
		enrich: func(ctx context.Context, in Input, merged Fields) (ProviderResult, error) {
			called++
			return ProviderResult{Status: "ok", Fields: Fields{Sources: []string{"p2"}}}, nil
		},
	}

	_, _, _, _ = runWaterfall(
		context.Background(),
		[]Provider{p1, p2},
		Input{Domain: "example.com", Company: "Example", PermissionRef: "DEMO-1"},
		defaultP0(),
		time.Now,
		Subject{Domain: "example.com"},
		"DEMO-1",
	)
	if called != 0 {
		t.Errorf("second provider called %d times, want 0", called)
	}
}

func TestRunWaterfallPropagatesError(t *testing.T) {
	p1 := &fakeProvider{
		name: "p1",
		enrich: func(ctx context.Context, in Input, merged Fields) (ProviderResult, error) {
			return ProviderResult{Status: "error", Error: "network failure"}, nil
		},
	}

	merged, _, _, errMsg := runWaterfall(
		context.Background(),
		[]Provider{p1},
		Input{Domain: "example.com", Company: "Example", PermissionRef: "DEMO-1"},
		defaultP0(),
		time.Now,
		Subject{Domain: "example.com"},
		"DEMO-1",
	)
	if errMsg == "" {
		t.Errorf("expected errMsg from failed provider")
	}
	if hasUsefulData(merged) {
		t.Errorf("expected no useful data, got %+v", merged)
	}
}
