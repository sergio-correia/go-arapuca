//go:build linux

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

// addAllowedHosts calls arapuca_config_add_allowed_host for each entry.
func addAllowedHosts(cfg *C.struct_arapuca_ArapucaConfig, hosts []AllowedHost) error {
	for _, ah := range hosts {
		ch := C.CString(ah.Host)
		runtime.LockOSThread()
		rc := C.arapuca_config_add_allowed_host(cfg, ch, C.uint16_t(ah.Port))
		if rc != 0 {
			err := fmt.Errorf("arapuca: add allowed host %s:%d: %s", ah.Host, ah.Port, lastError())
			runtime.UnlockOSThread()
			C.free(unsafe.Pointer(ch))
			return err
		}
		runtime.UnlockOSThread()
		C.free(unsafe.Pointer(ch))
	}
	return nil
}
