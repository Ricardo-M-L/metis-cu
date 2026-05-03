package tools

import (
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type imgPlat struct {
	stubPlat
	img image.Image
	err error
}

func (p *imgPlat) Screenshot() (image.Image, error) {
	return p.img, p.err
}

// makeTestImage builds a 100x100 RGBA with a checkerboard so the
// scaled output has visually distinct pixels (catches a no-op scaler).
func makeTestImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			if (x/10+y/10)%2 == 0 {
				img.Set(x, y, color.RGBA{255, 255, 255, 255})
			} else {
				img.Set(x, y, color.RGBA{0, 0, 0, 255})
			}
		}
	}
	return img
}

func TestZoom_OK(t *testing.T) {
	plat := &imgPlat{img: makeTestImage()}
	params := map[string]any{
		"region": map[string]any{
			"x": float64(10), "y": float64(10),
			"w": float64(20), "h": float64(20),
		},
		"factor": float64(2.0),
	}
	res, err := handleZoom(context.Background(), plat, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if res.MIMEType != "image/png" {
		t.Errorf("unexpected mime: %q", res.MIMEType)
	}
	raw, err := base64.StdEncoding.DecodeString(res.Image)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	out, err := png.Decode(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("png decode: %v", err)
	}
	if out.Bounds().Dx() != 40 || out.Bounds().Dy() != 40 {
		t.Errorf("scaled size = %v, want 40x40", out.Bounds())
	}
}

func TestZoom_RegionOutsideBounds(t *testing.T) {
	plat := &imgPlat{img: makeTestImage()}
	params := map[string]any{
		"region": map[string]any{
			"x": float64(500), "y": float64(500),
			"w": float64(10), "h": float64(10),
		},
		"factor": float64(1.0),
	}
	res, _ := handleZoom(context.Background(), plat, params)
	if !res.IsError {
		t.Fatal("expected IsError for out-of-bounds region")
	}
}

func TestZoom_InvalidFactor(t *testing.T) {
	plat := &imgPlat{img: makeTestImage()}
	params := map[string]any{
		"region": map[string]any{
			"x": float64(0), "y": float64(0),
			"w": float64(10), "h": float64(10),
		},
		"factor": float64(0),
	}
	res, _ := handleZoom(context.Background(), plat, params)
	if !res.IsError {
		t.Fatal("expected IsError for factor=0")
	}
}

func TestZoom_PlatformScreenshotError(t *testing.T) {
	plat := &imgPlat{err: platform.ErrNotImplemented}
	params := map[string]any{
		"region": map[string]any{
			"x": float64(0), "y": float64(0),
			"w": float64(10), "h": float64(10),
		},
		"factor": float64(2.0),
	}
	res, _ := handleZoom(context.Background(), plat, params)
	if !res.IsError {
		t.Fatal("expected IsError when platform screenshot fails")
	}
}
