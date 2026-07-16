package socialfootprint

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// SourceToolSpiderFoot identifies the SpiderFoot engine in audit records and
// result metadata.
const SourceToolSpiderFoot = "SpiderFoot 4.0 (embedded Python library via wrapper subprocess)"

// Environment variables controlling the optional SpiderFoot enrichment path.
const (
	spiderfootEnvEnabled  = "SOCIAL_FOOTPRINT_SPIDERFOOT_ENABLED"
	spiderfootEnvWrapper  = "SOCIAL_FOOTPRINT_SPIDERFOOT_WRAPPER"
	spiderfootEnvPython   = "SOCIAL_FOOTPRINT_SPIDERFOOT_PYTHON"
	spiderfootEnvMaxSites = "SOCIAL_FOOTPRINT_SPIDERFOOT_MAX_PLATFORMS"
	spiderfootEnvRoot     = "SOCIAL_FOOTPRINT_SPIDERFOOT_ROOT"
)

// spiderfootCuratedPlatforms is the platform allow-list passed to the Python
// wrapper. It mirrors Maigret's curated set and is capped at
// SOCIAL_FOOTPRINT_SPIDERFOOT_MAX_PLATFORMS (default 15).
//
// The wrapper maps these display names to the exact WhatsMyName site names
// ("X" for Twitter, "GitHub (User)" for GitHub, "Hacker News" for HackerNews,
// "about.me" for About.me) and seeds the sfp_accounts module with only those
// entries, so it cannot fan out to WhatsMyName's full ~500-site list.
var spiderfootCuratedPlatforms = []string{
	"GitHub", "GitLab", "Reddit", "Twitter", "Instagram",
	"Pinterest", "Medium", "Telegram", "Keybase", "HackerNews",
	"Steam", "SoundCloud", "Vimeo", "About.me", "Patreon",
}

// spiderfootRunner is the production runner for the optional SpiderFoot
// enrichment path. It invokes wrapper/spiderfoot_social.py as a subprocess
// and parses the minimized JSON it prints on stdout.
type spiderfootRunner struct{}

func (s *spiderfootRunner) run(ctx context.Context, handle string, platforms []string, timeout time.Duration) (wrapperOutput, error) {
	python := os.Getenv(spiderfootEnvPython)
	if python == "" {
		python = os.Getenv(pythonEnv)
	}
	if python == "" {
		python = "python3"
	}
	if _, err := exec.LookPath(python); err != nil {
		return wrapperOutput{}, fmt.Errorf("python interpreter %q not found for SpiderFoot; install Python 3.7+ or set %s", python, spiderfootEnvPython)
	}

	wrapper, err := locateWrapperFile(spiderfootEnvWrapper, "spiderfoot_social.py")
	if err != nil {
		return wrapperOutput{}, err
	}

	maxPlatforms := 15
	if v := os.Getenv(spiderfootEnvMaxSites); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxPlatforms = n
		}
	}

	siteTimeout := perSiteTimeoutSeconds
	if secs := int(timeout.Seconds()) - 5; secs > 0 && secs < siteTimeout {
		siteTimeout = secs
	}

	args := []string{
		wrapper,
		"--username", handle,
		"--sites", strings.Join(platforms, ","),
		"--timeout", strconv.Itoa(siteTimeout),
		"--max-sites", strconv.Itoa(maxPlatforms),
	}

	cmd := exec.CommandContext(ctx, python, args...)
	// Propagate the optional SpiderFoot root directory so the wrapper can add
	// it to sys.path without requiring a particular install layout.
	cmd.Env = os.Environ()
	if root := os.Getenv(spiderfootEnvRoot); root != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("SPIDERFOOT_ROOT=%s", root))
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return wrapperOutput{}, fmt.Errorf("spiderfoot check timed out after %s", timeout)
	}

	out, parseErr := parseWrapperOutput(stdout.String())
	if parseErr != nil {
		if runErr != nil {
			return wrapperOutput{}, fmt.Errorf("spiderfoot wrapper failed (%v); stderr: %s", runErr, tail(stderr.String(), 200))
		}
		return wrapperOutput{}, fmt.Errorf("could not parse spiderfoot wrapper output: %v; stdout: %s", parseErr, tail(stdout.String(), 200))
	}
	return out, nil
}

// compile-time interface check.
var _ maigretRunner = (*spiderfootRunner)(nil)
