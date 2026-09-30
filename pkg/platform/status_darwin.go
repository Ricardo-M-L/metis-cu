//go:build darwin

package platform

/*
#cgo LDFLAGS: -framework ApplicationServices -framework CoreGraphics
#include <ApplicationServices/ApplicationServices.h>
#include <CoreGraphics/CoreGraphics.h>
*/
import "C"

// PermissionStatus probes TCC without opening settings, requesting access,
// capturing a frame, or creating a platform backend.
func PermissionStatus() map[string]string {
	state := func(granted bool) string {
		if granted {
			return "granted"
		}
		return "notGranted"
	}
	return map[string]string{
		"screenRecording": state(bool(C.CGPreflightScreenCaptureAccess())),
		"accessibility":   state(C.AXIsProcessTrusted() != 0),
	}
}
