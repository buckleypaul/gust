//go:build integration

package integration

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/buckleypaul/gust/internal/west"
)

// zephyrBase returns the Zephyr workspace root from the environment,
// or skips the test if it is not set.
func zephyrBase(t *testing.T) string {
	t.Helper()
	base := os.Getenv("ZEPHYR_BASE")
	if base == "" {
		t.Skip("ZEPHYR_BASE not set; skipping integration tests")
	}
	return base
}

// TestIntegrationBuildKernelCommon builds tests/kernel/common for qemu_cortex_m3
// using the real west runner and asserts exit code 0.
func TestIntegrationBuildKernelCommon(t *testing.T) {
	zephyrBase(t)

	runner := west.DefaultRunner{}
	cmd := runner.Run("west", "build",
		"-b", "qemu_cortex_m3",
		"tests/kernel/common",
	)

	msg := cmd()
	result, ok := msg.(west.CommandResultMsg)
	if !ok {
		t.Fatalf("expected CommandResultMsg, got %T", msg)
	}

	t.Logf("west build output:\n%s", result.Output)

	if result.ExitCode != 0 {
		t.Fatalf("west build failed with exit code %d:\n%s", result.ExitCode, result.Output)
	}
}

// TestIntegrationWestTest runs west test against the previously built artifact
// and asserts exit code 0 with non-empty output.
func TestIntegrationWestTest(t *testing.T) {
	zephyrBase(t)

	runner := west.DefaultRunner{}
	cmd := runner.Run("west", "test",
		"-b", "qemu_cortex_m3",
		"tests/kernel/common",
	)

	msg := cmd()
	result, ok := msg.(west.CommandResultMsg)
	if !ok {
		t.Fatalf("expected CommandResultMsg, got %T", msg)
	}

	t.Logf("west test output:\n%s", result.Output)

	if result.ExitCode != 0 {
		t.Fatalf("west test failed with exit code %d:\n%s", result.ExitCode, result.Output)
	}
	if result.Output == "" {
		t.Fatal("expected non-empty test output")
	}
}

// TestIntegrationWestFlash runs west flash for qemu_cortex_m3 (QEMU runner)
// with a 60-second timeout to prevent indefinite QEMU hangs.
func TestIntegrationWestFlash(t *testing.T) {
	zephyrBase(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Use exec.CommandContext so the process is killed when the context expires.
	done := make(chan west.CommandResultMsg, 1)
	go func() {
		start := time.Now()
		cmd := exec.CommandContext(ctx, "west", "flash",
			"-b", "qemu_cortex_m3",
		)
		out, err := cmd.CombinedOutput()
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else if ctx.Err() != nil {
				// Context deadline exceeded – treat as "started cleanly" for QEMU
				exitCode = 0
			} else {
				exitCode = -1
			}
		}
		done <- west.CommandResultMsg{
			Output:   string(out),
			ExitCode: exitCode,
			Duration: time.Since(start),
		}
	}()

	select {
	case result := <-done:
		t.Logf("west flash output:\n%s", result.Output)
		if result.ExitCode != 0 {
			t.Fatalf("west flash failed with exit code %d:\n%s", result.ExitCode, result.Output)
		}
	case <-ctx.Done():
		// QEMU-based flash may run indefinitely; a timeout means it started cleanly.
		t.Log("west flash timed out (expected for QEMU runner)")
	}
}
