//go:build darwin

package platform

// macOS OCR via the Vision framework's VNRecognizeTextRequest. The
// Objective-C side lives in ocr_darwin.m and ships pixel-space
// bounding boxes already flipped to top-left origin so the Go side
// just unmarshals JSON and slots the regions into []OCRResult.
//
// Cost / characteristics (rough numbers from a Retina M2 MBP):
//   - 1280×800 screenshot: ~80–150 ms
//   - 2940×1912 Retina:    ~250–400 ms
//   - English + Chinese mixed: native via revision 3 (macOS 13+)
//
// Memory model. C side mallocs the JSON buffer; we copy it into a Go
// string immediately and call back into C to free, so the cgo
// boundary doesn't leak C memory if the JSON parse fails.

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework Vision -framework CoreImage -framework ImageIO -framework CoreGraphics

#include <stdlib.h>

char *VisionOCR_RecognizeText(const unsigned char *pngBytes, int pngLen);
void VisionOCR_Free(char *buf);
*/
import "C"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"unsafe"
)

// visionResult is the wire shape ocr_darwin.m hands back via its
// JSON envelope. Kept private to this file since the public surface
// is the OCRResult slice the Platform interface specifies.
type visionResult struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	X          int     `json:"x"`
	Y          int     `json:"y"`
	W          int     `json:"w"`
	H          int     `json:"h"`
}

type visionEnvelope struct {
	Results []visionResult `json:"results"`
	Error   string         `json:"error"`
}

// OCR encodes the source image as PNG once, hands the bytes to the
// Vision framework via cgo, and returns the recognised text regions.
// Always returns a non-nil slice (empty when no text was found) so
// callers can range over the result without a nil-check.
func (p *darwinPlatform) OCR(img image.Image) ([]OCRResult, error) {
	if img == nil {
		return nil, fmt.Errorf("ocr: nil image")
	}

	// PNG-encode once. Vision can decode JPEG too but PNG is
	// lossless — important for screenshots where 1-pixel anti-
	// aliased glyph edges decide whether a UI label is recognised.
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("ocr: png encode: %w", err)
	}
	if buf.Len() == 0 {
		return nil, fmt.Errorf("ocr: empty png after encode")
	}

	raw := buf.Bytes()
	cBuf := C.VisionOCR_RecognizeText(
		(*C.uchar)(unsafe.Pointer(&raw[0])),
		C.int(len(raw)),
	)
	if cBuf == nil {
		return nil, fmt.Errorf("ocr: native call returned NULL")
	}
	goJSON := C.GoString(cBuf)
	C.VisionOCR_Free(cBuf)

	var env visionEnvelope
	if err := json.Unmarshal([]byte(goJSON), &env); err != nil {
		return nil, fmt.Errorf("ocr: bad JSON from native side: %w; payload=%q", err, goJSON)
	}
	if env.Error != "" {
		return nil, fmt.Errorf("ocr: Vision framework: %s", env.Error)
	}

	out := make([]OCRResult, 0, len(env.Results))
	for _, r := range env.Results {
		out = append(out, OCRResult{
			Text: r.Text,
			Bounds: image.Rect(
				r.X, r.Y,
				r.X+r.W, r.Y+r.H,
			),
			Confidence: r.Confidence,
		})
	}
	return out, nil
}
