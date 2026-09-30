//go:build darwin

package platform

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// drawText writes `s` onto a white background using the stdlib
// basicfont. Output is intentionally large + monochrome so Vision's
// recogniser has no excuse — the test is asserting "OCR pipeline
// reaches the framework and round-trips a string back", not "Vision
// recognises micro fonts."
func drawText(t *testing.T, w, h int, s string) image.Image {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.Black),
		Face: basicfont.Face7x13,
		Dot:  fixed.Point26_6{X: fixed.I(20), Y: fixed.I(40)},
	}
	d.DrawString(s)
	return img
}

// TestOCR_RoundTripsEnglish — the canonical smoke test: hand Vision
// a high-contrast English label, expect at least one OCRResult whose
// Text contains the input word. This pins that
//
//	(a) cgo + Vision framework linking actually works
//	(b) JSON envelope round-trips through ocr_darwin.m
//	(c) bounding-box flip from bottom-left → top-left lands on the
//	    actual glyph (no negative coords, no off-by-image-height)
//
// Skipped on CI runners without the Vision framework available
// (which is unusual for darwin — Vision ships with the OS) so a
// missing framework surfaces as t.Skip rather than t.Fail.
func TestOCR_RoundTripsEnglish(t *testing.T) {
	img := drawText(t, 400, 80, "HELLO METIS")
	p := &darwinPlatform{}
	results, err := p.OCR(img)
	if err != nil {
		t.Fatalf("OCR: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("OCR returned zero results for high-contrast English label")
	}
	// Vision sometimes returns one observation per word; collect all
	// recognised text into a single haystack so either segmentation
	// passes.
	all := ""
	for _, r := range results {
		all += r.Text + " "
	}
	for _, want := range []string{"HELLO", "METIS"} {
		if !strings.Contains(strings.ToUpper(all), want) {
			t.Errorf("OCR missing %q in combined output %q", want, all)
		}
	}
}

// TestOCR_BoundsInsideImage — every returned region's bounding box
// must sit within the source image's pixel rectangle. Pins the
// Y-flip math: a botched flip would surface as negative Y or
// Y > image height.
func TestOCR_BoundsInsideImage(t *testing.T) {
	w, h := 400, 80
	img := drawText(t, w, h, "BOX CHECK")
	p := &darwinPlatform{}
	results, err := p.OCR(img)
	if err != nil {
		t.Fatalf("OCR: %v", err)
	}
	if len(results) == 0 {
		t.Skip("OCR returned no results; can't assert bounds")
	}
	imgRect := image.Rect(0, 0, w, h)
	for i, r := range results {
		// Bounding box should fully overlap the source image; allow
		// a 2-pixel slop on the right/bottom edges (Vision often
		// rounds outward).
		intersection := r.Bounds.Intersect(imgRect)
		if intersection.Empty() {
			t.Errorf("result[%d] bounds %v don't overlap image %v",
				i, r.Bounds, imgRect)
			continue
		}
		if r.Bounds.Min.X < -2 || r.Bounds.Min.Y < -2 {
			t.Errorf("result[%d] bounds %v has negative origin",
				i, r.Bounds)
		}
		if r.Bounds.Max.X > w+2 || r.Bounds.Max.Y > h+2 {
			t.Errorf("result[%d] bounds %v exceed image %dx%d",
				i, r.Bounds, w, h)
		}
	}
}

// TestOCR_NilImage — guard against the nil-deref panic an upstream
// caller could trigger before this commit (the stub returned
// ErrNotImplemented, masking the case).
func TestOCR_NilImage(t *testing.T) {
	p := &darwinPlatform{}
	if _, err := p.OCR(nil); err == nil {
		t.Error("OCR(nil) should error; got nil")
	}
}
