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
			Name: "double_click",
			Description: "Double left-click at (x, y) — typically opens an item or selects a word. " +
				"Cursor is moved first. The two clicks land within the OS's double-click " +
				"interval so apps recognize them as a single double-click event, not two singles.",
			Schema: map[string]any{
				"type":                 "object",
				"properties":           props,
				"required":             []string{"x", "y"},
				"additionalProperties": false,
			},
			Handler: handleDoubleClick,
		})
	})
}

func handleDoubleClick(ctx context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	pt, err := requireXY(params)
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid params: %v", err), IsError: true}, nil
	}
	if denied, deny := gateOrDeny(plat, "double_click"); deny {
		return denied, nil
	}
	if err := plat.MouseClick(pt, platform.ButtonLeft, 2); err != nil {
		return &Result{Text: fmt.Sprintf("double_click(%d, %d): %v", pt.X, pt.Y, err), IsError: true}, nil
	}
	img, mime := settleAndMaybeShot(ctx, plat, params)
	return &Result{Text: fmt.Sprintf("double-clicked at (%d, %d)", pt.X, pt.Y), Image: img, MIMEType: mime}, nil
}
