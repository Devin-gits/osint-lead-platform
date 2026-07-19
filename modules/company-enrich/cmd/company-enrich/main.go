// Command company-enrich reads a lead record as JSON on stdin, enriches the
// company, and writes the augmented record as JSON on stdout. Audit records are
// emitted as JSON lines on stderr.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	companyenrich "github.com/Moyeil-73/osint-lead-platform/modules/company-enrich"
)

func main() {
	if err := run(os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, `{"error":"%s"}\n`, err.Error())
		os.Exit(1)
	}
}

func run(stdin io.Reader, stdout, stderr io.Writer) error {
	ctx := context.Background()

	// Read all stdin as JSON.
	data, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	var in companyenrich.Input
	if err := json.Unmarshal(data, &in); err != nil {
		return fmt.Errorf("invalid input JSON: %w", err)
	}

	enricher := companyenrich.NewEnricher(0, 0)
	res, audits := enricher.Enrich(ctx, in)

	// Write audit records to stderr first.
	for _, a := range audits {
		line, err := json.Marshal(a)
		if err != nil {
			return fmt.Errorf("audit marshal: %w", err)
		}
		fmt.Fprintln(stderr, string(line))
	}

	// Write result to stdout.
	out, err := companyenrich.ResultToJSON(res)
	if err != nil {
		return fmt.Errorf("result marshal: %w", err)
	}
	fmt.Fprintln(stdout, string(out))

	return nil
}
