package arapuca

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// TestStdioGoProbe launches a Go binary through the sandbox with pipes.
// This is the exact pattern that fails in the wtmcp issue.
func TestStdioGoProbe(t *testing.T) {
	const probeBin = "/tmp/arapuca-probe"
	if _, err := os.Stat(probeBin); err != nil {
		t.Skipf("probe binary not found at %s (build with: go build -o %s /tmp/arapuca-probe.go)", probeBin, probeBin)
	}

	sb, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer sb.Close()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	cfg := Config{
		TaskID: "go-probe-test",
		Stdout: stdoutW,
		Stderr: stderrW,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	proc, err := sb.Launch(ctx, cfg, probeBin, nil, nil)
	if err != nil {
		stdoutW.Close()
		stderrW.Close()
		t.Fatalf("Launch: %v", err)
	}

	stdoutW.Close()
	stderrW.Close()

	stdout, _ := io.ReadAll(stdoutR)
	stderr, _ := io.ReadAll(stderrR)

	exitCode, err := proc.Wait()
	proc.Cleanup()

	t.Logf("exit=%d err=%v stdout=%q stderr=%q", exitCode, err, stdout, stderr)

	if exitCode != 0 {
		t.Errorf("expected exit 0, got %d (err: %v, stderr: %q)", exitCode, err, stderr)
	}
	if got := strings.TrimSpace(string(stdout)); got != "go-alive" {
		t.Errorf("expected stdout 'go-alive', got %q (stderr: %q)", got, stderr)
	}
}

// TestStdioGoProbeWithWrapper same but with Landlock wrapper.
func TestStdioGoProbeWithWrapper(t *testing.T) {
	const probeBin = "/tmp/arapuca-probe"
	if _, err := os.Stat(probeBin); err != nil {
		t.Skipf("probe binary not found at %s", probeBin)
	}
	if WrapperPath() == "" {
		t.Skip("arapuca wrapper binary not found")
	}

	sb, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer sb.Close()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	cfg := Config{
		TaskID: "go-probe-wrapper-test",
		Stdout: stdoutW,
		Stderr: stderrW,
		Profile: Profile{
			ReadPaths: []string{"/usr", "/lib", "/lib64", "/bin", "/tmp", "/proc/self"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	proc, err := sb.Launch(ctx, cfg, probeBin, nil, nil)
	if err != nil {
		stdoutW.Close()
		stderrW.Close()
		t.Fatalf("Launch: %v", err)
	}

	stdoutW.Close()
	stderrW.Close()

	stdout, _ := io.ReadAll(stdoutR)
	stderr, _ := io.ReadAll(stderrR)

	exitCode, err := proc.Wait()
	proc.Cleanup()

	t.Logf("exit=%d err=%v stdout=%q stderr=%q", exitCode, err, stdout, stderr)

	if exitCode != 0 {
		t.Errorf("expected exit 0, got %d (err: %v, stderr: %q)", exitCode, err, stderr)
	}
	if got := strings.TrimSpace(string(stdout)); got != "go-alive" {
		t.Errorf("expected stdout 'go-alive', got %q (stderr: %q)", got, stderr)
	}
}

// TestStdioGoProbeWithMemoryLimit same but with memory limit (the wtmcp config).
func TestStdioGoProbeWithMemoryLimit(t *testing.T) {
	const probeBin = "/tmp/arapuca-probe"
	if _, err := os.Stat(probeBin); err != nil {
		t.Skipf("probe binary not found at %s", probeBin)
	}
	if WrapperPath() == "" {
		t.Skip("arapuca wrapper binary not found")
	}

	sb, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer sb.Close()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	cfg := Config{
		TaskID: "go-probe-mem-test",
		Stdout: stdoutW,
		Stderr: stderrW,
		Profile: Profile{
			ReadPaths:   []string{"/usr", "/lib", "/lib64", "/bin", "/tmp", "/proc/self"},
			MaxMemoryMB: 256,
			MaxCPUPct:   100,
			MaxPIDs:     32,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	proc, err := sb.Launch(ctx, cfg, probeBin, nil, nil)
	if err != nil {
		stdoutW.Close()
		stderrW.Close()
		t.Fatalf("Launch: %v", err)
	}

	stdoutW.Close()
	stderrW.Close()

	stdout, _ := io.ReadAll(stdoutR)
	stderr, _ := io.ReadAll(stderrR)

	exitCode, err := proc.Wait()
	proc.Cleanup()

	t.Logf("exit=%d err=%v stdout=%q stderr=%q", exitCode, err, stdout, stderr)

	if exitCode != 0 {
		t.Errorf("expected exit 0, got %d (err: %v, stderr: %q)", exitCode, err, stderr)
	}
	if got := strings.TrimSpace(string(stdout)); got != "go-alive" {
		t.Errorf("expected stdout 'go-alive', got %q (stderr: %q)", got, stderr)
	}
}
