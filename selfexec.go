//go:build linux

package arapuca

/*
#include <arapuca.h>
*/
import "C"

// EnableSelfExecMode configures the library to use the current
// executable as the sandbox wrapper binary, eliminating the need
// for a separate arapuca binary. Must be called before any call
// to Launch; if called after, only subsequent launches are affected.
func EnableSelfExecMode() {
	C.arapuca_enable_selfexec_mode()
}
