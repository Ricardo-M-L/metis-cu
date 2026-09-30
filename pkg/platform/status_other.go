//go:build !darwin

package platform

// PermissionStatus does not infer OS authorization from environment variables
// or attempt input/capture to test it. These platforms lack a TCC equivalent.
func PermissionStatus() map[string]string {
	return map[string]string{"screenRecording": "unknown", "accessibility": "unknown"}
}
