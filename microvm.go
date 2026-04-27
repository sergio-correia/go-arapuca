//go:build microvm && linux

package arapuca

/*
#include <arapuca.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

// MicroVmAvailable reports whether KVM and qemu-img are present
// on this system.
func MicroVmAvailable() bool {
	return bool(C.arapuca_microvm_available())
}

// ImagePull downloads and caches a distro cloud image. Returns the
// local path to the cached qcow2 file. This call blocks for the
// duration of the download (potentially minutes for large images).
func ImagePull(distro, version string) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cDistro := C.CString(distro)
	defer C.free(unsafe.Pointer(cDistro))
	cVersion := C.CString(version)
	defer C.free(unsafe.Pointer(cVersion))

	cs := C.arapuca_image_pull(cDistro, cVersion)
	if cs == nil {
		return "", fmt.Errorf("arapuca: image pull: %s", lastError())
	}
	path := C.GoString(cs)
	C.arapuca_free_string(cs)
	return path, nil
}

func applyIsolation(profile unsafe.Pointer, iso *MicroVmIsolation) error {
	if err := iso.validate(); err != nil {
		return err
	}

	p := (*C.struct_arapuca_ArapucaProfile)(profile)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if iso.ImagePath != "" {
		cs := C.CString(iso.ImagePath)
		defer C.free(unsafe.Pointer(cs))
		rc := C.arapuca_profile_set_isolation_microvm_path(
			p, cs, C.uint32_t(iso.CPUs), C.uint32_t(iso.MemMB),
		)
		if rc != 0 {
			return fmt.Errorf("arapuca: %s", lastError())
		}
	} else {
		cDistro := C.CString(iso.Distro)
		defer C.free(unsafe.Pointer(cDistro))
		cVersion := C.CString(iso.Version)
		defer C.free(unsafe.Pointer(cVersion))
		rc := C.arapuca_profile_set_isolation_microvm(
			p, cDistro, cVersion,
			C.uint32_t(iso.CPUs), C.uint32_t(iso.MemMB),
		)
		if rc != 0 {
			return fmt.Errorf("arapuca: %s", lastError())
		}
	}
	return nil
}

func (iso *MicroVmIsolation) validate() error {
	if iso.ImagePath != "" && (iso.Distro != "" || iso.Version != "") {
		return fmt.Errorf("arapuca: ImagePath and Distro/Version are mutually exclusive")
	}
	if iso.ImagePath == "" && iso.Distro == "" {
		return fmt.Errorf("arapuca: either ImagePath or Distro must be set")
	}
	if iso.CPUs == 0 {
		return fmt.Errorf("arapuca: CPUs must be > 0")
	}
	if iso.MemMB == 0 {
		return fmt.Errorf("arapuca: MemMB must be > 0")
	}
	return nil
}
