package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

// fakeScreenshotPlat returns a synthetic RGBA whose dimensions can be
// dialled up to simulate a Retina capture without depending on the
// user's actual display.
type fakeScreenshotPlat struct {
	stubPlat
	w, h int
}

func (p *fakeScreenshotPlat) Screenshot() (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, p.w, p.h))
	// Paint a recognisable diagonal so the encoded PNG isn't a flat
	// black canvas (lets us sanity-check the round-trip if needed).
	for x := 0; x < p.w; x++ {
		img.Set(x, x*p.h/p.w, color.RGBA{R: 200, G: 200, B: 200, A: 255})
	}
	return img, nil
}

func (p *fakeScreenshotPlat) SwitchDisplay(int) error { return nil }

// TestScreenshot_DownsamplesToDefault: a 4000x2500 capture should be
// shrunk under the default 1280x800 cap with aspect ratio preserved.
func TestScreenshot_DownsamplesToDefault(t *testing.T) {
	p := &fakeScreenshotPlat{w: 4000, h: 2500}
	res, err := handleScreenshot(context.Background(), p, nil)
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if !strings.Contains(res.Text, "downsampled") {
		t.Errorf("expected downsample summary, got: %s", res.Text)
	}
	if !strings.Contains(res.Text, "4000x2500") {
		t.Errorf("expected original size in summary, got: %s", res.Text)
	}
	got := decodePNG(t, res.Image)
	gw, gh := got.Bounds().Dx(), got.Bounds().Dy()
	if gw > DefaultScreenshotMaxW || gh > DefaultScreenshotMaxH {
		t.Errorf("downsampled %dx%d still exceeds default %dx%d", gw, gh, DefaultScreenshotMaxW, DefaultScreenshotMaxH)
	}
	// Aspect ratio preserved within 1px rounding.
	wantRatio := 4000.0 / 2500.0
	gotRatio := float64(gw) / float64(gh)
	if delta := wantRatio - gotRatio; delta < -0.01 || delta > 0.01 {
		t.Errorf("aspect ratio drifted: want ~%.3f got %.3f (size %dx%d)", wantRatio, gotRatio, gw, gh)
	}
}

// TestScreenshot_NoDownsampleWhenSmall: an image already inside the
// cap should pass through unchanged so we don't waste CPU and don't
// confuse the user with a "downsampled to identical size" line.
func TestScreenshot_NoDownsampleWhenSmall(t *testing.T) {
	p := &fakeScreenshotPlat{w: 800, h: 600}
	res, err := handleScreenshot(context.Background(), p, nil)
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if strings.Contains(res.Text, "downsampled") {
		t.Errorf("did not expect downsample for 800x600 within 1280x800 cap; got: %s", res.Text)
	}
	got := decodePNG(t, res.Image)
	if got.Bounds().Dx() != 800 || got.Bounds().Dy() != 600 {
		t.Errorf("expected 800x600 passthrough, got %dx%d", got.Bounds().Dx(), got.Bounds().Dy())
	}
}

// TestScreenshot_UsesRegistryLimits: a Registry-bound limit should
// override the package default. Confirms the ctx-plumbing path used by
// the live MCP server.
func TestScreenshot_UsesRegistryLimits(t *testing.T) {
	p := &fakeScreenshotPlat{w: 2000, h: 1500}
	r := &Registry{
		plat:           p,
		ScreenshotMaxW: 640,
		ScreenshotMaxH: 480,
	}
	ctx := context.WithValue(context.Background(), registryKey{}, r)
	res, err := handleScreenshot(ctx, p, nil)
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if !strings.Contains(res.Text, "cap 640x480") {
		t.Errorf("expected registry-limit cap in summary, got: %s", res.Text)
	}
	got := decodePNG(t, res.Image)
	if got.Bounds().Dx() > 640 || got.Bounds().Dy() > 480 {
		t.Errorf("downsampled %dx%d exceeds registry cap 640x480", got.Bounds().Dx(), got.Bounds().Dy())
	}
}

// TestDownsample_PreservesAspect verifies the helper directly across
// landscape / portrait / square inputs.
func TestDownsample_PreservesAspect(t *testing.T) {
	cases := []struct {
		name             string
		sw, sh           int
		maxW, maxH       int
		wantW, wantH     int
		passthroughExact bool
	}{
		{"landscape-shrinks", 4000, 2500, 1280, 800, 1280, 800, false},
		{"portrait-shrinks", 2000, 4000, 1280, 800, 400, 800, false},
		{"square-shrinks", 3000, 3000, 1280, 800, 800, 800, false},
		{"already-fits", 800, 600, 1280, 800, 800, 600, true},
		{"exact-fit", 1280, 800, 1280, 800, 1280, 800, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := image.NewRGBA(image.Rect(0, 0, tc.sw, tc.sh))
			got := downsampleScreenshot(src, tc.maxW, tc.maxH)
			gw, gh := got.Bounds().Dx(), got.Bounds().Dy()
			if gw != tc.wantW || gh != tc.wantH {
				t.Errorf("downsample(%dx%d → cap %dx%d) = %dx%d, want %dx%d",
					tc.sw, tc.sh, tc.maxW, tc.maxH, gw, gh, tc.wantW, tc.wantH)
			}
			// Passthrough cases should return the SAME pointer (no
			// allocation) — catches future refactors that drop the
			// no-op fast path.
			if tc.passthroughExact && got != image.Image(src) {
				t.Errorf("expected passthrough to return src unchanged")
			}
		})
	}
}

func decodePNG(t *testing.T, b64 string) image.Image {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("png decode: %v", err)
	}
	return img
}

// TestScreenshot_JPEGFormat: Registry config switches the encoder to
// JPEG and the resulting bytes decode as JPEG (not PNG). MIME type and
// Result.Text "JPEG q=N" tag also change. Locks in the Tier-1 borrow
// from SoC's compress_screenshot path.
func TestScreenshot_JPEGFormat(t *testing.T) {
	p := &fakeScreenshotPlat{w: 800, h: 600}
	r := &Registry{
		plat:             p,
		ScreenshotMaxW:   1280,
		ScreenshotMaxH:   800,
		ScreenshotFormat: "jpeg",
		ScreenshotJPEGQ:  85,
	}
	ctx := context.WithValue(context.Background(), registryKey{}, r)
	res, err := handleScreenshot(ctx, p, nil)
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if res.MIMEType != "image/jpeg" {
		t.Errorf("MIMEType = %q, want image/jpeg", res.MIMEType)
	}
	if !strings.Contains(res.Text, "JPEG q=85") {
		t.Errorf("expected JPEG q=85 tag in summary, got: %s", res.Text)
	}
	raw, err := base64.StdEncoding.DecodeString(res.Image)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	img, err := jpeg.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("jpeg decode: %v\nfirst bytes: %x", err, raw[:min(8, len(raw))])
	}
	if img.Bounds().Dx() != 800 || img.Bounds().Dy() != 600 {
		t.Errorf("decoded JPEG dim = %dx%d, want 800x600", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

// TestScreenshot_DefaultsToPNG: with no Registry override the encoder
// stays at PNG (lossless default — sharp edges for vision/OCR models).
func TestScreenshot_DefaultsToPNG(t *testing.T) {
	p := &fakeScreenshotPlat{w: 200, h: 100}
	res, err := handleScreenshot(context.Background(), p, nil)
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if res.MIMEType != "image/png" {
		t.Errorf("MIMEType = %q, want image/png", res.MIMEType)
	}
	if !strings.Contains(res.Text, "PNG") || strings.Contains(res.Text, "JPEG") {
		t.Errorf("expected PNG-only summary, got: %s", res.Text)
	}
}
