package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
	xdraw "golang.org/x/image/draw"
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "screenshot",
			Description: "Capture the current active display as a PNG image. " +
				"If `display` is provided, switch to that display index first " +
				"(0-based; use `switch_display` for a sticky switch).",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"display": map[string]any{
						"type":        "integer",
						"minimum":     0,
						"description": "Optional 0-based display index to capture. Omit to use the active display.",
					},
				},
				"additionalProperties": false,
			},
			Handler: handleScreenshot,
		})
	})
}

func handleScreenshot(ctx context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	if raw, ok := params["display"]; ok {
		idx, err := asInt(raw)
		if err != nil {
			return &Result{Text: fmt.Sprintf("invalid display: %v", err), IsError: true}, nil
		}
		if err := plat.SwitchDisplay(idx); err != nil {
			return &Result{Text: fmt.Sprintf("switch_display(%d): %v", idx, err), IsError: true}, nil
		}
	}
	img, err := plat.Screenshot()
	if err != nil {
		return &Result{Text: fmt.Sprintf("screenshot: %v", err), IsError: true}, nil
	}

	orig := img.Bounds()
	maxW, maxH := screenshotLimits(ctx)
	img = downsampleScreenshot(img, maxW, maxH)
	final := img.Bounds()

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return &Result{Text: fmt.Sprintf("png encode: %v", err), IsError: true}, nil
	}
	var summary string
	if final.Dx() == orig.Dx() && final.Dy() == orig.Dy() {
		summary = fmt.Sprintf("captured %dx%d PNG (%d bytes)", orig.Dx(), orig.Dy(), buf.Len())
	} else {
		summary = fmt.Sprintf("captured %dx%d → downsampled %dx%d PNG (%d bytes, cap %dx%d)",
			orig.Dx(), orig.Dy(), final.Dx(), final.Dy(), buf.Len(), maxW, maxH)
	}
	return &Result{
		Text:     summary,
		Image:    base64.StdEncoding.EncodeToString(buf.Bytes()),
		MIMEType: "image/png",
	}, nil
}

// screenshotLimits returns the per-screenshot pixel cap. Reads the
// active Registry off the context (installed by Registry.Call) and
// falls back to the package-level defaults when called outside the
// MCP dispatch path (direct handler tests).
func screenshotLimits(ctx context.Context) (int, int) {
	if reg, ok := ctx.Value(registryKey{}).(*Registry); ok && reg != nil {
		w, h := reg.ScreenshotMaxW, reg.ScreenshotMaxH
		if w > 0 && h > 0 {
			return w, h
		}
	}
	return DefaultScreenshotMaxW, DefaultScreenshotMaxH
}

// downsampleScreenshot scales src to fit inside maxW × maxH preserving
// aspect ratio. Returns src unchanged when it already fits — avoids a
// round-trip through CatmullRom that would lose ~1-2% sharpness for no
// gain. CatmullRom is the slowest of the x/image/draw scalers but the
// sharpest; LLM vision models care about edge fidelity, and a
// once-per-screenshot allocation is well within budget.
func downsampleScreenshot(src image.Image, maxW, maxH int) image.Image {
	if maxW <= 0 || maxH <= 0 {
		return src
	}
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw <= maxW && sh <= maxH {
		return src
	}
	rw := float64(maxW) / float64(sw)
	rh := float64(maxH) / float64(sh)
	r := rw
	if rh < r {
		r = rh
	}
	dw := int(float64(sw) * r)
	dh := int(float64(sh) * r)
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	return dst
}
