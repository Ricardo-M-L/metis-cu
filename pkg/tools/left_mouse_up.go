package tools

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		props := xySchema()
		props["return_screenshot"] = returnScreenshotSchema()
		r.register(Spec{
			Name: "left_mouse_up",
			Description: "Release the left mouse button at (x, y). The cursor is moved first so " +
				"callers driving a manual drag with `left_mouse_down` can pick the precise " +
				"release point without an extra `mouse_move` call in between.",
			Schema: map[string]any{
				"type":                 "object",
				"properties":           props,
				"required":             []string{"x", "y"},
				"additionalProperties": false,
			},
			Handler: handleLeftMouseUp,
		})
	})
}

func handleLeftMouseUp(ctx context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	pt, err := requireXY(params)
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid params: %v", err), IsError: true}, nil
	}
	if denied, deny := gateOrDeny(plat, "left_mouse_up"); deny {
		return denied, nil
	}
	if err := plat.MouseUp(pt, platform.ButtonLeft); err != nil {
		return &Result{Text: fmt.Sprintf("left_mouse_up(%d, %d): %v", pt.X, pt.Y, err), IsError: true}, nil
	}
	img, mime := settleAndMaybeShot(ctx, plat, params)
	return &Result{Text: fmt.Sprintf("left mouse up at (%d, %d)", pt.X, pt.Y), Image: img, MIMEType: mime}, nil
}
