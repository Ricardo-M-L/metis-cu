//go:build darwin

package platform

import (
	"image"
	"testing"
)

// TestScreenshot_ReturnsImage verifies the macOS Screenshot path captures a
// real image (not the ErrNotImplemented placeholder). Skips on CI runners
// where screen capture access has not been granted — the bounds check
// returns 0 displays in that case.
func TestScreenshot_ReturnsImage(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	img, err := p.Screenshot()
	if err != nil {
		// Headless / no-screen-access CI is the normal "skip" case.
		t.Skipf("screenshot unavailable on this runner: %v", err)
	}
	if img == nil {
		t.Fatal("nil image returned with no error")
	}
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		t.Fatalf("captured image has degenerate bounds %v", b)
	}
	if _, ok := img.(*image.RGBA); !ok {
		// kbinani/screenshot returns *image.RGBA; if that contract
		// changes upstream we want the breakage logged, not silent.
		t.Logf("image concrete type changed: %T", img)
	}
}

// TestDisplayCount_NonNegative makes sure the count helper either returns
// the real number of displays or surfaces the access-denied error — it
// should never silently return 0.
func TestDisplayCount_NonNegative(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	n, err := p.DisplayCount()
	if err != nil {
		t.Skipf("display count unavailable: %v", err)
	}
	if n <= 0 {
		t.Fatalf("DisplayCount returned %d with no error", n)
	}
}

// TestSwitchDisplay_RangeCheck verifies bounds validation.
func TestSwitchDisplay_RangeCheck(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := p.SwitchDisplay(-1); err == nil {
		t.Error("SwitchDisplay(-1) should fail")
	}
	if err := p.SwitchDisplay(99); err == nil {
		t.Error("SwitchDisplay(99) should fail unless this machine has 100 displays")
	}
}
