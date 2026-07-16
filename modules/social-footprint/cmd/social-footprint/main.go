// Command social-footprint is the CLI front end for the social-footprint module.
// It reads one lead record as JSON on stdin, derives candidate handles from the
// lead's email (and optionally an enriched domain_intel.harvester sub-object),
// runs a rate-limited, scope-capped Maigret spot check per handle, and writes the
// augmented record — with a "social_footprint" key added — as JSON on stdout.
// One structured audit line per handle checked (or one for a skip) is written to
// stderr for every call, satisfying the audit requirement in
// docs/architecture.md.
//
// stdin/stdout are used (rather than an HTTP server) so the module composes
// cleanly in a shell/DAG pipeline as a single static binary with no daemon —
// matching the email-validate, domain-intel, and phone-validate reference
// modules.
//
// Usage:
//
//	echo '{"email":"jane.smith@acme.com"}' | social-footprint
//
// Exit code is 0 whenever a well-formed lead was read and a record emitted —
// including a "skipped" result (no derivable handle) and per-handle "unknown"
// failures, which are reported in-band. A non-zero exit means the input itself
// could not be read or parsed.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	socialfootprint "github.com/Moyeil-73/osint-lead-platform/modules/social-footprint"
)

// resultKey is the namespaced key added to the lead record. Kept in one place so
// downstream modules and tests agree on the contract.
const resultKey = "social_footprint"

const (
	timeoutEnv     = "SOCIAL_FOOTPRINT_TIMEOUT"      // Go duration, per-handle subprocess bound
	minIntervalEnv = "SOCIAL_FOOTPRINT_MIN_INTERVAL" // Go duration, min spacing between leads
)

func main() {
	if err := run(os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "social-footprint: %v\n", err)
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

	validator := socialfootprint.NewValidator(durationFromEnv(timeoutEnv), durationFromEnv(minIntervalEnv))
	result, audits := validator.Check(lead)

	// Audit first — one line per handle checked — so a call is logged even if
	// stdout encoding later fails.
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

// durationFromEnv reads a Go duration from the named env var; an unset or
// unparseable value returns 0, which the Validator maps to the package default.
func durationFromEnv(name string) time.Duration {
	if v := os.Getenv(name); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return 0
}
