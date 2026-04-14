// Package arapuca provides Go bindings for the arapuca process
// sandbox library via cgo. It wraps the C API to provide idiomatic
// Go types for sandboxed subprocess management.
//
// Arapuca enforces OS-level isolation (Landlock, seccomp BPF,
// cgroups v2, network namespaces on Linux; sandbox-exec on macOS)
// on untrusted subprocesses.
//
// Thread safety: arapuca_last_error() uses thread-local storage.
// All cgo call sequences that check errors use runtime.LockOSThread()
// to prevent Go's goroutine scheduler from moving the goroutine to a
// different OS thread between the call and the error check.
package arapuca

/*
#cgo linux,amd64  LDFLAGS: -L${SRCDIR}/lib/linux_amd64 -larapuca -ldl -lpthread -lm
#cgo linux,arm64  LDFLAGS: -L${SRCDIR}/lib/linux_arm64 -larapuca -ldl -lpthread -lm
#cgo darwin,amd64 LDFLAGS: -L${SRCDIR}/lib/darwin_amd64 -larapuca -ldl -lpthread -lm -framework Security
#cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/lib/darwin_arm64 -larapuca -ldl -lpthread -lm -framework Security
#cgo CFLAGS: -I${SRCDIR}/lib

#include "arapuca.h"
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

// Sandbox wraps the arapuca sandbox handle. It is the entry point
// for launching sandboxed subprocesses. Create with New().
type Sandbox struct {
	sb *C.struct_arapuca_ArapucaSandbox
	mu sync.Mutex // protects sb pointer
}

// New creates a new sandbox for the current platform.
func New() (*Sandbox, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	sb := C.arapuca_sandbox_new()
	if sb == nil {
		return nil, fmt.Errorf("arapuca: %s", lastError())
	}
	return &Sandbox{sb: sb}, nil
}

// CgroupsAvailable reports whether cgroups v2 resource limits are
// available on this system.
func (s *Sandbox) CgroupsAvailable() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sb == nil {
		return false
	}
	return bool(C.arapuca_sandbox_cgroups_available(s.sb))
}

// Close releases the sandbox handle. Safe to call multiple times.
func (s *Sandbox) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sb != nil {
		C.arapuca_sandbox_free(s.sb)
		s.sb = nil
	}
}

// Launch starts a sandboxed subprocess. The subprocess is isolated
// according to the Config's Profile. Extra file descriptors in
// extraFiles are inherited by the subprocess at FD positions 3, 4, ...
//
// The returned Process must be waited on and cleaned up.
//
// Context cancellation: if ctx is cancelled, the subprocess is killed
// via SIGKILL to the process group.
func (s *Sandbox) Launch(ctx context.Context, cfg Config, cmd string, args []string, extraFiles []*os.File) (*Process, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sb == nil {
		return nil, errors.New("arapuca: sandbox already closed")
	}

	// Build profile.
	profile := C.arapuca_profile_new()
	if profile == nil {
		return nil, errors.New("arapuca: failed to create profile")
	}
	defer C.arapuca_profile_free(profile)

	for _, p := range cfg.Profile.ReadPaths {
		cs := C.CString(p)
		C.arapuca_profile_add_read_path(profile, cs)
		C.free(unsafe.Pointer(cs))
	}
	for _, p := range cfg.Profile.WritePaths {
		cs := C.CString(p)
		C.arapuca_profile_add_write_path(profile, cs)
		C.free(unsafe.Pointer(cs))
	}
	C.arapuca_profile_set_memory_mb(profile, C.uint64_t(cfg.Profile.MaxMemoryMB))
	C.arapuca_profile_set_cpu_pct(profile, C.uint32_t(cfg.Profile.MaxCPUPct))
	C.arapuca_profile_set_max_pids(profile, C.uint32_t(cfg.Profile.MaxPIDs))
	C.arapuca_profile_set_max_file_size_mb(profile, C.uint64_t(cfg.Profile.MaxFileSizeMB))
	C.arapuca_profile_set_netns(profile, C.bool(cfg.Profile.UseNetNS))

	// Build config.
	lcfg := C.arapuca_config_new()
	if lcfg == nil {
		return nil, errors.New("arapuca: failed to create config")
	}
	defer C.arapuca_config_free(lcfg)

	C.arapuca_config_set_profile(lcfg, profile)
	setConfigStr(lcfg, cfg.TaskID, func(c *C.struct_arapuca_ArapucaConfig, cs *C.char) {
		C.arapuca_config_set_task_id(c, cs)
	})
	setConfigStr(lcfg, cfg.Phase, func(c *C.struct_arapuca_ArapucaConfig, cs *C.char) {
		C.arapuca_config_set_phase(c, cs)
	})
	setConfigStr(lcfg, cfg.SocketDir, func(c *C.struct_arapuca_ArapucaConfig, cs *C.char) {
		C.arapuca_config_set_socket_dir(c, cs)
	})
	if cfg.WorkDir != "" {
		setConfigStr(lcfg, cfg.WorkDir, func(c *C.struct_arapuca_ArapucaConfig, cs *C.char) {
			C.arapuca_config_set_work_dir(c, cs)
		})
	}
	if cfg.Stdin != nil {
		C.arapuca_config_set_stdin_fd(lcfg, C.int32_t(cfg.Stdin.Fd()))
	}
	if cfg.Stdout != nil {
		C.arapuca_config_set_stdout_fd(lcfg, C.int32_t(cfg.Stdout.Fd()))
	}
	if cfg.Stderr != nil {
		C.arapuca_config_set_stderr_fd(lcfg, C.int32_t(cfg.Stderr.Fd()))
	}
	if cfg.NetworkProxySocket != "" {
		setConfigStr(lcfg, cfg.NetworkProxySocket, func(c *C.struct_arapuca_ArapucaConfig, cs *C.char) {
			C.arapuca_config_set_network_proxy(c, cs)
		})
	}

	// Build command.
	cCmd := C.CString(cmd)
	defer C.free(unsafe.Pointer(cCmd))

	cArgs := make([]*C.char, len(args))
	for i, a := range args {
		cArgs[i] = C.CString(a)
	}
	defer func() {
		for _, a := range cArgs {
			C.free(unsafe.Pointer(a))
		}
	}()

	var argsPtr **C.char
	if len(cArgs) > 0 {
		argsPtr = &cArgs[0]
	}

	// Extract extra FDs.
	var fds []C.int
	for _, f := range extraFiles {
		fds = append(fds, C.int(f.Fd()))
	}
	var fdsPtr *C.int
	if len(fds) > 0 {
		fdsPtr = &fds[0]
	}

	// Launch.
	runtime.LockOSThread()
	proc := C.arapuca_launch(
		s.sb, lcfg, cCmd,
		argsPtr, C.size_t(len(cArgs)),
		fdsPtr, C.size_t(len(fds)),
	)
	if proc == nil {
		err := fmt.Errorf("arapuca: %s", lastError())
		runtime.UnlockOSThread()
		return nil, err
	}
	runtime.UnlockOSThread()

	// Prevent GC from finalizing the *os.File (and closing the FD)
	// before the C code has duplicated it via F_DUPFD_CLOEXEC.
	runtime.KeepAlive(cfg.Stdin)
	runtime.KeepAlive(cfg.Stdout)
	runtime.KeepAlive(cfg.Stderr)

	pid := int(C.arapuca_process_pid(proc))
	p := &Process{
		proc: proc,
		pid:  pid,
		done: make(chan struct{}),
	}

	// Context cancellation goroutine.
	go func() {
		select {
		case <-ctx.Done():
			// Kill process group. Negative PID = process group.
			// ESRCH is harmless (process already exited).
			_ = syscall.Kill(-pid, syscall.SIGKILL)
		case <-p.done:
			// Process exited normally.
		}
	}()

	return p, nil
}

// Process represents a running sandboxed subprocess.
type Process struct {
	proc *C.struct_arapuca_ArapucaProcess
	pid  int
	done chan struct{}
	mu   sync.Mutex
}

// PID returns the process ID of the sandboxed subprocess.
func (p *Process) PID() int {
	return p.pid
}

// Wait waits for the subprocess to exit. Returns the exit code and
// any error. If the process was killed by a signal, the exit code is
// 0 and the error describes the signal.
func (p *Process) Wait() (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.proc == nil {
		return 0, errors.New("arapuca: process already cleaned up")
	}

	runtime.LockOSThread()
	rc := C.arapuca_process_wait(p.proc)
	var err error
	if rc == -1 {
		err = fmt.Errorf("arapuca: wait: %s", lastError())
	}
	runtime.UnlockOSThread()

	// Signal the cancellation goroutine that we're done.
	select {
	case <-p.done:
	default:
		close(p.done)
	}

	if rc < -1 {
		// Killed by signal. rc = -signal_number.
		return 0, fmt.Errorf("killed by signal %d", -rc)
	}
	if rc == -1 {
		return 0, err
	}
	return int(rc), nil
}

// ResourceStats reads resource usage from the subprocess's cgroup.
// Must be called before Cleanup(). Returns zero values if cgroups
// are unavailable.
func (p *Process) ResourceStats() ResourceUsage {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.proc == nil {
		return ResourceUsage{}
	}
	var stats C.struct_arapuca_ArapucaResourceUsage
	C.arapuca_process_resource_stats(p.proc, &stats)
	return ResourceUsage{
		MemoryCurrentBytes: int64(stats.memory_current_bytes),
		MemoryPeakBytes:    int64(stats.memory_peak_bytes),
		CPUUsageSeconds:    float64(stats.cpu_usage_seconds),
		PIDCount:           int64(stats.pid_count),
		IOReadBytes:        int64(stats.io_read_bytes),
		IOWriteBytes:       int64(stats.io_write_bytes),
	}
}

// OOMCount reads the OOM kill count from the subprocess's cgroup.
// Must be called before Cleanup().
func (p *Process) OOMCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.proc == nil {
		return 0
	}
	return int(C.arapuca_process_oom_count(p.proc))
}

// Cleanup releases the subprocess's resources (temp directory, cgroup).
// Must only be called after Wait() returns. Safe to call multiple times.
func (p *Process) Cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.proc != nil {
		C.arapuca_process_cleanup(p.proc)
		p.proc = nil
	}
}

// ─── Types ──────────────────────────────────────────────────────────

// Profile defines the restrictions applied to a sandboxed subprocess.
type Profile struct {
	ReadPaths     []string // Allowed read-only paths.
	WritePaths    []string // Allowed read-write paths.
	MaxMemoryMB   uint64   // Memory limit in MB (0 = no limit).
	MaxCPUPct     uint32   // CPU percentage (0 = no limit; 200 = 2 cores).
	MaxPIDs       uint32   // Max processes (0 = no limit).
	MaxFileSizeMB uint64   // Max file size in MB (0 = no limit).
	UseNetNS      bool     // Use network namespace isolation.
}

// Config holds the full configuration for launching a sandboxed process.
type Config struct {
	Profile            Profile
	SocketDir          string   // Per-agent socket directory.
	TaskID             string   // Task identifier.
	Phase              string   // Current phase (opaque to arapuca).
	WorkDir            string   // Working directory (empty = inherit).
	Stdin              *os.File // Redirect stdin (nil = inherit).
	Stdout             *os.File // Redirect stdout (nil = inherit).
	Stderr             *os.File // Redirect stderr (nil = inherit).
	NetworkProxySocket string   // Path to network proxy Unix socket.
}

// ResourceUsage holds cgroup v2 resource usage statistics.
type ResourceUsage struct {
	MemoryCurrentBytes int64
	MemoryPeakBytes    int64
	CPUUsageSeconds    float64
	PIDCount           int64
	IOReadBytes        int64
	IOWriteBytes       int64
}

// ─── Utility Functions ──────────────────────────────────────────────

// MakeSocketDir creates a temporary directory with mode 0700 and a
// random suffix for per-agent Unix sockets. The caller is responsible
// for removing the directory when done.
func MakeSocketDir() (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	cs := C.arapuca_make_socket_dir()
	if cs == nil {
		return "", fmt.Errorf("arapuca: %s", lastError())
	}
	dir := C.GoString(cs)
	C.arapuca_free_string(cs)
	return dir, nil
}

// MakeTmpDir creates a temporary directory for the given task with
// a random suffix. The caller is responsible for removing it.
func MakeTmpDir(taskID string) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	cTaskID := C.CString(taskID)
	defer C.free(unsafe.Pointer(cTaskID))
	cs := C.arapuca_make_tmp_dir(cTaskID)
	if cs == nil {
		return "", fmt.Errorf("arapuca: %s", lastError())
	}
	dir := C.GoString(cs)
	C.arapuca_free_string(cs)
	return dir, nil
}

// WrapperPath returns the path to the arapuca binary, or empty
// string if not found.
func WrapperPath() string {
	cs := C.arapuca_wrapper_path()
	if cs == nil {
		return ""
	}
	path := C.GoString(cs)
	C.arapuca_free_string(cs)
	return path
}

// LandlockABIVersion probes the kernel's Landlock ABI version.
// Returns 0 if Landlock is unavailable.
func LandlockABIVersion() int {
	return int(C.arapuca_landlock_abi_version())
}

// NetNSAvailable probes whether network namespace isolation works.
func NetNSAvailable() bool {
	return bool(C.arapuca_netns_available())
}

// DiskUsageMB returns the disk usage of a directory in MB.
func DiskUsageMB(path string) uint64 {
	cs := C.CString(path)
	defer C.free(unsafe.Pointer(cs))
	return uint64(C.arapuca_disk_usage_mb(cs))
}

// ─── Internal helpers ───────────────────────────────────────────────

// lastError reads the thread-local error from arapuca.
// Must be called on the same OS thread as the failing call
// (use runtime.LockOSThread).
func lastError() string {
	cs := C.arapuca_last_error()
	if cs == nil {
		return "unknown error"
	}
	return C.GoString(cs)
}

// setConfigStr is a helper to set a string field on a config handle.
func setConfigStr(cfg *C.struct_arapuca_ArapucaConfig, value string, setter func(*C.struct_arapuca_ArapucaConfig, *C.char)) {
	cs := C.CString(value)
	defer C.free(unsafe.Pointer(cs))
	setter(cfg, cs)
}
