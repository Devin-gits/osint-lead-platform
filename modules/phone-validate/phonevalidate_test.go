package phonevalidate

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestValidate_RealNumbers runs the real offline libphonenumber scanner (no
// mocks) against a clearly valid E.164 number and a clearly malformed one,
// asserting on the actual returned Result. No API key is set, so numverify is
// skipped cleanly and the module still returns full local-scanner results. The
// local scanner is fully offline, so this test needs no network.
func TestValidate_RealNumbers(t *testing.T) {
	t.Setenv(APIKeyEnv, "") // ensure numverify is skipped regardless of ambient env
	v := NewValidator(5 * time.Second)

	t.Run("clearly valid number", func(t *testing.T) {
		res, audits := v.Validate("+14152007986")

		if res.Status != "ok" {
			t.Fatalf("status = %q, want ok (local error: %q)", res.Status, res.Local.Error)
		}
		if !res.FormatValid {
			t.Errorf("format_valid = false, want true for +14152007986")
		}
		if !res.IsValidNumber {
			t.Errorf("is_valid_number = false, want true for +14152007986")
		}
		if res.Country != "US" {
			t.Errorf("country = %q, want US", res.Country)
		}
		if res.E164 != "+14152007986" {
			t.Errorf("e164 = %q, want +14152007986", res.E164)
		}
		if res.CountryCode != 1 {
			t.Errorf("country_code = %d, want 1", res.CountryCode)
		}
		if res.LineType == "" {
			t.Errorf("line_type is empty, want a classification")
		}
		// numverify must be skipped (no key), NOT unknown.
		if res.Numverify.Status != StatusSkipped {
			t.Errorf("numverify.status = %q, want %q", res.Numverify.Status, StatusSkipped)
		}
		if len(res.SourceTools) != 1 || res.SourceTools[0] != LocalTool {
			t.Errorf("source_tools = %v, want just [local] when numverify skipped", res.SourceTools)
		}
		assertAudits(t, audits, "ok", StatusSkipped)
	})

	t.Run("clearly malformed number", func(t *testing.T) {
		res, audits := v.Validate("not-a-phone")

		// The number cannot be parsed at all → local degrades to unknown.
		if res.Status != "unknown" {
			t.Fatalf("status = %q, want unknown for malformed input", res.Status)
		}
		if res.FormatValid || res.IsValidNumber {
			t.Errorf("format_valid/is_valid_number = %v/%v, want false/false", res.FormatValid, res.IsValidNumber)
		}
		if res.Country != "unknown" || res.LineType != "unknown" || res.Carrier != "unknown" {
			t.Errorf("country/line_type/carrier = %q/%q/%q, want all unknown", res.Country, res.LineType, res.Carrier)
		}
		if res.Local.Error == "" {
			t.Errorf("local.error is empty, want a parse-error note")
		}
		assertAudits(t, audits, "unknown", StatusSkipped)
	})
}

// TestValidate_InvalidButParseable covers a number that parses (well-formed
// length) but is not an assignable/valid number — the 555 fictional US range.
// format_valid should be true while is_valid_number is false.
func TestValidate_InvalidButParseable(t *testing.T) {
	t.Setenv(APIKeyEnv, "")
	v := NewValidator(0) // exercises the DefaultTimeout fallback
	res, _ := v.Validate("+1 555 444 1212")

	if res.Status != "ok" {
		t.Fatalf("status = %q, want ok (parse succeeded)", res.Status)
	}
	if !res.FormatValid {
		t.Errorf("format_valid = false, want true (plausible length)")
	}
	if res.IsValidNumber {
		t.Errorf("is_valid_number = true, want false for the 555 fictional range")
	}
}

// TestValidate_MissingPhone confirms the graceful-degradation contract: an empty
// phone must not error out — local is "unknown" with a note, numverify is
// "skipped", and audit lines are still emitted so the pipeline keeps running.
func TestValidate_MissingPhone(t *testing.T) {
	t.Setenv(APIKeyEnv, "")
	v := NewValidator(5 * time.Second)
	res, audits := v.Validate("   ")

	if res.Status != "unknown" {
		t.Errorf("status = %q, want unknown", res.Status)
	}
	if res.Local.Error == "" {
		t.Errorf("local.error is empty, want a missing-phone note")
	}
	if res.Numverify.Status != StatusSkipped {
		t.Errorf("numverify.status = %q, want %q", res.Numverify.Status, StatusSkipped)
	}
	assertAudits(t, audits, "unknown", StatusSkipped)
}

// TestNumverify_StubServer exercises the real numverify HTTP integration against
// a local stub (via NUMVERIFY_BASE_URL) returning canned numverify /validate
// JSON. This proves the request/parse/merge path works end-to-end without a real
// API key, and that numverify's carrier/line_type take precedence in the merged
// top-level verdict.
func TestNumverify_StubServer(t *testing.T) {
	const body = `{"valid":true,"number":"14152007986","international_format":"+14152007986",` +
		`"country_code":"US","country_name":"United States of America","location":"Novato",` +
		`"carrier":"AT&T Mobility LLC","line_type":"mobile"}`

	var gotNumber, gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotNumber = r.URL.Query().Get("number")
		gotKey = r.URL.Query().Get("access_key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	t.Setenv(APIKeyEnv, "test-key-123")
	t.Setenv(BaseURLEnv, srv.URL)

	v := NewValidator(5 * time.Second)
	res, audits := v.Validate("+14152007986")

	if gotKey != "test-key-123" {
		t.Errorf("stub received access_key = %q, want test-key-123", gotKey)
	}
	if gotNumber != "+14152007986" {
		t.Errorf("stub received number = %q, want +14152007986", gotNumber)
	}
	if res.Numverify.Status != StatusOK {
		t.Fatalf("numverify.status = %q, want ok (error: %q)", res.Numverify.Status, res.Numverify.Error)
	}
	if res.Numverify.Carrier != "AT&T Mobility LLC" {
		t.Errorf("numverify.carrier = %q, want AT&T Mobility LLC", res.Numverify.Carrier)
	}
	// numverify's live carrier/line_type must win in the merged verdict.
	if res.Carrier != "AT&T Mobility LLC" {
		t.Errorf("merged carrier = %q, want numverify's AT&T Mobility LLC", res.Carrier)
	}
	if res.LineType != "mobile" {
		t.Errorf("merged line_type = %q, want mobile", res.LineType)
	}
	if len(res.SourceTools) != 2 {
		t.Errorf("source_tools = %v, want both local and numverify", res.SourceTools)
	}
	assertAudits(t, audits, "ok", StatusOK)
}

// TestNumverify_APIErrorDegrades confirms that a numverify error envelope
// (success=false, HTTP 200) degrades numverify to "unknown" while the local
// scanner still returns a valid verdict — the pipeline is never blocked.
func TestNumverify_APIErrorDegrades(t *testing.T) {
	const errBody = `{"success":false,"error":{"code":101,"type":"invalid_access_key","info":"You have not supplied a valid API Access Key."}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(errBody))
	}))
	defer srv.Close()

	t.Setenv(APIKeyEnv, "bad-key")
	t.Setenv(BaseURLEnv, srv.URL)

	v := NewValidator(5 * time.Second)
	res, _ := v.Validate("+14152007986")

	if res.Status != "ok" {
		t.Fatalf("local status = %q, want ok (local must be unaffected by numverify failure)", res.Status)
	}
	if res.Numverify.Status != StatusUnknown {
		t.Errorf("numverify.status = %q, want unknown", res.Numverify.Status)
	}
	if res.Numverify.Error == "" {
		t.Errorf("numverify.error is empty, want the API error detail")
	}
	// Falls back to the local country since numverify failed.
	if res.Country != "US" {
		t.Errorf("country = %q, want US from local fallback", res.Country)
	}
}

func TestRedact(t *testing.T) {
	cases := map[string]string{
		"+14152007986":         "+14*******86",
		"14152007986":          "14*******86",
		"+1234":                "+****",
		"":                     "",
		"  +1 (415) 200-7986 ": "+14*******86",
	}
	for in, want := range cases {
		if got := redact(in); got != want {
			t.Errorf("redact(%q) = %q, want %q", in, got, want)
		}
	}
}

func assertAudits(t *testing.T, audits []AuditRecord, wantLocalStatus, wantNumverifyStatus string) {
	t.Helper()
	if len(audits) != 2 {
		t.Fatalf("expected 2 audit records (one per source), got %d", len(audits))
	}
	if audits[0].Tool != LocalTool {
		t.Errorf("audits[0].tool = %q, want %q", audits[0].Tool, LocalTool)
	}
	if audits[1].Tool != NumverifyTool {
		t.Errorf("audits[1].tool = %q, want %q", audits[1].Tool, NumverifyTool)
	}
	if audits[0].Status != wantLocalStatus {
		t.Errorf("local audit.status = %q, want %q", audits[0].Status, wantLocalStatus)
	}
	if audits[1].Status != wantNumverifyStatus {
		t.Errorf("numverify audit.status = %q, want %q", audits[1].Status, wantNumverifyStatus)
	}
	for _, a := range audits {
		if a.LegalBasis != LegalBasis {
			t.Errorf("audit.legal_basis = %q, want %q", a.LegalBasis, LegalBasis)
		}
		if _, err := time.Parse(time.RFC3339, a.CheckedAt); err != nil {
			t.Errorf("audit.checked_at = %q, not RFC3339: %v", a.CheckedAt, err)
		}
	}
}

func TestNormalizePhone(t *testing.T) {
	cases := map[string]string{
		"+14152007986":         "+14152007986",
		"  +1 (415) 200-7986 ": "+14152007986",
		"14152007986":          "+14152007986",
		"+44 20 7946 0958":     "+442079460958",
		"not-a-phone":          "",
		"":                     "",
	}
	for in, want := range cases {
		if got := normalizePhone(in); got != want {
			t.Errorf("normalizePhone(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRiskFlags(t *testing.T) {
	t.Setenv(APIKeyEnv, "")
	v := NewValidator(5 * time.Second)

	t.Run("invalid number", func(t *testing.T) {
		res, _ := v.Validate("+1 555 444 1212")
		if !contains(res.RiskFlags, "invalid_number") {
			t.Errorf("risk_flags = %v, want invalid_number", res.RiskFlags)
		}
		if !contains(res.RiskFlags, "carrier_unknown") {
			t.Errorf("risk_flags = %v, want carrier_unknown", res.RiskFlags)
		}
	})

	t.Run("toll-free line type", func(t *testing.T) {
		res, _ := v.Validate("+18005551234")
		if res.LineType != "toll_free" {
			t.Errorf("line_type = %q, want toll_free", res.LineType)
		}
		if !contains(res.RiskFlags, "toll_free") {
			t.Errorf("risk_flags = %v, want toll_free", res.RiskFlags)
		}
	})

	t.Run("premium-rate line type", func(t *testing.T) {
		res, _ := v.Validate("+19005550199")
		if res.LineType != "premium_rate" {
			t.Errorf("line_type = %q, want premium_rate", res.LineType)
		}
		if !contains(res.RiskFlags, "premium_rate") {
			t.Errorf("risk_flags = %v, want premium_rate", res.RiskFlags)
		}
	})

	t.Run("numverify line type wins", func(t *testing.T) {
		body := `{"valid":true,"number":"14152007986","country_code":"US","carrier":"AT&T Mobility LLC","line_type":"voip"}`
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		}))
		defer srv.Close()

		t.Setenv(APIKeyEnv, "test-key")
		t.Setenv(BaseURLEnv, srv.URL)

		v2 := NewValidator(5 * time.Second)
		res, _ := v2.Validate("+14152007986")
		if res.LineType != "voip" {
			t.Errorf("line_type = %q, want voip", res.LineType)
		}
		if !contains(res.RiskFlags, "voip") {
			t.Errorf("risk_flags = %v, want voip", res.RiskFlags)
		}
	})
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}

// TestNumverify_ConfigFile verifies the optional NUMVERIFY_CONFIG JSON file is
// read and used when env vars are not set; env vars still take precedence.
func TestNumverify_ConfigFile(t *testing.T) {
	var gotKey string
	body := `{"valid":true,"number":"14152007986","country_code":"US","carrier":"Test Carrier","line_type":"mobile"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.URL.Query().Get("access_key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "numverify.json")
	cfg := []byte(`{"api_key":"from-config","base_url":"` + srv.URL + `"}`)
	if err := os.WriteFile(cfgPath, cfg, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv(APIKeyEnv, "")
	t.Setenv(BaseURLEnv, "")
	t.Setenv("NUMVERIFY_CONFIG", cfgPath)

	v := NewValidator(5 * time.Second)
	res, _ := v.Validate("+14152007986")

	if gotKey != "from-config" {
		t.Errorf("stub received access_key = %q, want from-config", gotKey)
	}
	if res.Numverify.Status != StatusOK {
		t.Fatalf("numverify.status = %q, want ok", res.Numverify.Status)
	}
	if res.Carrier != "Test Carrier" {
		t.Errorf("merged carrier = %q, want Test Carrier", res.Carrier)
	}
}

// TestNumverify_LiveAPIGuarded makes a real numverify API call only when a key
// is configured and tests are not running in short mode. CI can run the full
// suite and hit the live endpoint; local short runs stay offline.
func TestNumverify_LiveAPIGuarded(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live numverify API call in short mode")
	}
	if os.Getenv(APIKeyEnv) == "" {
		t.Skip("NUMVERIFY_API_KEY not set; skipping live numverify test")
	}

	v := NewValidator(30 * time.Second)
	res, audits := v.Validate("+14152007986")

	if res.Status != "ok" {
		t.Fatalf("local status = %q, want ok", res.Status)
	}
	if res.Numverify.Status != StatusOK && res.Numverify.Status != StatusUnknown {
		t.Fatalf("numverify.status = %q, want ok or unknown", res.Numverify.Status)
	}
	if len(audits) != 2 {
		t.Fatalf("expected 2 audit records, got %d", len(audits))
	}
}
