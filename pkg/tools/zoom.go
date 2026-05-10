package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
	"golang.org/x/image/draw"
)

// Legacy aliases used by the schema description so it compiles
// without re-templating. Runtime caps come from zoomMaxFactorFor and
// zoomMaxOutputPixelsFor which honour the Registry override (DD-3).
const (
	zoomMaxFactor       = DefaultZoomMaxFactor
	zoomMaxOutputPixels = DefaultZoomMaxOutputPixels
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "zoom",
			Description: "Capture the active display, crop to `region`, and scale by " +
				"`factor` (e.g. 2.0 = 2x). Returns a PNG image. Useful when the " +
				"model needs to read small UI text that's illegible at full-screen " +
				"resolution. The region is in logical pixels; out-of-bounds regions " +
				"are clipped to the screenshot's bounds. Factor capped at 16x and " +
				"output capped at 16M pixels to prevent OOM (BUG-18).",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"region": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"x": map[string]any{"type": "integer", "description": "Top-left X (logical px)."},
							"y": map[string]any{"type": "integer", "description": "Top-left Y (logical px)."},
							"w": map[string]any{"type": "integer", "minimum": 1, "description": "Width in logical px."},
							"h": map[string]any{"type": "integer", "minimum": 1, "description": "Height in logical px."},
						},
						"required":             []string{"x", "y", "w", "h"},
						"additionalProperties": false,
						"description":          "Region of the active display to enlarge.",
					},
					"factor": map[string]any{
						"type":             "number",
						"exclusiveMinimum": 0,
						"maximum":          zoomMaxFactor,
						"description":      "Scale factor. >1 enlarges, <1 shrinks. 1.0 returns the cropped region at native size. Capped at 16x.",
					},
				},
				"required":             []string{"region", "factor"},
				"additionalProperties": false,
			},
			Handler: handleZoom,
		})
	})
}

func handleZoom(ctx context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	region, err := requireRect(params, "region")
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid region: %v", err), IsError: true}, nil
	}
	factor, err := requireFloat(params, "factor")
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid factor: %v", err), IsError: true}, nil
	}
	if factor <= 0 {
		return &Result{Text: fmt.Sprintf("invalid factor: %v (must be > 0)", factor), IsError: true}, nil
	}
	factorCap := zoomMaxFactorFor(ctx)
	if factor > factorCap {
		return &Result{
			Text:    fmt.Sprintf("invalid factor: %v exceeds max %v (BUG-18 cap to prevent OOM)", factor, factorCap),
			IsError: true,
		}, nil
	}
	img, err := plat.Screenshot()
	if err != nil {
		return &Result{Text: fmt.Sprintf("zoom: screenshot: %v", err), IsError: true}, nil
	}

	// Clip the requested region against the screenshot bounds. We
	// translate region into the image's coordinate space first
	// (Bounds().Min is not always (0,0) for sub-images).
	bounds := img.Bounds()
	rect := image.Rect(
		bounds.Min.X+region.X,
		bounds.Min.Y+region.Y,
		bounds.Min.X+region.X+region.W,
		bounds.Min.Y+region.Y+region.H,
	).Intersect(bounds)
	if rect.Empty() {
		return &Result{Text: fmt.Sprintf("zoom: region (%d,%d,%dx%d) is outside screenshot bounds %v",
			region.X, region.Y, region.W, region.H, bounds), IsError: true}, nil
	}

	// SubImage returns a view sharing the underlying pixel buffer when
	// the concrete type supports it, otherwise we copy. RGBA is the
	// common case from kbinani/screenshot.
	type subImager interface {
		SubImage(image.Rectangle) image.Image
	}
	var crop image.Image
	if si, ok := img.(subImager); ok {
		crop = si.SubImage(rect)
	} else {
		dst := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
		draw.Copy(dst, image.Point{}, img, rect, draw.Src, nil)
		crop = dst
	}

	// CatmullRom is the highest-quality general-purpose scaler in
	// x/image/draw — slower than ApproxBiLinear but the zoom tool is
	// called sparingly and the model wants legible text, not speed.
	scaledW := int(float64(rect.Dx())*factor + 0.5)
	scaledH := int(float64(rect.Dy())*factor + 0.5)
	if scaledW < 1 {
		scaledW = 1
	}
	if scaledH < 1 {
		scaledH = 1
	}
	// Absolute pixel cap as a second backstop (BUG-18). The factor cap
	// alone isn't enough — a 4K region at factor=4 is already 33MP.
	pixelCap := zoomMaxOutputPixelsFor(ctx)
	if int64(scaledW)*int64(scaledH) > int64(pixelCap) {
		return &Result{
			Text: fmt.Sprintf("zoom: output %dx%d (%.1fM pixels) exceeds %dM cap; reduce region or factor",
				scaledW, scaledH,
				float64(scaledW)*float64(scaledH)/1_000_000,
				pixelCap/1_000_000),
			IsError: true,
		}, nil
	}
	out := image.NewRGBA(image.Rect(0, 0, scaledW, scaledH))
	draw.CatmullRom.Scale(out, out.Bounds(), crop, crop.Bounds(), draw.Src, nil)

	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return &Result{Text: fmt.Sprintf("zoom: png encode: %v", err), IsError: true}, nil
	}
	return &Result{
		Text: fmt.Sprintf("zoomed region (%d,%d,%dx%d) by %gx → %dx%d PNG (%d bytes)",
			region.X, region.Y, region.W, region.H, factor, scaledW, scaledH, buf.Len()),
		Image:    base64.StdEncoding.EncodeToString(buf.Bytes()),
		MIMEType: "image/png",
	}, nil
}
