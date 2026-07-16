//go:build integration

package socialfootprint

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

// TestSubprocessRunner_RealMaigret runs a real Maigret subprocess against a
// known, safe public handle. It is guarded by the `integration` build tag so
// `go test ./...` (the default) does not trigger network calls; run it with:
//
//	go test -tags integration ./...
//
// or `make test-integration`.
func TestSubprocessRunner_RealMaigret(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}

	runner := &subprocessRunner{}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	out, err := runner.run(ctx, "soxoj", curatedPlatforms, 120*time.Second)
	if err != nil {
		t.Fatalf("real Maigret run failed: %v", err)
	}
	if out.Error != "" {
		t.Fatalf("real Maigret run returned wrapper error: %s", out.Error)
	}
	if len(out.Results) == 0 {
		t.Fatal("expected at least one platform result from real Maigret run")
	}
}
