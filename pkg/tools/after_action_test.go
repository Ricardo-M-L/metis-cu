package tools

import (
	"context"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type fakeAfterActionPlat struct {
	stubPlat
	shotCalled int
}

func (p *fakeAfterActionPlat) Screenshot() (image.Image, error) {
	p.shotCalled++
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	img.Set(0, 0, color.RGBA{R: 200, A: 255})
	return img, nil
}

// TestSettleAndMaybeShot_DefaultTrueAutoShot: 2026-05-22 — default
// flipped to true. Calling with empty params should NOW return a
// screenshot (the "saves a model round-trip" behavior). The old
// "no-op when omitted" test is replaced with this one.
func TestSettleAndMaybeShot_DefaultTrueAutoShot(t *testing.T) {
	p := &fakeAfterActionPlat{}
	img, mime := settleAndMaybeShot(context.Background(), p, map[string]any{})
	if img == "" || mime == "" {
		t.Errorf("default-true should auto-capture; got (%q, %q)", img, mime)
	}
	if p.shotCalled != 1 {
		t.Errorf("Screenshot called %d times; want 1 (default-true)", p.shotCalled)
	}
}

// TestSettleAndMaybeShot_ExplicitFalseSkips: the opt-out path.
// Callers wanting to batch unattended actions can pass false to
// suppress the auto-capture.
func TestSettleAndMaybeShot_ExplicitFalseSkips(t *testing.T) {
	p := &fakeAfterActionPlat{}
	img, mime := settleAndMaybeShot(context.Background(), p, map[string]any{"return_screenshot": false})
	if img != "" || mime != "" {
		t.Errorf("explicit false should suppress; got (%q, %q)", img, mime)
	}
	if p.shotCalled != 0 {
		t.Errorf("Screenshot called %d times; want 0", p.shotCalled)
	}
}

// TestSettleAndMaybeShot_ReturnScreenshotTrue: handler-supplied
// return_screenshot=true triggers Screenshot + encode, even with no
// settle configured.
func TestSettleAndMaybeShot_ReturnScreenshotTrue(t *testing.T) {
	p := &fakeAfterActionPlat{}
	img, mime := settleAndMaybeShot(context.Background(), p, map[string]any{"return_screenshot": true})
	if img == "" || mime == "" {
		t.Fatalf("expected non-empty image+mime; got (%q, %q)", img, mime)
	}
	if mime != "image/png" {
		t.Errorf("default mime should be image/png; got %q", mime)
	}
	if p.shotCalled != 1 {
		t.Errorf("Screenshot called %d times; want 1", p.shotCalled)
	}
}

// TestSettleAndMaybeShot_SettleViaRegistry: a positive MouseSettleMs
// on the Registry causes the helper to sleep — measured by elapsed
// wall-clock. Use a tight bound so a CI runner with high jitter
// doesn't fail spuriously: lower bound 80% of configured.
func TestSettleAndMaybeShot_SettleViaRegistry(t *testing.T) {
	p := &fakeAfterActionPlat{}
	r := &Registry{plat: p, MouseSettleMs: 100}
	ctx := context.WithValue(context.Background(), registryKey{}, r)
	start := time.Now()
	settleAndMaybeShot(ctx, p, map[string]any{})
	elapsed := time.Since(start)
	if elapsed < 80*time.Millisecond {
		t.Errorf("settle did not happen: elapsed %v, expected >= 80ms", elapsed)
	}
}

// TestSettleAndMaybeShot_SettleCancelledByCtx: a cancelled ctx during
// the settle wait short-circuits without screenshotting.
func TestSettleAndMaybeShot_SettleCancelledByCtx(t *testing.T) {
	p := &fakeAfterActionPlat{}
	r := &Registry{plat: p, MouseSettleMs: 5000}
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), registryKey{}, r))
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	img, mime := settleAndMaybeShot(ctx, p, map[string]any{"return_screenshot": true})
	elapsed := time.Since(start)
	if elapsed > 200*time.Millisecond {
		t.Errorf("ctx cancel did not abort settle: elapsed %v (configured 5s, cancel at 20ms)", elapsed)
	}
	if img != "" || mime != "" {
		t.Errorf("cancelled ctx should produce empty result; got (%q, %q)", img, mime)
	}
	if p.shotCalled != 0 {
		t.Errorf("Screenshot called %d times after cancel; want 0", p.shotCalled)
	}
}

// TestSettleAndMaybeShot_BadParamGracefulFalse: a wrong-type
// return_screenshot is treated as false (silently swallowed by
// optionalBool's err) — the helper never propagates errors.
func TestSettleAndMaybeShot_BadParamGracefulFalse(t *testing.T) {
	p := &fakeAfterActionPlat{}
	img, _ := settleAndMaybeShot(context.Background(), p, map[string]any{"return_screenshot": 12345})
	if img != "" {
		t.Errorf("expected empty image for bad param; got %q", img)
	}
	_ = platform.ErrNotImplemented // keeps imports tidy if later code refactors
}
