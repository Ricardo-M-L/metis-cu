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
			Name: "triple_click",
			Description: "Triple left-click at (x, y) — selects an entire line/paragraph in most " +
				"text widgets. Cursor is moved first. The three clicks are paced within the " +
				"OS multi-click window so apps recognize them as a single triple-click.",
			Schema: map[string]any{
				"type":                 "object",
				"properties":           props,
				"required":             []string{"x", "y"},
				"additionalProperties": false,
			},
			Handler: handleTripleClick,
		})
	})
}

func handleTripleClick(ctx context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	pt, err := requireXY(params)
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid params: %v", err), IsError: true}, nil
	}
	if denied, deny := gateOrDeny(plat, "triple_click"); deny {
		return denied, nil
	}
	if err := plat.MouseClick(pt, platform.ButtonLeft, 3); err != nil {
		return &Result{Text: fmt.Sprintf("triple_click(%d, %d): %v", pt.X, pt.Y, err), IsError: true}, nil
	}
	img, mime := settleAndMaybeShot(ctx, plat, params)
	return &Result{Text: fmt.Sprintf("triple-clicked at (%d, %d)", pt.X, pt.Y), Image: img, MIMEType: mime}, nil
}
