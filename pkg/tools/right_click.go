package tools

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "right_click",
			Description: "Single right-click at (x, y) — opens the context menu in most apps. " +
				"Cursor is moved first. Tier 'click' apps reject this; the gate runs before " +
				"the handler so a denied call returns a tier error.",
			Schema: map[string]any{
				"type":                 "object",
				"properties":           xySchema(),
				"required":             []string{"x", "y"},
				"additionalProperties": false,
			},
			Handler: handleRightClick,
		})
	})
}

func handleRightClick(_ context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	pt, err := requireXY(params)
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid params: %v", err), IsError: true}, nil
	}
	if denied, deny := gateOrDeny(plat, "right_click"); deny {
		return denied, nil
	}
	if err := plat.MouseClick(pt, platform.ButtonRight, 1); err != nil {
		return &Result{Text: fmt.Sprintf("right_click(%d, %d): %v", pt.X, pt.Y, err), IsError: true}, nil
	}
	return &Result{Text: fmt.Sprintf("right-clicked at (%d, %d)", pt.X, pt.Y)}, nil
}
