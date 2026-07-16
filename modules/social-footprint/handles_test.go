package socialfootprint

import (
	"reflect"
	"testing"
)

func handleStrings(cands []handleCandidate) []string {
	out := make([]string, len(cands))
	for i, c := range cands {
		out[i] = c.handle
	}
	return out
}

func handleOrigins(cands []handleCandidate) []string {
	out := make([]string, len(cands))
	for i, c := range cands {
		out[i] = c.origin
	}
	return out
}

func originOf(cands []handleCandidate, handle string) string {
	for _, c := range cands {
		if c.handle == handle {
			return c.origin
		}
	}
	return ""
}

func TestDeriveHandles_DirectUsername(t *testing.T) {
	cands := deriveHandles(map[string]interface{}{"username": "@jane.smith"})
	got := handleStrings(cands)
	want := []string{"jane.smith"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("derived %v, want %v", got, want)
	}
	if originOf(cands, "jane.smith") != originDirect {
		t.Errorf("origin for jane.smith = %q, want %q", originOf(cands, "jane.smith"), originDirect)
	}
}

func TestDeriveHandles_DirectUsernamePrecedesEmail(t *testing.T) {
	// An explicit username/handle should be checked first; email variants follow.
	cands := deriveHandles(map[string]interface{}{
		"username": "direct_user",
		"email":    "jane.smith@acme.com",
	})
	got := handleStrings(cands)
	if len(got) == 0 || got[0] != "direct_user" {
		t.Errorf("expected direct_user first, got %v", got)
	}
	if originOf(cands, "direct_user") != originDirect {
		t.Errorf("origin for direct_user = %q, want %q", originOf(cands, "direct_user"), originDirect)
	}
	// Email-derived handles should still appear after the direct one.
	if originOf(cands, "jane.smith") != originEmailLocal {
		t.Errorf("origin for jane.smith = %q, want %q", originOf(cands, "jane.smith"), originEmailLocal)
	}
}

func TestDeriveHandles_EmailPrimaryAndVariants(t *testing.T) {
	cands := deriveHandles(map[string]interface{}{"email": "jane.smith@acme.com"})
	got := handleStrings(cands)
	want := []string{"jane.smith", "janesmith", "jsmith"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("derived %v, want %v", got, want)
	}
	wantOrigins := []string{originEmailLocal, originEmailVariant, originEmailVariant}
	if gotOrigins := handleOrigins(cands); !reflect.DeepEqual(gotOrigins, wantOrigins) {
		t.Errorf("origins %v, want %v", gotOrigins, wantOrigins)
	}
}

func TestDeriveHandles_PlusTagStripped(t *testing.T) {
	cands := deriveHandles(map[string]interface{}{"email": "bob+newsletter@acme.com"})
	got := handleStrings(cands)
	if len(got) == 0 || got[0] != "bob" {
		t.Errorf("derived %v, want first = bob (plus-tag stripped)", got)
	}
	if originOf(cands, "bob") != originEmailLocal {
		t.Errorf("origin for bob = %q, want %q", originOf(cands, "bob"), originEmailLocal)
	}
}

func TestDeriveHandles_SimpleLocalNoVariants(t *testing.T) {
	cands := deriveHandles(map[string]interface{}{"email": "bob@acme.com"})
	got := handleStrings(cands)
	want := []string{"bob"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("derived %v, want %v (no variants for undotted local)", got, want)
	}
	if originOf(cands, "bob") != originEmailLocal {
		t.Errorf("origin for bob = %q, want %q", originOf(cands, "bob"), originEmailLocal)
	}
}

func TestDeriveHandles_NoEmailNoHandles(t *testing.T) {
	got := deriveHandles(map[string]interface{}{"name": "Nobody", "phone": "+1555"})
	if len(got) != 0 {
		t.Errorf("expected no handles, got %v", handleStrings(got))
	}
}

func TestDeriveHandles_SecondaryFromHarvester(t *testing.T) {
	lead := map[string]interface{}{
		"email": "jane@acme.com",
		"domain_intel": map[string]interface{}{
			"harvester": map[string]interface{}{
				"emails": []interface{}{"jsmith@acme.com"},
				"hosts": []interface{}{
					map[string]interface{}{"host": "www.acme.com"},     // infra -> skipped
					map[string]interface{}{"host": "careers.acme.com"}, // -> "careers"
				},
			},
		},
	}
	got := handleStrings(deriveHandles(lead))
	// primary email local "jane" first, then harvester email local "jsmith",
	// then host fragment "careers"; "www" excluded as infra.
	assertContains(t, got, "jane")
	assertContains(t, got, "jsmith")
	assertContains(t, got, "careers")
	for _, h := range got {
		if h == "www" {
			t.Errorf("infra label www should be excluded, got %v", got)
		}
	}
}

func TestDeriveHandles_Dedup(t *testing.T) {
	lead := map[string]interface{}{
		"email": "jane@acme.com",
		"domain_intel": map[string]interface{}{
			"harvester": map[string]interface{}{
				"emails": []interface{}{"jane@acme.com"}, // same local -> deduped
			},
		},
	}
	got := handleStrings(deriveHandles(lead))
	seen := map[string]int{}
	for _, h := range got {
		seen[h]++
		if seen[h] > 1 {
			t.Errorf("duplicate handle %q in %v", h, got)
		}
	}
}

func TestNormalizeHandle(t *testing.T) {
	cases := map[string]string{
		"Jane.Smith":                         "jane.smith",
		"  bob  ":                            "bob",
		"a":                                  "", // too short
		"1234":                               "", // no letter
		"j@ne!":                              "jne",
		"...x...":                            "", // trims to "x", too short
		"john_doe-99":                        "john_doe-99",
		"@jane.smith":                        "jane.smith",
		"https://github.com/jane.smith?tab=": "jane.smith",
		"http://www.x.com/jsmith":            "jsmith",
		"x.com/jsmith":                       "jsmith",
		"https://github.com/@jsmith":         "jsmith",
	}
	for in, want := range cases {
		if got := normalizeHandle(in); got != want {
			t.Errorf("normalizeHandle(%q) = %q, want %q", in, got, want)
		}
	}
}
