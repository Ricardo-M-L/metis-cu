package platform

import (
	"image"
	"image/color"
	"testing"
)

// TestNormaliseToLogical_PassthroughWhenAlreadyMatches: a 1440x900
// capture on a 1440x900 logical display must NOT allocate a new image
// — verifies the fast-path that protects non-Retina users from the
// CatmullRom cost on every screenshot.
func TestNormaliseToLogical_PassthroughWhenAlreadyMatches(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1440, 900))
	got := normaliseToLogical(src, 1440, 900)
	// Same pointer = passthrough.
	if got != image.Image(src) {
		t.Errorf("expected passthrough when bounds already match logical dims")
	}
}

// TestNormaliseToLogical_DownsamplesRetina: 2x physical (2880x1800)
// from a 1440x900 logical display must collapse to 1440x900 — the
// canonical Retina case. Without this, BUG-7 makes every model click
// land at 2x its target.
func TestNormaliseToLogical_DownsamplesRetina(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2880, 1800))
	got := normaliseToLogical(src, 1440, 900)
	if got.Bounds().Dx() != 1440 || got.Bounds().Dy() != 900 {
		t.Errorf("Retina 2x: got %dx%d, want 1440x900",
			got.Bounds().Dx(), got.Bounds().Dy())
	}
}

// TestNormaliseToLogical_FractionalScale: GNOME 150% scaling — a
// 2160x1350 capture on a 1440x900 logical display.
func TestNormaliseToLogical_FractionalScale(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2160, 1350))
	got := normaliseToLogical(src, 1440, 900)
	if got.Bounds().Dx() != 1440 || got.Bounds().Dy() != 900 {
		t.Errorf("1.5x: got %dx%d, want 1440x900",
			got.Bounds().Dx(), got.Bounds().Dy())
	}
}

// TestNormaliseToLogical_DegenerateLogicalDimsIsNoop: if the platform
// reports zero/negative logical dims (shouldn't happen, but defensive),
// return the source unchanged rather than allocating a 0x0 image.
func TestNormaliseToLogical_DegenerateLogicalDimsIsNoop(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for _, c := range []struct{ w, h int }{{0, 100}, {100, 0}, {-1, 100}, {100, -1}, {0, 0}} {
		got := normaliseToLogical(src, c.w, c.h)
		if got != image.Image(src) {
			t.Errorf("logical dims %dx%d should passthrough, got new image %dx%d",
				c.w, c.h, got.Bounds().Dx(), got.Bounds().Dy())
		}
	}
}

// TestNormaliseToLogical_PreservesRoughVisualContent: paint a black
// dot at the centre of a 4x physical capture, downsample to logical;
// the centre should still be darker than the corners. Sanity-only —
// CatmullRom is well-tested upstream.
func TestNormaliseToLogical_PreservesRoughVisualContent(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 400, 400))
	// White background.
	for y := 0; y < 400; y++ {
		for x := 0; x < 400; x++ {
			src.Set(x, y, color.RGBA{255, 255, 255, 255})
		}
	}
	// Black 40x40 square at centre.
	for y := 180; y < 220; y++ {
		for x := 180; x < 220; x++ {
			src.Set(x, y, color.RGBA{0, 0, 0, 255})
		}
	}
	got := normaliseToLogical(src, 100, 100)
	centre := got.At(50, 50)
	corner := got.At(5, 5)
	cR, _, _, _ := centre.RGBA()
	kR, _, _, _ := corner.RGBA()
	if cR >= kR {
		t.Errorf("expected centre (R=%d) darker than corner (R=%d) after downsample", cR, kR)
	}
}
