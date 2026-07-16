package domainintel

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestAnalyze_RealDomain runs both real sub-tools (no mocks) against a
// well-known, long-established domain and asserts on the actual combined
// result. web-check-lite performs live DNS/SSL/WHOIS lookups; theHarvester is
// invoked as a real subprocess if present on PATH. The test therefore requires
// outbound network. If theHarvester is not installed, the harvester sub-result
// must still degrade to "unknown" (not fail the test) — the whole point of the
// graceful-degradation contract.
func TestAnalyze_RealDomain(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network + subprocess test in -short mode")
	}
	a := NewAnalyzer(45 * time.Second)
	res, audits := a.Analyze("owasp.org")

	// web-check must resolve a long-lived domain.
	if res.WebCheck.Status != "ok" {
		t.Fatalf("web_check.status = %q, want ok (error: %q)", res.WebCheck.Status, res.WebCheck.Error)
	}
	if !res.WebCheck.Resolvable || res.WebCheck.DNS == nil || len(res.WebCheck.DNS.A) == 0 {
		t.Errorf("web_check: expected A records for owasp.org, got %+v", res.WebCheck.DNS)
	}
	if res.WebCheck.SSL == nil || !res.WebCheck.SSL.Valid {
		t.Errorf("web_check.ssl: expected a valid cert for owasp.org, got %+v", res.WebCheck.SSL)
	}

	// harvester: if the binary exists it should return hosts; if not, it must
	// degrade to unknown with an explanatory error — never crash.
	if _, err := lookupHarvesterBin(); err == nil {
		if res.Harvester.Status != "ok" {
			t.Errorf("harvester.status = %q, want ok when theHarvester is installed (error: %q)", res.Harvester.Status, res.Harvester.Error)
		}
		if res.Harvester.HostCount == 0 {
			t.Errorf("harvester: expected some hosts for owasp.org, got 0")
		}
	} else {
		if res.Harvester.Status != "unknown" {
			t.Errorf("harvester.status = %q, want unknown when theHarvester absent", res.Harvester.Status)
		}
		if res.Harvester.Error == "" {
			t.Errorf("harvester: expected an error note when theHarvester absent")
		}
	}

	// Exactly one audit line per tool, both tagged with the legal basis.
	if len(audits) != 2 {
		t.Fatalf("expected 2 audit records (one per tool), got %d", len(audits))
	}
	for _, au := range audits {
		if au.LegalBasis != LegalBasis {
			t.Errorf("audit.legal_basis = %q, want %q", au.LegalBasis, LegalBasis)
		}
		if au.Domain != "owasp.org" {
			t.Errorf("audit.domain = %q, want owasp.org", au.Domain)
		}
		if _, err := time.Parse(time.RFC3339, au.CheckedAt); err != nil {
			t.Errorf("audit.checked_at = %q not RFC3339: %v", au.CheckedAt, err)
		}
	}
	if audits[0].Tool != WebCheckTool || audits[1].Tool != HarvesterTool {
		t.Errorf("audit tools = [%q,%q], want [%q,%q]", audits[0].Tool, audits[1].Tool, WebCheckTool, HarvesterTool)
	}
}

// TestAnalyze_MissingDomain confirms graceful degradation: an empty domain must
// not error out — both sub-tools return "unknown" with an error note and audit
// lines are still emitted, so the pipeline keeps running.
func TestAnalyze_MissingDomain(t *testing.T) {
	a := NewAnalyzer(0) // exercises DefaultTimeout fallback
	res, audits := a.Analyze("   ")

	if res.WebCheck.Status != "unknown" || res.WebCheck.Error == "" {
		t.Errorf("web_check: want status unknown with error, got %+v", res.WebCheck)
	}
	if res.Harvester.Status != "unknown" || res.Harvester.Error == "" {
		t.Errorf("harvester: want status unknown with error, got %+v", res.Harvester)
	}
	if len(audits) != 2 {
		t.Fatalf("expected 2 audit records, got %d", len(audits))
	}
	for _, au := range audits {
		if au.Status != "unknown" {
			t.Errorf("audit.status = %q, want unknown", au.Status)
		}
		if au.Error == "" {
			t.Errorf("audit.error is empty for failed tool %q", au.Tool)
		}
	}
}

// TestHarvesterAbsent forces the theHarvester-not-found path via the binary
// override and asserts a clean degrade to "unknown" (no network needed).
func TestHarvesterAbsent(t *testing.T) {
	t.Setenv(HarvesterBinaryEnv, "definitely-not-a-real-binary-xyz")
	res := runHarvester(context.Background(), "example.com", 5*time.Second)
	if res.Status != "unknown" {
		t.Errorf("status = %q, want unknown", res.Status)
	}
	if res.Error == "" {
		t.Errorf("expected an error note about the missing binary")
	}
	if res.SourceTool != HarvesterTool {
		t.Errorf("source_tool = %q, want %q", res.SourceTool, HarvesterTool)
	}
}

func TestNormalizeDomain(t *testing.T) {
	cases := map[string]string{
		"owasp.org":                        "owasp.org",
		"  OWASP.ORG  ":                    "owasp.org",
		"https://owasp.org/path?x=1":       "owasp.org",
		"http://user@sub.owasp.org:8443/p": "sub.owasp.org",
		"owasp.org.":                       "owasp.org",
		"":                                 "",
	}
	for in, want := range cases {
		if got := normalizeDomain(in); got != want {
			t.Errorf("normalizeDomain(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSplitHost(t *testing.T) {
	if h := splitHost("mail.owasp.org:104.20.44.163"); h.Host != "mail.owasp.org" || h.IP != "104.20.44.163" {
		t.Errorf("splitHost with ip = %+v", h)
	}
	if h := splitHost("lonelyhost.owasp.org"); h.Host != "lonelyhost.owasp.org" || h.IP != "" {
		t.Errorf("splitHost without ip = %+v", h)
	}
	// IPv6 addresses contain colons; the host must split on the FIRST colon so
	// the whole address is preserved as the IP.
	if h := splitHost("wiki.owasp.org:2606:4700:10::6814:2ca3"); h.Host != "wiki.owasp.org" || h.IP != "2606:4700:10::6814:2ca3" {
		t.Errorf("splitHost with ipv6 = %+v", h)
	}
}

func TestAllowlistExcludesBreachDBs(t *testing.T) {
	for _, s := range filteredHarvesterSources() {
		if _, blocked := blockedSources[s]; blocked {
			t.Errorf("filtered sources must not contain breach-database source %q", s)
		}
	}
	if len(filteredHarvesterSources()) == 0 {
		t.Error("filtered sources are empty; theHarvester needs at least one -b source")
	}
}

func TestHarvesterArgvExcludesBlockedSources(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "args")
	fakeBin, err := filepath.Abs(filepath.Join("testdata", "fake-theHarvester"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(HarvesterBinaryEnv, fakeBin)
	t.Setenv("FAKE_HARVESTER_ARGS_FILE", argsFile)

	originalSources := allowedSources
	allowedSources = append(append([]string(nil), originalSources...), "dehashed", "leaklookup", "haveibeenpwned")
	t.Cleanup(func() { allowedSources = originalSources })

	res := runHarvester(context.Background(), "example.com", 5*time.Second)
	if res.Status != "ok" {
		t.Fatalf("status = %q, want ok: %s", res.Status, res.Error)
	}
	if res.HostCount != 2 || len(res.IPs) != 2 || len(res.Emails) != 1 {
		t.Fatalf("unexpected parsed result: %+v", res)
	}

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	args := string(data)
	for source := range blockedSources {
		if strings.Contains(args, source) {
			t.Errorf("blocked source %q appeared in argv: %s", source, args)
		}
	}
	if !strings.Contains(args, "-l "+strconv.Itoa(harvesterLimit)) {
		t.Errorf("argv missing fixed limit: %s", args)
	}
	if !strings.Contains(args, "-f ") {
		t.Errorf("argv missing output basename: %s", args)
	}
}

func TestResultJSONSchema(t *testing.T) {
	result := Result{
		WebCheck: WebCheckResult{
			DNS:  &DNSResult{A: []string{"192.0.2.1"}, CNAME: []string{"edge.example.net"}},
			SSL:  &SSLResult{Valid: true, Protocol: "TLS 1.3", SANs: []string{"example.com"}},
			HTTP: &HTTPResult{StatusCode: 200, Server: "example", Headers: map[string]string{"Content-Type": "text/html"}},
		},
		Harvester: HarvesterResult{Hosts: []Host{}, IPs: []string{}, Emails: []string{}},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"web_check"`, `"harvester"`, `"cname"`, `"protocol"`, `"sans"`, `"http"`, `"status_code"`} {
		if !strings.Contains(string(data), key) {
			t.Errorf("JSON missing stable key %s: %s", key, data)
		}
	}
}

func TestInspectHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Server", "test-server")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Powered-By", "test")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	domain := strings.TrimPrefix(server.URL, "http://")
	result := inspectHTTP(context.Background(), domain, 2*time.Second)
	if result.Error != "" {
		t.Fatalf("inspectHTTP returned error: %s", result.Error)
	}
	if result.StatusCode != http.StatusAccepted || result.Server != "test-server" {
		t.Errorf("unexpected HTTP result: %+v", result)
	}
	if result.Headers["Content-Type"] != "application/json" || result.Headers["X-Powered-By"] != "test" {
		t.Errorf("unexpected selected headers: %+v", result.Headers)
	}
}

func TestTLSVersionName(t *testing.T) {
	cases := map[uint16]string{
		0:      "unknown",
		0x0303: "TLS 1.2",
		0x0304: "TLS 1.3",
	}
	for version, want := range cases {
		if got := tlsVersionName(version); got != want {
			t.Errorf("tlsVersionName(%#x) = %q, want %q", version, got, want)
		}
	}
}

// lookupHarvesterBin resolves the theHarvester binary the same way runHarvester
// does, for the conditional assertion in TestAnalyze_RealDomain.
func lookupHarvesterBin() (string, error) {
	bin := os.Getenv(HarvesterBinaryEnv)
	if bin == "" {
		bin = defaultHarvesterBin
	}
	return exec.LookPath(bin)
}
