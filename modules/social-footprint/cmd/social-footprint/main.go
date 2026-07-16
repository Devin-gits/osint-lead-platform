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
//	social-footprint --username jane.smith --email jane.smith@acme.com
//
// Exit code is 0 whenever a well-formed lead was read and a record emitted —
// including a "skipped" result (no derivable handle) and per-handle "unknown"
// failures, which are reported in-band. A non-zero exit means the input itself
// could not be read or parsed.
package main

import (
	"encoding/json"
	"flag"
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
	if err := run(os.Stdin, os.Stdout, os.Stderr, os.Args[1:]...); err != nil {
		fmt.Fprintf(os.Stderr, "social-footprint: %v\n", err)
		os.Exit(1)
	}
}

func run(stdin io.Reader, stdout, stderr io.Writer, args ...string) error {
	username, email, timeoutFlag, err := parseFlags(args, stderr)
	if err != nil {
		return err
	}

	terminal := isTerminal(stdin)

	var raw []byte
	if !terminal {
		raw, err = io.ReadAll(stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
	}

	// Preserve the entire input record verbatim by decoding into a generic map,
	// so the module never drops or overwrites raw ingested fields.
	var lead map[string]interface{}
	if len(raw) == 0 {
		if terminal || (username != "" || email != "") {
			lead = map[string]interface{}{}
		} else {
			return fmt.Errorf("reading stdin: EOF")
		}
	} else {
		if err := json.Unmarshal(raw, &lead); err != nil {
			return fmt.Errorf("parsing lead JSON from stdin: %w", err)
		}
	}
	if lead == nil {
		lead = map[string]interface{}{}
	}

	// Flags augment/override the stdin lead record.
	if username != "" {
		lead["username"] = username
	}
	if email != "" {
		lead["email"] = email
	}

	// If stdin is a terminal and no flags were provided, there is nothing to
	// check; print usage and exit.
	if terminal && username == "" && email == "" {
		fmt.Fprintln(stderr, "Usage: social-footprint [--username HANDLE] [--email ADDRESS] [--timeout DURATION]")
		return fmt.Errorf("no input: provide --username, --email, or a lead record on stdin")
	}

	timeout := durationFromEnv(timeoutEnv)
	if timeoutFlag > 0 {
		timeout = timeoutFlag
	}

	validator := socialfootprint.NewValidatorWithBackend(timeout, durationFromEnv(minIntervalEnv), socialfootprint.BackendMaigret)
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

// parseFlags extracts the optional --username, --email, and --timeout flags from
// the supplied argument slice. It returns a zero timeout when the flag is not
// set, which tells run() to fall back to the env var / package default.
func parseFlags(args []string, stderr io.Writer) (username, email string, timeout time.Duration, err error) {
	fs := flag.NewFlagSet("social-footprint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&username, "username", "", "explicit handle to check (optional; takes priority over email-derived handles)")
	fs.StringVar(&email, "email", "", "lead email address (optional; overrides or supplies the email field in stdin JSON)")
	fs.DurationVar(&timeout, "timeout", 0, "per-handle subprocess timeout (optional; overrides SOCIAL_FOOTPRINT_TIMEOUT)")
	if err := fs.Parse(args); err != nil {
		return "", "", 0, err
	}
	return username, email, timeout, nil
}

// isTerminal reports whether r is an os.File backed by a character device (e.g.
// an interactive terminal). It is used to allow an empty/missing stdin when the
// binary is run directly by a user without a pipe.
func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
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
