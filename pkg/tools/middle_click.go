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
			Name: "middle_click",
			Description: "Single middle-click (scroll-wheel button) at (x, y). " +
				"Cursor is moved first. Useful for opening links in a new background tab " +
				"in browsers and for paste-on-X11 — though X11 paste isn't relevant on macOS.",
			Schema: map[string]any{
				"type":                 "object",
				"properties":           props,
				"required":             []string{"x", "y"},
				"additionalProperties": false,
			},
			Handler: handleMiddleClick,
		})
	})
}

func handleMiddleClick(ctx context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	pt, err := requireXY(params)
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid params: %v", err), IsError: true}, nil
	}
	if denied, deny := gateOrDeny(plat, "middle_click"); deny {
		return denied, nil
	}
	if err := plat.MouseClick(pt, platform.ButtonMiddle, 1); err != nil {
		return &Result{Text: fmt.Sprintf("middle_click(%d, %d): %v", pt.X, pt.Y, err), IsError: true}, nil
	}
	img, mime := settleAndMaybeShot(ctx, plat, params)
	return &Result{Text: fmt.Sprintf("middle-clicked at (%d, %d)", pt.X, pt.Y), Image: img, MIMEType: mime}, nil
}
