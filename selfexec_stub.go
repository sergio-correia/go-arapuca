//go:build !linux

package arapuca

// EnableSelfExecMode is only supported on Linux where the
// constructor trampoline (Landlock/seccomp) is available.
// On other platforms it is a no-op.
func EnableSelfExecMode() {}
