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

// Compile-time interface checks.
var _ maigretRunner = (*sherlockRunner)(nil)
var _ maigretRunner = (*bothRunner)(nil)

const (
	sherlockPythonEnv  = "SOCIAL_FOOTPRINT_SHERLOCK_PYTHON"
	sherlockWrapperEnv = "SOCIAL_FOOTPRINT_SHERLOCK_WRAPPER"

	// sherlockPerSiteTimeoutSeconds is passed as --timeout to the wrapper.
	// Sherlock runs sites in parallel threads internally, so total wall time
	// is bounded by the slowest site, not the sum of all sites.
	sherlockPerSiteTimeoutSeconds = 12
)

// sherlockCuratedPlatforms is the Sherlock-specific allow-list.
// Names must exactly match keys in Sherlock's bundled data.json (verified v0.16.1).
// This is a compile-time constant; do not make it runtime-configurable.
var sherlockCuratedPlatforms = []string{
	"GitHub", "GitLab", "Reddit", "Twitter", "Instagram",
	"Pinterest", "Medium", "Telegram", "Keybase", "HackerNews",
	"Steam", "SoundCloud", "Vimeo", "Patreon",
}

// sherlockRunner implements maigretRunner using wrapper/sherlock_check.py.
type sherlockRunner struct{}

func (s *sherlockRunner) run(ctx context.Context, handle string, platforms []string, timeout time.Duration) (wrapperOutput, error) {
	python := os.Getenv(sherlockPythonEnv)
	if python == "" {
		python = os.Getenv(pythonEnv) // fall back to shared SOCIAL_FOOTPRINT_PYTHON
		if python == "" {
			python = "python3"
		}
	}
	if _, err := exec.LookPath(python); err != nil {
		return wrapperOutput{}, fmt.Errorf(
			"python interpreter %q not found for sherlock; install Python 3.10+ or set %s",
			python, sherlockPythonEnv,
		)
	}

	wrapper, err := locateSherlockWrapper()
	if err != nil {
		return wrapperOutput{}, err
	}

	siteTimeout := sherlockPerSiteTimeoutSeconds
	if secs := int(timeout.Seconds()) - 5; secs > 0 && secs < siteTimeout {
		siteTimeout = secs
	}

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
		return wrapperOutput{}, fmt.Errorf("sherlock check timed out after %s", timeout)
	}

	// Try to parse JSON regardless of exit code — wrapper emits JSON on error paths too.
	out, parseErr := parseWrapperOutput(stdout.String()) // defined in maigret.go
	if parseErr != nil {
		if runErr != nil {
			return wrapperOutput{}, fmt.Errorf(
				"sherlock wrapper failed (%v); stderr: %s", runErr, tail(stderr.String(), 200),
			)
		}
		return wrapperOutput{}, fmt.Errorf(
			"could not parse sherlock wrapper output: %v; stdout: %s",
			parseErr, tail(stdout.String(), 200),
		)
	}

	// TODO: set out.SourceTool = SourceToolSherlock once wrapperOutput carries a
	// SourceTool field (it currently does not, so we skip this fix).
	return out, nil
}

// locateSherlockWrapper finds wrapper/sherlock_check.py using the shared
// wrapper locator from maigret.go.
func locateSherlockWrapper() (string, error) {
	return locateWrapperFile(sherlockWrapperEnv, "sherlock_check.py")
}

// bothRunner implements maigretRunner by running Maigret (primary) then
// Sherlock (secondary) and merging results.
//
// Merge rule:
//   - Sherlock upgrades a platform result ONLY when Maigret returned "unknown"
//     AND Sherlock has a definitive "claimed" or "available" answer.
//   - Maigret's "claimed" or "available" is never downgraded.
//   - Sherlock platforms not covered by Maigret are appended if definitive.
type bothRunner struct {
	primary   maigretRunner // Maigret
	secondary maigretRunner // Sherlock
}

func (b *bothRunner) run(ctx context.Context, handle string, platforms []string, timeout time.Duration) (wrapperOutput, error) {
	// Split the total wall-clock budget so the two sequential runs cannot exceed
	// the caller's deadline: primary gets up to 60% (minimum 5s), secondary gets
	// the remainder (minimum 5s). Use child contexts so each runner respects its
	// own slice while still inheriting cancellation from the parent context.
	primaryTimeout := timeout * 60 / 100
	if primaryTimeout < 5*time.Second {
		primaryTimeout = 5 * time.Second
	}
	secondaryTimeout := timeout - primaryTimeout
	if secondaryTimeout < 5*time.Second {
		secondaryTimeout = 5 * time.Second
	}

	primaryCtx, primaryCancel := context.WithTimeout(ctx, primaryTimeout)
	defer primaryCancel()
	primaryOut, primaryErr := b.primary.run(primaryCtx, handle, platforms, primaryTimeout)
	if primaryErr != nil {
		// Primary failed: fall back to Sherlock alone with the caller's platform list.
		secondaryCtx, secondaryCancel := context.WithTimeout(ctx, secondaryTimeout)
		defer secondaryCancel()
		return b.secondary.run(secondaryCtx, handle, platforms, secondaryTimeout)
	}

	// Index primary results for O(1) lookup.
	primaryByPlatform := make(map[string]int, len(primaryOut.Results))
	for i, r := range primaryOut.Results {
		primaryByPlatform[r.Platform] = i
	}

	merged := make([]platformResult, len(primaryOut.Results))
	copy(merged, primaryOut.Results)

	// Run Sherlock with its own curated platform list.
	secondaryCtx, secondaryCancel := context.WithTimeout(ctx, secondaryTimeout)
	defer secondaryCancel()
	secondaryOut, secondaryErr := b.secondary.run(secondaryCtx, handle, sherlockCuratedPlatforms, secondaryTimeout)
	if secondaryErr == nil {
		for _, sr := range secondaryOut.Results {
			definitive := sr.Status == "claimed" || sr.Status == "available"
			if i, ok := primaryByPlatform[sr.Platform]; ok {
				if merged[i].Status == "unknown" && definitive {
					merged[i] = sr // upgrade
				}
				// If Maigret was definitive, never overwrite — do nothing.
			} else if definitive {
				// Platform Maigret didn't check — add Sherlock's definitive result.
				merged = append(merged, sr)
			}
		}
	}

	primaryOut.Results = merged
	return primaryOut, nil
}
