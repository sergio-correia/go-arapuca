package arapuca

import (
	"os"
	"testing"
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
