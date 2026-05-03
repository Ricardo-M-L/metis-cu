package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
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

func handleScreenshot(_ context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
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
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return &Result{Text: fmt.Sprintf("png encode: %v", err), IsError: true}, nil
	}
	bounds := img.Bounds()
	return &Result{
		Text:     fmt.Sprintf("captured %dx%d PNG (%d bytes)", bounds.Dx(), bounds.Dy(), buf.Len()),
		Image:    base64.StdEncoding.EncodeToString(buf.Bytes()),
		MIMEType: "image/png",
	}, nil
}
