package domainintel

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// HarvesterTool identifies the underlying engine and the version this module's
// output parsing was verified against. theHarvester is GPL-2.0; per the Stage 1
// decision and its evaluation we invoke it ONLY as an external CLI subprocess
// (mere aggregation) and NEVER import its Python modules, to keep this repo's
// MIT code clear of the copyleft boundary.
const HarvesterTool = "laramies/theHarvester@v4.11.1 (CLI subprocess)"

// HarvesterBinaryEnv overrides the theHarvester executable name/path.
const HarvesterBinaryEnv = "DOMAIN_INTEL_HARVESTER_BIN"

// defaultHarvesterBin is the executable looked up on PATH when the env override
// is unset.
const defaultHarvesterBin = "theHarvester"

// allowedSources is the keyless-capable, non-breach-database source allowlist
// passed to theHarvester's -b flag. Breach-database modules (haveibeenpwned,
// dehashed, leaklookup) are deliberately EXCLUDED per the Stage 1 decision's
// compliance note and docs/compliance.md; so are all paid/keyed sources
// (Hunter, SecurityTrails, Shodan, Censys) since keyless operation is the
// documented default. These four are keyless and return host/subdomain data.
var allowedSources = []string{"hackertarget", "crtsh", "rapiddns", "certspotter"}

// Host is one discovered subdomain and its resolved IP, parsed from
// theHarvester's "subdomain:ip" host strings.
type Host struct {
	Host string `json:"host"`
	IP   string `json:"ip,omitempty"`
}

// HarvesterResult is the "harvester" sub-result of the domain_intel key. It
// answers "what hosts/subdomains/contacts hang off this domain". Status is
// "unknown" if theHarvester was unavailable, timed out, or crashed.
type HarvesterResult struct {
	Status     string   `json:"status"` // "ok" if the subprocess ran and JSON parsed; else "unknown"
	Hosts      []Host   `json:"hosts"`
	HostCount  int      `json:"host_count"`
	IPs        []string `json:"ips"`
	Emails     []string `json:"emails"`
	Sources    []string `json:"sources"` // the allowlisted -b sources queried
	CheckedAt  string   `json:"checked_at"`
	SourceTool string   `json:"source_tool"`
	Error      string   `json:"error,omitempty"`
}

// harvesterJSON is the subset of theHarvester's -f JSON output this module
// consumes. theHarvester writes many optional keys (vhosts, asns, people,
// linkedin_*, takeover_results, shodan); we read only the domain-intel-relevant
// ones. Verified against v4.11.1 output (see __main__.py JSON report section).
type harvesterJSON struct {
	Hosts  []string `json:"hosts"`
	IPs    []string `json:"ips"`
	Emails []string `json:"emails"`
}

// runHarvester invokes theHarvester as a subprocess against domain, restricted
// to the keyless source allowlist, writes JSON via -f, and parses it. It never
// panics the pipeline: a missing binary, a non-zero exit, a timeout, or
// unparseable output all yield Status "unknown" with an Error note.
func runHarvester(ctx context.Context, domain string, timeout time.Duration) HarvesterResult {
	now := time.Now().UTC()
	sources := append([]string(nil), allowedSources...)
	res := HarvesterResult{
		Status:     "unknown",
		Hosts:      []Host{},
		IPs:        []string{},
		Emails:     []string{},
		Sources:    sources,
		CheckedAt:  now.Format(time.RFC3339),
		SourceTool: HarvesterTool,
	}

	domain = normalizeDomain(domain)
	if domain == "" {
		res.Error = "no domain field present on lead record"
		return res
	}

	bin := os.Getenv(HarvesterBinaryEnv)
	if bin == "" {
		bin = defaultHarvesterBin
	}
	if _, err := exec.LookPath(bin); err != nil {
		res.Error = fmt.Sprintf("theHarvester not found (%q); install it separately and/or set %s — see README", bin, HarvesterBinaryEnv)
		return res
	}

	tmpDir, err := os.MkdirTemp("", "domain-intel-harvester-")
	if err != nil {
		res.Error = fmt.Sprintf("could not create temp dir: %v", err)
		return res
	}
	defer os.RemoveAll(tmpDir)
	outBase := filepath.Join(tmpDir, "out")

	c, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// -d domain  -b <allowlist>  -f <base> (writes <base>.json + .xml)
	// The source allowlist is a fixed constant, never interpolated from the
	// lead record, so the -b argument cannot be influenced by input.
	args := []string{"-d", domain, "-b", strings.Join(sources, ","), "-f", outBase}
	cmd := exec.CommandContext(c, bin, args...)
	combined, runErr := cmd.CombinedOutput()

	if c.Err() == context.DeadlineExceeded {
		res.Error = fmt.Sprintf("theHarvester timed out after %s", timeout)
		return res
	}

	jsonPath := outBase + ".json"
	data, readErr := os.ReadFile(jsonPath)
	if readErr != nil {
		// No JSON file means the run produced nothing usable; surface the
		// subprocess error and a tail of its output for debugging.
		note := ""
		if runErr != nil {
			note = ": " + runErr.Error()
		}
		res.Error = fmt.Sprintf("theHarvester produced no JSON output%s (%s)", note, tailLines(string(combined), 3))
		return res
	}

	var parsed harvesterJSON
	if err := json.Unmarshal(data, &parsed); err != nil {
		res.Error = fmt.Sprintf("could not parse theHarvester JSON: %v", err)
		return res
	}

	for _, h := range parsed.Hosts {
		res.Hosts = append(res.Hosts, splitHost(h))
	}
	res.HostCount = len(res.Hosts)
	if parsed.IPs != nil {
		res.IPs = parsed.IPs
	}
	if parsed.Emails != nil {
		res.Emails = parsed.Emails
	}
	res.Status = "ok"
	return res
}

// splitHost parses theHarvester's "subdomain:ip" host string. It splits on the
// FIRST colon: a hostname never contains a colon, so everything after it is the
// IP — which for IPv6 hosts (e.g. "sub.example.org:2606:4700::1") itself
// contains colons and must be preserved whole. A host with no resolved IP (no
// colon) yields an empty IP field.
func splitHost(s string) Host {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, ":"); i >= 0 {
		return Host{Host: s[:i], IP: s[i+1:]}
	}
	return Host{Host: s}
}

// tailLines returns the last n non-empty lines of s, joined by " | ", for
// compact error context.
func tailLines(s string, n int) string {
	var lines []string
	for _, ln := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(ln); t != "" {
			lines = append(lines, t)
		}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, " | ")
}
