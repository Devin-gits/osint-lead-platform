// Command email-validate is the CLI front end for the email-validate module.
// It reads one lead record as JSON on stdin, adds an "email_validate" key with
// the validation result, and writes the augmented record as JSON on stdout.
// A structured audit line (one JSON object) is written to stderr for every
// call, satisfying the audit requirement in docs/architecture.md.
//
// stdin/stdout are used (rather than an HTTP server) so the module composes
// cleanly in a shell/DAG pipeline as a single static binary with no daemon to
// run, matching the sequential module-chaining the pipeline is built from.
//
// Usage:
//
//	echo '{"email":"support@github.com"}' | email-validate
//
// Exit code is 0 whenever a well-formed lead was read and a record emitted —
// including validation failures, which are reported in-band as
// email_validate.status == "unknown". A non-zero exit means the input itself
// could not be read or parsed.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	emailvalidate "github.com/Moyeil-73/osint-lead-platform/modules/email-validate"
)

// resultKey is the namespaced key added to the lead record. Kept in one place
// so downstream modules and tests agree on the contract.
const resultKey = "email_validate"

func main() {
	if err := run(os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "email-validate: %v\n", err)
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

	email, _ := lead["email"].(string)

	val := emailvalidate.NewValidator(timeoutFromEnv())
	result, audit := val.Validate(email)

	// Audit first, so a call is logged even if stdout encoding later fails.
	if line, err := json.Marshal(audit); err == nil {
		fmt.Fprintln(stderr, string(line))
	}

	lead[resultKey] = result

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(lead); err != nil {
		return fmt.Errorf("encoding result to stdout: %w", err)
	}
	return nil
}

// timeoutFromEnv reads EMAIL_VALIDATE_TIMEOUT (a Go duration like "5s"); an
// unset or unparseable value falls back to the package default.
func timeoutFromEnv() time.Duration {
	if v := os.Getenv("EMAIL_VALIDATE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return emailvalidate.DefaultTimeout
}
