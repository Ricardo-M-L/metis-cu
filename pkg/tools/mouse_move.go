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
			Name: "mouse_move",
			Description: "Move the cursor to (x, y) absolute screen coordinates. " +
				"Coordinates are in logical pixels with the top-left as the origin. " +
				"This is a teleport — for human-like motion use `left_click_drag`.",
			Schema: map[string]any{
				"type":                 "object",
				"properties":           props,
				"required":             []string{"x", "y"},
				"additionalProperties": false,
			},
			Handler: handleMouseMove,
		})
	})
}

func handleMouseMove(ctx context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	pt, err := requireXY(params)
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid params: %v", err), IsError: true}, nil
	}
	if denied, deny := gateOrDeny(plat, "mouse_move"); deny {
		return denied, nil
	}
	if err := plat.MouseMove(pt); err != nil {
		return &Result{Text: fmt.Sprintf("mouse_move(%d, %d): %v", pt.X, pt.Y, err), IsError: true}, nil
	}
	img, mime := settleAndMaybeShot(ctx, plat, params)
	return &Result{Text: fmt.Sprintf("moved cursor to (%d, %d)", pt.X, pt.Y), Image: img, MIMEType: mime}, nil
}
