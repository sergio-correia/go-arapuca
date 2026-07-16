//go:build !linux

package arapuca

/*
#include <arapuca.h>
*/
import "C"

import "errors"

// addAllowedHosts is not supported on non-Linux platforms because the
// CONNECT proxy FFI is Linux-only.
func addAllowedHosts(cfg *C.struct_arapuca_ArapucaConfig, hosts []AllowedHost) error {
	if len(hosts) > 0 {
		return errors.New("arapuca: AllowedHosts is only supported on Linux")
	}
	return nil
}
