//go:build !darwin

package platform

import (
	"fmt"
	"runtime"
)

// RequestPermission is available only on macOS, where the OS exposes explicit
// TCC requests for accessibility and screen recording.
func RequestPermission(kind string) error {
	if kind != "accessibility" && kind != "screen-recording" {
		return fmt.Errorf("unknown permission %q", kind)
	}
	return fmt.Errorf("requesting %s permission is unsupported on %s", kind, runtime.GOOS)
}
