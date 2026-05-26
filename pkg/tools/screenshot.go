package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
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

	format, quality := screenshotFormat(ctx)
	buf, mime, err := encodeScreenshot(img, format, quality)
	if err != nil {
		return &Result{Text: fmt.Sprintf("%s encode: %v", format, err), IsError: true}, nil
	}
	tag := "PNG"
	if format == "jpeg" {
		tag = fmt.Sprintf("JPEG q=%d", quality)
	}
	var summary string
	if final.Dx() == orig.Dx() && final.Dy() == orig.Dy() {
		summary = fmt.Sprintf("captured %dx%d %s (%d bytes)", orig.Dx(), orig.Dy(), tag, buf.Len())
	} else {
		summary = fmt.Sprintf("captured %dx%d → downsampled %dx%d %s (%d bytes, cap %dx%d)",
			orig.Dx(), orig.Dy(), final.Dx(), final.Dy(), tag, buf.Len(), maxW, maxH)
	}
	return &Result{
		Text:     summary,
		Image:    base64.StdEncoding.EncodeToString(buf.Bytes()),
		MIMEType: mime,
	}, nil
}

// screenshotFormat returns (format, quality) for the active call, in
// that order. Pulls from the Registry on ctx (DD-3) and falls back to
// "jpeg"/85 (changed from "png"/85 on 2026-05-26 — see
// ScreenshotConfig docstring for context: PNG payloads forced metis's
// context-overflow snipper on every cu screenshot). Quality is only
// used when format is "jpeg".
func screenshotFormat(ctx context.Context) (string, int) {
	if reg, ok := ctx.Value(registryKey{}).(*Registry); ok && reg != nil {
		f := reg.ScreenshotFormat
		q := reg.ScreenshotJPEGQ
		if f == "" {
			f = "jpeg"
		}
		if q < 1 || q > 100 {
			q = 85
		}
		return f, q
	}
	return "jpeg", 85
}

// encodeScreenshot picks the encoder by format. JPEG flattens alpha
// onto a white background first (PNG-with-alpha → JPEG would otherwise
// turn transparent pixels black, which is jarring for any UI element
// that uses alpha for shadows / rounded corners). Returns the encoded
// buffer and MIME type.
func encodeScreenshot(img image.Image, format string, quality int) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	switch format {
	case "jpeg":
		flat := flattenAlphaOnWhite(img)
		if err := jpeg.Encode(&buf, flat, &jpeg.Options{Quality: quality}); err != nil {
			return nil, "", err
		}
		return &buf, "image/jpeg", nil
	default:
		if err := png.Encode(&buf, img); err != nil {
			return nil, "", err
		}
		return &buf, "image/png", nil
	}
}

// flattenAlphaOnWhite composites src over an opaque white background.
// Cheap pre-step before JPEG encode so transparent regions render as
// white (the user-expected colour for typical UI screenshots) rather
// than the black JPEG would produce.
func flattenAlphaOnWhite(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	white := image.NewUniform(color.White)
	xdraw.Copy(dst, b.Min, white, b, xdraw.Src, nil)
	xdraw.Copy(dst, b.Min, src, b, xdraw.Over, nil)
	return dst
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
