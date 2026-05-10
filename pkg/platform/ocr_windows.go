//go:build windows

package platform

import "image"

// OCR on Windows — Windows.Media.Ocr via WinRT would be the native
// path. Pending a follow-up cgo binding; for now returns
// ErrNotImplemented so click_text / find_text_on_screen surface a
// clean error.
func (p *windowsPlatform) OCR(_ image.Image) ([]OCRResult, error) {
	return nil, ErrNotImplemented
}
