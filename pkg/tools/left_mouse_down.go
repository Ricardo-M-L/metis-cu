package tools

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "left_mouse_down",
			Description: "Press and hold the left mouse button at (x, y) without releasing. " +
				"Pair with `left_mouse_up` to compose a custom drag — for the common case " +
				"prefer `left_click_drag`, which atomically moves + presses + drags + releases.",
			Schema: map[string]any{
				"type":                 "object",
				"properties":           xySchema(),
				"required":             []string{"x", "y"},
				"additionalProperties": false,
			},
			Handler: handleLeftMouseDown,
		})
	})
}

func handleLeftMouseDown(_ context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	pt, err := requireXY(params)
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid params: %v", err), IsError: true}, nil
	}
	if denied, deny := gateOrDeny(plat, "left_mouse_down"); deny {
		return denied, nil
	}
	if err := plat.MouseDown(pt, platform.ButtonLeft); err != nil {
		return &Result{Text: fmt.Sprintf("left_mouse_down(%d, %d): %v", pt.X, pt.Y, err), IsError: true}, nil
	}
	return &Result{Text: fmt.Sprintf("left mouse down at (%d, %d)", pt.X, pt.Y)}, nil
}
