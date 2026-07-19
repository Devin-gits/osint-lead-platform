// Command extraction is the CLI front end for the extraction module. It reads
// one lead record as JSON on stdin, runs the configured extraction backend
// (Crawl4AI by default, or Firecrawl if EXTRACTION_BACKEND=firecrawl), and
// writes the augmented record as JSON on stdout. One structured audit line is
// written to stderr for every call.
//
// Usage:
//
//	echo '{"url":"https://example.com"}' | extraction
//	extraction --url https://example.com --timeout 30s --backend firecrawl
//
// Exit code is 0 whenever a well-formed lead was read and a record emitted —
// including extraction failures, which are reported in-band as
// extraction.status == "error" or "skipped". A non-zero exit means the input
// itself could not be read or parsed, or no URL was provided.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	extraction "github.com/Moyeil-73/osint-lead-platform/modules/extraction"
)

const resultKey = "extraction"

const (
	timeoutEnv     = "EXTRACTION_TIMEOUT"
	minIntervalEnv = "EXTRACTION_MIN_INTERVAL"
	backendEnv     = "EXTRACTION_BACKEND"
)

func main() {
	if err := run(os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "extraction: %v\n", err)
		os.Exit(1)
	}
}

func run(stdin io.Reader, stdout, stderr io.Writer) error {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	var lead map[string]interface{}
	if err := json.Unmarshal(raw, &lead); err != nil {
		return fmt.Errorf("parsing lead JSON from stdin: %w", err)
	}
	if lead == nil {
		lead = map[string]interface{}{}
	}

	url := flagString("url")
	if url != "" {
		lead["url"] = url
	}
	backend := flagString("backend")
	if backend != "" {
		lead["__backend_flag"] = backend
	}

	urlFromLead, _ := lead["url"].(string)
	if urlFromLead == "" {
		return fmt.Errorf("no url provided; supply --url or a 'url' field in stdin JSON")
	}

	// Allow --backend flag to override env for a single CLI invocation.
	if backend != "" {
		os.Setenv(backendEnv, backend)
	}

	extractor := extraction.NewExtractor(timeoutFromEnv(), minIntervalFromEnv(), backend)

	input := extraction.Input{
		URL:           urlFromLead,
		PermissionRef: stringField(lead, "permission_ref"),
		SourceID:      stringField(lead, "source_id"),
		Email:         stringField(lead, "email"),
		Name:          stringField(lead, "name"),
		Company:       stringField(lead, "company"),
		Domain:        stringField(lead, "domain"),
	}

	result, audit := extractor.Extract(context.Background(), input)

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

func stringField(lead map[string]interface{}, key string) string {
	if v, ok := lead[key].(string); ok {
		return v
	}
	return ""
}

func flagString(name string) string {
	for i, arg := range os.Args[1:] {
		if arg == "--"+name && i+1 < len(os.Args[1:]) {
			return os.Args[1:][i+1]
		}
		if prefix := "--" + name + "="; len(arg) > len(prefix) && arg[:len(prefix)] == prefix {
			return arg[len(prefix):]
		}
	}
	return ""
}

func timeoutFromEnv() time.Duration {
	if v := os.Getenv(timeoutEnv); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return extraction.DefaultTimeout
}

func minIntervalFromEnv() time.Duration {
	if v := os.Getenv(minIntervalEnv); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return extraction.DefaultMinInterval
}
