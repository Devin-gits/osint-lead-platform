package extraction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Crawl4AI is Apache-2.0 with an attribution clause. The Go code stays MIT-clean
// by invoking it through wrapper/crawl4ai_extract.py as a subprocess CLI.
const (
	crawl4aiPythonEnv  = "EXTRACTION_CRAWL4AI_PYTHON"
	crawl4aiWrapperEnv = "EXTRACTION_CRAWL4AI_WRAPPER"
)

type crawl4aiRunner struct {
	sourceToolName string
}

func newCrawl4AIRunner() *crawl4aiRunner {
	return &crawl4aiRunner{sourceToolName: SourceToolCrawl4AI}
}

func (c *crawl4aiRunner) sourceTool() string {
	return c.sourceToolName
}

func (c *crawl4aiRunner) run(ctx context.Context, url string, timeout time.Duration) (Result, error) {
	python := os.Getenv(crawl4aiPythonEnv)
	if python == "" {
		python = "python3"
	}
	if _, err := exec.LookPath(python); err != nil {
		return Result{
			Status: "error",
			URL:    url,
			Error:  fmt.Sprintf("python interpreter %q not found; install Python 3.10+ and Crawl4AI — see README", python),
		}, nil
	}

	wrapper, err := locateCrawl4AIWrapper()
	if err != nil {
		return Result{
			Status: "error",
			URL:    url,
			Error:  err.Error(),
		}, nil
	}

	args := []string{
		wrapper,
		"--url", url,
	}
	if timeout > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", int(timeout.Seconds())))
	}

	cmd := exec.CommandContext(ctx, python, args...)
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return Result{
			Status: "error",
			URL:    url,
			Error:  fmt.Sprintf("crawl4ai subprocess timed out after %s", timeout),
		}, nil
	}

	// The wrapper prints a JSON object on stdout even on its own error paths, so
	// try to parse regardless of exit code before treating runErr as fatal.
	res, parseErr := parseCrawl4AIOutput(stdout.String())
	if parseErr != nil {
		note := ""
		if runErr != nil {
			note = "; stderr: " + tail(stderr.String(), 200)
		}
		return Result{
			Status: "error",
			URL:    url,
			Error:  fmt.Sprintf("could not parse crawl4ai wrapper output: %v%s", parseErr, note),
		}, nil
	}
	return res, nil
}

// locateCrawl4AIWrapper finds wrapper/crawl4ai_extract.py. Order: explicit env
// override, alongside the running binary, then relative to current working dir
// (covers go test / go run).
func locateCrawl4AIWrapper() (string, error) {
	if p := os.Getenv(crawl4aiWrapperEnv); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		return "", fmt.Errorf("%s=%q does not exist", crawl4aiWrapperEnv, p)
	}

	var candidates []string
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			// Binaries built inside modules/extraction (e.g., bin/extraction)
			// live one directory below the module root.
			filepath.Join(dir, "..", "wrapper", "crawl4ai_extract.py"),
			filepath.Join(dir, "wrapper", "crawl4ai_extract.py"),
			filepath.Join(dir, "crawl4ai_extract.py"),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "wrapper", "crawl4ai_extract.py"),
			filepath.Join(wd, "..", "wrapper", "crawl4ai_extract.py"),
			filepath.Join(wd, "..", "..", "wrapper", "crawl4ai_extract.py"),
			// Monorepo layout: module may be reached from services/control-plane,
			// repository root, or another sibling directory.
			filepath.Join(wd, "..", "..", "modules", "extraction", "wrapper", "crawl4ai_extract.py"),
			filepath.Join(wd, "..", "modules", "extraction", "wrapper", "crawl4ai_extract.py"),
			filepath.Join(wd, "modules", "extraction", "wrapper", "crawl4ai_extract.py"),
		)
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("crawl4ai wrapper script not found; set %s to wrapper/crawl4ai_extract.py — see README", crawl4aiWrapperEnv)
}

func parseCrawl4AIOutput(s string) (Result, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Result{}, fmt.Errorf("empty wrapper output")
	}
	var r Result
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		return Result{}, err
	}
	return r, nil
}

func tail(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return "..." + s[len(s)-max:]
}
