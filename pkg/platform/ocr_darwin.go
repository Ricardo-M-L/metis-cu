//go:build darwin

package platform

import "image"

// OCR on macOS — VNRecognizeTextRequest via the Vision framework
// would be the native path (free, fast, ~80–150ms on FHD). Pending a
// follow-up cgo binding; for now returns ErrNotImplemented so the
// click_text / find_text_on_screen tools surface a clean "not yet
// implemented on darwin" error rather than wedging.
func (p *darwinPlatform) OCR(_ image.Image) ([]OCRResult, error) {
	return nil, ErrNotImplemented
}
