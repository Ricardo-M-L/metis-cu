package tools

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "left_click_drag",
			Description: "Press the left button at `from`, drag smoothly to `to`, and release. " +
				"Use for canvas selections, draggable list reorders, slider scrubs, and the " +
				"like — apps that distinguish drag from teleport see this as a real drag.",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"from":              pointSchema("Drag start point — where the mouse-down event fires."),
					"to":                pointSchema("Drag end point — where the mouse-up event fires after the smooth motion completes."),
					"return_screenshot": returnScreenshotSchema(),
				},
				"required":             []string{"from", "to"},
				"additionalProperties": false,
			},
			Handler: handleLeftClickDrag,
		})
	})
}

func handleLeftClickDrag(ctx context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	from, err := requirePoint(params, "from")
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid params: %v", err), IsError: true}, nil
	}
	to, err := requirePoint(params, "to")
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid params: %v", err), IsError: true}, nil
	}
	if denied, deny := gateOrDeny(plat, "left_click_drag"); deny {
		return denied, nil
	}
	if err := plat.MouseDrag(ctx, from, to, platform.ButtonLeft); err != nil {
		return &Result{
			Text:    fmt.Sprintf("left_click_drag((%d,%d) → (%d,%d)): %v", from.X, from.Y, to.X, to.Y, err),
			IsError: true,
		}, nil
	}
	img, mime := settleAndMaybeShot(ctx, plat, params)
	return &Result{Text: fmt.Sprintf("dragged from (%d, %d) to (%d, %d)", from.X, from.Y, to.X, to.Y), Image: img, MIMEType: mime}, nil
}
