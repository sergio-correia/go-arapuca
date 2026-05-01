package arapuca

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// TestStdioPipeEcho tests basic stdout capture via pipes.
// Launches /bin/echo through the sandbox and reads stdout.
// No Landlock paths = no wrapper binary = simplest path.
func TestStdioPipeEcho(t *testing.T) {
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
		TaskID: "echo-test",
		Stdout: stdoutW,
		Stderr: stderrW,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	proc, err := sb.Launch(ctx, cfg, "/bin/echo", []string{"hello"}, nil)
	if err != nil {
		stdoutW.Close()
		stderrW.Close()
		t.Fatalf("Launch: %v", err)
	}

	// Close child-side FDs in parent.
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
	if got := strings.TrimSpace(string(stdout)); got != "hello" {
		t.Errorf("expected stdout 'hello', got %q (stderr: %q)", got, stderr)
	}
}

// TestStdioPipeEchoWithWrapper tests stdout with Landlock wrapper.
// Setting ReadPaths forces the arapuca wrapper binary to be used.
func TestStdioPipeEchoWithWrapper(t *testing.T) {
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
		TaskID: "echo-wrapper-test",
		Stdout: stdoutW,
		Stderr: stderrW,
		Profile: Profile{
			ReadPaths: []string{"/usr", "/lib", "/lib64", "/bin"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	proc, err := sb.Launch(ctx, cfg, "/bin/echo", []string{"hello"}, nil)
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
	if got := strings.TrimSpace(string(stdout)); got != "hello" {
		t.Errorf("expected stdout 'hello', got %q (stderr: %q)", got, stderr)
	}
}

// TestStdioPipeCatRelay tests stdin→stdout relay through pipes.
func TestStdioPipeCatRelay(t *testing.T) {
	sb, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer sb.Close()

	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	cfg := Config{
		TaskID: "cat-test",
		Stdin:  stdinR,
		Stdout: stdoutW,
		Stderr: stderrW,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	proc, err := sb.Launch(ctx, cfg, "/bin/cat", nil, nil)
	if err != nil {
		stdinR.Close()
		stdoutW.Close()
		stderrW.Close()
		t.Fatalf("Launch: %v", err)
	}

	stdinR.Close()
	stdoutW.Close()
	stderrW.Close()

	fmt.Fprintln(stdinW, "relay-test")
	stdinW.Close()

	stdout, _ := io.ReadAll(stdoutR)
	stderr, _ := io.ReadAll(stderrR)

	exitCode, err := proc.Wait()
	proc.Cleanup()

	t.Logf("exit=%d err=%v stdout=%q stderr=%q", exitCode, err, stdout, stderr)

	if exitCode != 0 {
		t.Errorf("expected exit 0, got %d", exitCode)
	}
	if got := strings.TrimSpace(string(stdout)); got != "relay-test" {
		t.Errorf("expected stdout 'relay-test', got %q", got)
	}
}

// TestStdioPipeCatRelayWithWrapper tests stdin→stdout with wrapper.
func TestStdioPipeCatRelayWithWrapper(t *testing.T) {
	if WrapperPath() == "" {
		t.Skip("arapuca wrapper binary not found")
	}

	sb, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer sb.Close()

	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	cfg := Config{
		TaskID: "cat-wrapper-test",
		Stdin:  stdinR,
		Stdout: stdoutW,
		Stderr: stderrW,
		Profile: Profile{
			ReadPaths: []string{"/usr", "/lib", "/lib64", "/bin"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	proc, err := sb.Launch(ctx, cfg, "/bin/cat", nil, nil)
	if err != nil {
		stdinR.Close()
		stdoutW.Close()
		stderrW.Close()
		t.Fatalf("Launch: %v", err)
	}

	stdinR.Close()
	stdoutW.Close()
	stderrW.Close()

	fmt.Fprintln(stdinW, "relay-test")
	stdinW.Close()

	stdout, _ := io.ReadAll(stdoutR)
	stderr, _ := io.ReadAll(stderrR)

	exitCode, err := proc.Wait()
	proc.Cleanup()

	t.Logf("exit=%d err=%v stdout=%q stderr=%q", exitCode, err, stdout, stderr)

	if exitCode != 0 {
		t.Errorf("expected exit 0, got %d (stderr: %q)", exitCode, stderr)
	}
	if got := strings.TrimSpace(string(stdout)); got != "relay-test" {
		t.Errorf("expected stdout 'relay-test', got %q (stderr: %q)", got, stderr)
	}
}
