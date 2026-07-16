// Command domain-intel is the CLI front end for the domain-intel module. It
// reads one lead record as JSON on stdin, adds a "domain_intel" key combining
// the web-check-lite and theHarvester sub-results, and writes the augmented
// record as JSON on stdout. One structured audit line per underlying tool is
// written to stderr for every call, satisfying the audit requirement in
// docs/architecture.md.
//
// stdin/stdout are used (rather than an HTTP server) so the module composes
// cleanly in a shell/DAG pipeline as a single static binary with no daemon —
// matching the email-validate reference module and the sequential
// module-chaining the pipeline is built from.
//
// Usage:
//
//	echo '{"domain":"owasp.org"}' | domain-intel
//
// Exit code is 0 whenever a well-formed lead was read and a record emitted —
// including sub-tool failures, which are reported in-band as
// domain_intel.<tool>.status == "unknown". A non-zero exit means the input
// itself could not be read or parsed.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	domainintel "github.com/Moyeil-73/osint-lead-platform/modules/domain-intel"
)

// resultKey is the namespaced key added to the lead record. Kept in one place
// so downstream modules and tests agree on the contract.
const resultKey = "domain_intel"

// timeoutEnv overrides the per-sub-tool timeout with a Go duration (e.g. "30s").
const timeoutEnv = "DOMAIN_INTEL_TIMEOUT"

func main() {
	if err := run(os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "domain-intel: %v\n", err)
		os.Exit(1)
	}
}

func run(stdin io.Reader, stdout, stderr io.Writer) error {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	// Preserve the entire input record verbatim by decoding into a generic map,
	// so the module never drops or overwrites raw ingested fields.
	var lead map[string]interface{}
	if err := json.Unmarshal(raw, &lead); err != nil {
		return fmt.Errorf("parsing lead JSON from stdin: %w", err)
	}
	if lead == nil {
		lead = map[string]interface{}{}
	}

	domain, _ := lead["domain"].(string)

	analyzer := domainintel.NewAnalyzer(timeoutFromEnv())
	result, audits := analyzer.Analyze(domain)

	// Audit first — one line per tool — so a call is logged even if stdout
	// encoding later fails.
	for _, a := range audits {
		if line, err := json.Marshal(a); err == nil {
			fmt.Fprintln(stderr, string(line))
		}
	}

	lead[resultKey] = result

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(lead); err != nil {
		return fmt.Errorf("encoding result to stdout: %w", err)
	}
	return nil
}

// timeoutFromEnv reads DOMAIN_INTEL_TIMEOUT (a Go duration like "30s"); an unset
// or unparseable value falls back to the package default.
func timeoutFromEnv() time.Duration {
	if v := os.Getenv(timeoutEnv); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return domainintel.DefaultTimeout
}
