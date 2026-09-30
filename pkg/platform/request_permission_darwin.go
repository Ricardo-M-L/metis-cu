//go:build darwin

package platform

/*
#cgo LDFLAGS: -framework ApplicationServices -framework CoreGraphics -framework CoreFoundation
#include <ApplicationServices/ApplicationServices.h>
#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>

static int metisRequestAccessibilityPermission(void) {
    const void *keys[] = { kAXTrustedCheckOptionPrompt };
    const void *values[] = { kCFBooleanTrue };
    CFDictionaryRef options = CFDictionaryCreate(kCFAllocatorDefault, keys, values, 1,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    if (options == NULL) return -1;
    // The system prompt is asynchronous; false does not mean the request failed.
    (void)AXIsProcessTrustedWithOptions(options);
    CFRelease(options);
    return 0;
}
*/
import "C"

import "fmt"

// RequestPermission may trigger a macOS TCC prompt. Only the explicit CLI
// --request-permission action calls this function; PermissionStatus and
// --describe remain read-only. A pending or declined grant is not an error.
func RequestPermission(kind string) error {
	switch kind {
	case "accessibility":
		if C.metisRequestAccessibilityPermission() != 0 {
			return fmt.Errorf("could not allocate accessibility request options")
		}
	case "screen-recording":
		_ = C.CGRequestScreenCaptureAccess()
	default:
		return fmt.Errorf("unknown permission %q", kind)
	}
	return nil
}
