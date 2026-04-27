//go:build !microvm || !linux

package arapuca

import (
	"errors"
	"unsafe"
)

// MicroVmAvailable always returns false without the microvm build tag.
func MicroVmAvailable() bool { return false }

// ImagePull is unavailable without the microvm build tag.
func ImagePull(_, _ string) (string, error) {
	return "", errors.New("arapuca: microvm support not compiled (build with -tags microvm)")
}

func applyIsolation(_ unsafe.Pointer, _ *MicroVmIsolation) error {
	return errors.New("arapuca: microvm support not compiled (build with -tags microvm)")
}
