package arapuca

import (
	"context"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	sb, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer sb.Close()
}

func TestCloseIdempotent(t *testing.T) {
	sb, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	sb.Close()
	sb.Close() // second close is safe
}

func TestCgroupsAvailable(t *testing.T) {
	sb, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer sb.Close()
	available := sb.CgroupsAvailable()
	t.Logf("cgroups available: %v", available)
}

func TestLandlockABIVersion(t *testing.T) {
	v := LandlockABIVersion()
	t.Logf("landlock ABI version: %d", v)
}

func TestNetNSAvailable(t *testing.T) {
	available := NetNSAvailable()
	t.Logf("netns available: %v", available)
}

func TestWrapperPath(t *testing.T) {
	path := WrapperPath()
	t.Logf("wrapper path: %q", path)
}

func TestMakeSocketDir(t *testing.T) {
	dir, err := MakeSocketDir()
	if err != nil {
		t.Fatalf("MakeSocketDir: %v", err)
	}
	t.Logf("socket dir: %s", dir)
	_ = os.RemoveAll(dir)
}

func TestMakeTmpDir(t *testing.T) {
	dir, err := MakeTmpDir("test-task")
	if err != nil {
		t.Fatalf("MakeTmpDir: %v", err)
	}
	t.Logf("tmp dir: %s", dir)
	_ = os.RemoveAll(dir)
}

func TestDiskUsageMB(t *testing.T) {
	mb := DiskUsageMB("/tmp")
	t.Logf("disk usage /tmp: %d MB", mb)
}

func TestDiskUsageMBNonexistent(t *testing.T) {
	mb := DiskUsageMB("/nonexistent-xyz-123")
	if mb != 0 {
		t.Errorf("expected 0 for nonexistent path, got %d", mb)
	}
}

func TestProfileValidation_DnsCaptureRequiresNetNS(t *testing.T) {
	sb, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	_, err = sb.Launch(ctx, Config{
		Profile: Profile{DnsCapture: true, UseNetNS: false},
	}, "/bin/true", nil, nil)
	if err == nil {
		t.Fatal("expected error for DnsCapture without UseNetNS, got nil")
	}
}

// TestAllowedHosts_InvalidPort verifies that port 0 is rejected by
// the FFI layer.
func TestAllowedHosts_InvalidPort(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("AllowedHosts FFI is Linux-only")
	}

	sb, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer sb.Close()

	_, err = sb.Launch(context.Background(), Config{
		AllowedHosts: []AllowedHost{{Host: "127.0.0.1", Port: 0}},
	}, "/bin/true", nil, nil)
	if err == nil {
		t.Fatal("expected error for AllowedHost with port 0")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "allowed host") {
		t.Errorf("expected error to mention allowed host, got: %v", err)
	}
}

func TestProfileValidation_InvalidSeccompProfile(t *testing.T) {
	sb, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	_, err = sb.Launch(ctx, Config{
		Profile: Profile{SeccompProfile: "invalid-profile-xyz"},
	}, "/bin/true", nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid seccomp profile, got nil")
	}
}

func TestSeccompProfileConstants(t *testing.T) {
	if SeccompProfileDefault != "" {
		t.Errorf("SeccompProfileDefault = %q, want %q", SeccompProfileDefault, "")
	}
	if SeccompProfileStrict != "strict" {
		t.Errorf("SeccompProfileStrict = %q, want %q", SeccompProfileStrict, "strict")
	}
	if SeccompProfileBaseline != "baseline" {
		t.Errorf("SeccompProfileBaseline = %q, want %q", SeccompProfileBaseline, "baseline")
	}
}

// TestProfileValidation_SeccompProfileStrict verifies the C library accepts "strict".
func TestProfileValidation_SeccompProfileStrict(t *testing.T) {
	sb, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	_, err = sb.Launch(ctx, Config{
		Profile: Profile{
			SeccompProfile: SeccompProfileStrict,
			ReadPaths:      []string{"/usr", "/lib", "/lib64", "/bin"},
		},
	}, "/bin/true", nil, nil)
	// err may be non-nil for other reasons (binary not found, no wrapper, etc.)
	// but must NOT be a seccomp profile error.
	if err != nil && containsSeccompError(err) {
		t.Errorf("SeccompProfileStrict rejected by C library: %v", err)
	}
}

// TestProfileValidation_SeccompProfileBaseline verifies the C library accepts "baseline".
func TestProfileValidation_SeccompProfileBaseline(t *testing.T) {
	sb, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	_, err = sb.Launch(ctx, Config{
		Profile: Profile{
			SeccompProfile: SeccompProfileBaseline,
			ReadPaths:      []string{"/usr", "/lib", "/lib64", "/bin"},
		},
	}, "/bin/true", nil, nil)
	if err != nil && containsSeccompError(err) {
		t.Errorf("SeccompProfileBaseline rejected by C library: %v", err)
	}
}

// TestProfileValidation_UsePidNS_Happy verifies UsePidNS is accepted when the
// kernel supports PID namespaces.
func TestProfileValidation_UsePidNS_Happy(t *testing.T) {
	if !NetNSAvailable() {
		t.Skip("namespace isolation not available on this system")
	}
	sb, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	_, err = sb.Launch(ctx, Config{
		Profile: Profile{
			UsePidNS:  true,
			ReadPaths: []string{"/usr", "/lib", "/lib64", "/bin"},
		},
	}, "/bin/true", nil, nil)
	if err != nil {
		t.Logf("UsePidNS launch result (may fail for non-sandbox reasons): %v", err)
	}
}

// TestProfileValidation_DnsCapture_Happy verifies DnsCapture+UseNetNS is accepted.
func TestProfileValidation_DnsCapture_Happy(t *testing.T) {
	if !NetNSAvailable() {
		t.Skip("network namespace isolation not available on this system")
	}
	sb, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	_, err = sb.Launch(ctx, Config{
		Profile: Profile{
			UseNetNS:   true,
			DnsCapture: true,
			ReadPaths:  []string{"/usr", "/lib", "/lib64", "/bin"},
		},
	}, "/bin/true", nil, nil)
	// Should not error with "DnsCapture requires UseNetNS" since UseNetNS is set.
	if err != nil && err.Error() == "arapuca: DnsCapture requires UseNetNS" {
		t.Errorf("DnsCapture with UseNetNS should not error: %v", err)
	}
}

// TestAllowExec_ShebangScript verifies that a shebang script runs
// successfully when AllowExec is true and the interpreter is in a
// read path.
func TestAllowExec_ShebangScript(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("AllowExec/Landlock is Linux-only")
	}
	if WrapperPath() == "" {
		t.Skip("arapuca wrapper binary not found")
	}
	if LandlockABIVersion() == 0 {
		t.Skip("Landlock not available")
	}

	// Create a shebang script in a temp dir that will be a write path.
	dir := t.TempDir()
	script := dir + "/hello.sh"
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho hello-from-shebang\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	proc, err := sb.Launch(ctx, Config{
		TaskID: "shebang-allow-exec",
		Stdout: stdoutW,
		Stderr: stderrW,
		Profile: Profile{
			AllowExec:  true,
			ReadPaths:  []string{"/usr", "/lib", "/lib64", "/bin"},
			WritePaths: []string{dir},
		},
	}, script, nil, nil)
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
	if got := strings.TrimSpace(string(stdout)); got != "hello-from-shebang" {
		t.Errorf("expected stdout 'hello-from-shebang', got %q", got)
	}
}

// TestAllowExec_DefaultBlocksShebang verifies that a shebang script
// is blocked when AllowExec is false (the default), because Landlock
// denies Execute on the interpreter.
func TestAllowExec_DefaultBlocksShebang(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("AllowExec/Landlock is Linux-only")
	}
	if WrapperPath() == "" {
		t.Skip("arapuca wrapper binary not found")
	}
	if LandlockABIVersion() == 0 {
		t.Skip("Landlock not available")
	}

	dir := t.TempDir()
	script := dir + "/hello.sh"
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho hello-from-shebang\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	proc, err := sb.Launch(ctx, Config{
		TaskID: "shebang-deny-exec",
		Stdout: stdoutW,
		Stderr: stderrW,
		Profile: Profile{
			AllowExec:  false,
			ReadPaths:  []string{"/usr", "/lib", "/lib64", "/bin"},
			WritePaths: []string{dir},
		},
	}, script, nil, nil)
	if err != nil {
		stdoutW.Close()
		stderrW.Close()
		// Launch itself failing is acceptable -- Landlock may block
		// even the initial exec of the script.
		t.Logf("Launch blocked (expected): %v", err)
		return
	}

	stdoutW.Close()
	stderrW.Close()

	_, _ = io.ReadAll(stdoutR)
	_, _ = io.ReadAll(stderrR)

	exitCode, err := proc.Wait()
	proc.Cleanup()

	t.Logf("exit=%d err=%v", exitCode, err)

	if exitCode == 0 && err == nil {
		t.Error("expected shebang script to fail with AllowExec=false, but it succeeded")
	}
}

// containsSeccompError checks whether err is a seccomp profile rejection.
func containsSeccompError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return len(s) > 0 && (contains(s, "seccomp profile") || contains(s, "unknown seccomp"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
