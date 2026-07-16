package socialfootprint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// curatedPlatforms is the scope-discipline allow-list: the ONLY sites Maigret is
// allowed to check, passed explicitly to the wrapper on every call. Maigret's
// default fans out to hundreds/thousands of sites (evaluations/maigret.md §6),
// which is exactly the "bulk non-consensual scraping" pattern docs/compliance.md
// restricts. We instead check a fixed, curated set of ~15 major platforms where
// a claimed/unclaimed signal is a meaningful "is this a real, active person?"
// indicator and whose scale is defensible for a per-lead spot check. Changing
// scope means editing this list in code and re-review — it cannot be widened at
// runtime.
var curatedPlatforms = []string{
	"GitHub", "GitLab", "Reddit", "Twitter", "Instagram",
	"Pinterest", "Medium", "Telegram", "Keybase", "HackerNews",
	"Steam", "SoundCloud", "Vimeo", "About.me", "Patreon",
}

// perSiteTimeoutSeconds is the per-site request timeout handed to the wrapper
// (Maigret's own --timeout). It is smaller than the whole-run Go timeout, which
// bounds the subprocess overall.
const perSiteTimeoutSeconds = 12

// Env overrides for locating the Python interpreter and the wrapper script.
const (
	pythonEnv  = "SOCIAL_FOOTPRINT_PYTHON"
	wrapperEnv = "SOCIAL_FOOTPRINT_WRAPPER"
)

// platformResult mirrors one entry of the wrapper's JSON "results" array.
type platformResult struct {
	Platform   string            `json:"platform"`
	Status     string            `json:"status"`
	URL        string            `json:"url"`
	HTTPStatus int               `json:"http_status"`
	Instagram  *InstagramDetails `json:"instagram,omitempty"`
}

// wrapperOutput is the JSON contract the Python wrapper prints on stdout.
type wrapperOutput struct {
	Tool           string           `json:"tool"`
	Version        string           `json:"version"`
	Username       string           `json:"username"`
	SitesRequested []string         `json:"sites_requested"`
	Results        []platformResult `json:"results"`
	CheckedAt      string           `json:"checked_at"`
	Error          string           `json:"error"`
}

// maigretRunner is the shared backend-runner interface for this module. Despite
// the name (it was introduced for Maigret), it is implemented by Maigret,
// Sherlock, and Osintgram runners so the Validator can swap backends without a
// parallel orchestration path.
type maigretRunner interface {
	run(ctx context.Context, handle string, platforms []string, timeout time.Duration) (wrapperOutput, error)
}

// subprocessRunner is the production maigretRunner: it invokes the Python wrapper
// (which embeds Maigret as a library) as a subprocess and parses its JSON.
type subprocessRunner struct{}

func (s *subprocessRunner) run(ctx context.Context, handle string, platforms []string, timeout time.Duration) (wrapperOutput, error) {
	python := os.Getenv(pythonEnv)
	if python == "" {
		python = "python3"
	}
	if _, err := exec.LookPath(python); err != nil {
		return wrapperOutput{}, fmt.Errorf("python interpreter %q not found; install Python 3.10+ or set %s — see README", python, pythonEnv)
	}

	wrapper, err := locateWrapperFile(wrapperEnv, "maigret_check.py")
	if err != nil {
		return wrapperOutput{}, err
	}

	// The per-site timeout is bounded so the wrapper cannot outlive the Go
	// context deadline; leave headroom for process start + DB load.
	siteTimeout := perSiteTimeoutSeconds
	if secs := int(timeout.Seconds()) - 5; secs > 0 && secs < siteTimeout {
		siteTimeout = secs
	}

	// All arguments are passed as an argv slice (no shell), and the platform
	// list is a fixed in-code constant — never interpolated from lead input —
	// so neither the handle nor the site list can inject a shell command.
	args := []string{
		wrapper,
		"--username", handle,
		"--sites", strings.Join(platforms, ","),
		"--timeout", strconv.Itoa(siteTimeout),
		"--max-sites", strconv.Itoa(len(platforms)),
	}
	cmd := exec.CommandContext(ctx, python, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return wrapperOutput{}, fmt.Errorf("maigret check timed out after %s", timeout)
	}

	// The wrapper prints a JSON object on stdout even on its own error paths, so
	// try to parse regardless of exit code before treating runErr as fatal.
	out, parseErr := parseWrapperOutput(stdout.String())
	if parseErr != nil {
		if runErr != nil {
			return wrapperOutput{}, fmt.Errorf("maigret wrapper failed (%v); stderr: %s", runErr, tail(stderr.String(), 200))
		}
		return wrapperOutput{}, fmt.Errorf("could not parse maigret wrapper output: %v; stdout: %s", parseErr, tail(stdout.String(), 200))
	}
	return out, nil
}

// locateWrapperFile finds a wrapper script by filename. Order: explicit env
// override (envKey), then alongside the running binary (wrapper/<filename> and
// ./<filename>), then relative to the current working directory (covers
// `go test`/`go run` where the binary lives in a temp dir).
func locateWrapperFile(envKey, filename string) (string, error) {
	if p := os.Getenv(envKey); p != "" {
		if fileExists(p) {
			return p, nil
		}
		return "", fmt.Errorf("%s=%q does not exist", envKey, p)
	}

	var candidates []string
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "wrapper", filename),
			filepath.Join(dir, filename),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, "wrapper", filename),
			filepath.Join(wd, "..", "wrapper", filename),
			filepath.Join(wd, "..", "..", "wrapper", filename),
		)
	}
	for _, c := range candidates {
		if fileExists(c) {
			return c, nil
		}
	}
	return "", fmt.Errorf("wrapper script %q not found; set %s to its path — see README", filename, envKey)
}

func parseWrapperOutput(s string) (wrapperOutput, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return wrapperOutput{}, fmt.Errorf("empty output")
	}

	var out wrapperOutput
	// First try the whole string: the wrapper emits compact, single-line JSON.
	if err := json.Unmarshal([]byte(s), &out); err == nil {
		return out, nil
	}

	// Fallback: if stray log lines preceded or followed the JSON, scan for the
	// first well-formed JSON object anywhere in the output (the wrapper emits
	// exactly one). We accept either compact or pretty-printed objects.
	if err := json.Unmarshal([]byte(extractJSONObject(s)), &out); err == nil {
		return out, nil
	}

	return wrapperOutput{}, fmt.Errorf("could not parse wrapper output as JSON")
}

// extractJSONObject pulls the first top-level JSON object from s. It looks for
// the first '{' and returns the substring that balances object braces, or s if
// no object is found. Only '{' and '}' are counted for object boundaries; '['/']'
// (arrays) are ignored so a results array cannot make the depth negative or cause
// an early return at an inner object.
func extractJSONObject(s string) string {
	start := strings.Index(s, "{")
	if start < 0 {
		return s
	}
	depth := 0
	inString := false
	escape := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inString {
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return s[start:]
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func tail(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return "..." + s[len(s)-n:]
	}
	return s
}
