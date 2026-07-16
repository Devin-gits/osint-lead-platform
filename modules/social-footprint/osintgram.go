package socialfootprint

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Osintgram is GPL-3.0, so the Go code stays license-clean by invoking it through
// a small MIT wrapper that shells out to Osintgram's main.py as a subprocess CLI.
// No Osintgram source is imported, vendored, or linked.
const (
	osintgramPythonEnv  = "SOCIAL_FOOTPRINT_OSINTGRAM_PYTHON"
	osintgramWrapperEnv = "SOCIAL_FOOTPRINT_OSINTGRAM_WRAPPER"
	osintgramHomeEnv    = "SOCIAL_FOOTPRINT_OSINTGRAM_HOME"
)

// osintgramRunner runs a single scope-limited Osintgram check for one handle.
// It implements the maigretRunner interface so the existing Validator can use it
// without a parallel orchestration path.
type osintgramRunner struct{}

func (o *osintgramRunner) run(ctx context.Context, handle string, platforms []string, timeout time.Duration) (wrapperOutput, error) {
	python := os.Getenv(osintgramPythonEnv)
	if python == "" {
		python = os.Getenv(pythonEnv)
	}
	if python == "" {
		python = "python3"
	}
	if _, err := exec.LookPath(python); err != nil {
		return wrapperOutput{}, fmt.Errorf("python interpreter %q not found; install Python 3.10+ or set %s/%s — see README",
			python, osintgramPythonEnv, pythonEnv)
	}

	wrapper, err := locateOsintgramWrapper()
	if err != nil {
		return wrapperOutput{}, err
	}

	home := os.Getenv(osintgramHomeEnv)
	if home == "" {
		return wrapperOutput{}, fmt.Errorf("%s is not set; install Osintgram separately and point it at the checkout — see README", osintgramHomeEnv)
	}
	mainPy := filepath.Join(home, "main.py")
	if _, err := os.Stat(mainPy); err != nil {
		return wrapperOutput{}, fmt.Errorf("Osintgram main.py not found at %s=%q (%v)", osintgramHomeEnv, home, err)
	}

	args := []string{
		wrapper,
		"--handle", handle,
		"--command", "info",
	}
	if timeout > 0 {
		args = append(args, "--timeout", strconv.Itoa(int(timeout.Seconds())))
	}

	cmd := exec.CommandContext(ctx, python, args...)
	// main.py resolves config/credentials.ini relative to its working directory.
	cmd.Dir = home
	// Pass through the operator's environment so HIKERAPI_TOKEN and any
	// credentials.ini path config reach Osintgram. Secrets are never logged.
	cmd.Env = os.Environ()

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return wrapperOutput{}, fmt.Errorf("osintgram check timed out after %s", timeout)
	}

	// The wrapper prints a JSON object on stdout even on its own error paths, so
	// try to parse regardless of exit code before treating runErr as fatal.
	out, parseErr := parseWrapperOutput(stdout.String())
	if parseErr != nil {
		if runErr != nil {
			return wrapperOutput{}, fmt.Errorf("osintgram wrapper failed (%v); stderr: %s", runErr, tail(stderr.String(), 200))
		}
		return wrapperOutput{}, fmt.Errorf("could not parse osintgram wrapper output: %v; stdout: %s", parseErr, tail(stdout.String(), 200))
	}
	return out, nil
}

// locateOsintgramWrapper finds osintgram_check.py. Order: explicit env override,
// alongside the running binary (wrapper/osintgram_check.py and ./osintgram_check.py),
// then relative to this source file's directory (covers `go test`/`go run` where
// the binary lives in a temp dir).
func locateOsintgramWrapper() (string, error) {
	if p := os.Getenv(osintgramWrapperEnv); p != "" {
		if fileExists(p) {
			return p, nil
		}
		return "", fmt.Errorf("%s=%q does not exist", osintgramWrapperEnv, p)
	}

	var candidates []string
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "wrapper", "osintgram_check.py"),
			filepath.Join(dir, "osintgram_check.py"),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "wrapper", "osintgram_check.py"),
			filepath.Join(wd, "..", "wrapper", "osintgram_check.py"),
			filepath.Join(wd, "..", "..", "wrapper", "osintgram_check.py"),
		)
	}
	for _, c := range candidates {
		if fileExists(c) {
			return c, nil
		}
	}
	return "", fmt.Errorf("osintgram wrapper script not found; set %s to the path of wrapper/osintgram_check.py — see README", osintgramWrapperEnv)
}
