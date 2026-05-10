package tools

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		props := xySchema()
		props["modifiers"] = map[string]any{
			"type":        "array",
			"items":       map[string]any{"type": "string", "enum": []string{"cmd", "ctrl", "alt", "shift"}},
			"description": "Modifier keys to hold during the click. \"cmd\" maps to Ctrl on Linux/Windows. Released automatically even if the click errors.",
		}
		props["return_screenshot"] = returnScreenshotSchema()
		r.register(Spec{
			Name: "left_click",
			Description: "Single left-click at (x, y). The cursor is moved to the target first. " +
				"Optional `modifiers` (cmd, ctrl, alt, shift) are pressed before the click and " +
				"released after — \"cmd\" is translated to \"ctrl\" on Linux/Windows so models " +
				"trained on macOS conventions get the expected semantics everywhere. " +
				"Set `return_screenshot=true` to receive the post-click screenshot inline.",
			Schema: map[string]any{
				"type":                 "object",
				"properties":           props,
				"required":             []string{"x", "y"},
				"additionalProperties": false,
			},
			Handler: handleLeftClick,
		})
	})
}

func handleLeftClick(ctx context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	pt, err := requireXY(params)
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid params: %v", err), IsError: true}, nil
	}
	mods, err := optionalModifiers(params)
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid modifiers: %v", err), IsError: true}, nil
	}
	if denied, deny := gateOrDeny(plat, "left_click"); deny {
		return denied, nil
	}
	if len(mods) == 0 {
		if err := plat.MouseClick(pt, platform.ButtonLeft, 1); err != nil {
			return &Result{Text: fmt.Sprintf("left_click(%d, %d): %v", pt.X, pt.Y, err), IsError: true}, nil
		}
		img, mime := settleAndMaybeShot(ctx, plat, params)
		return &Result{Text: fmt.Sprintf("left-clicked at (%d, %d)", pt.X, pt.Y), Image: img, MIMEType: mime}, nil
	}
	if err := plat.MouseClickWithModifiers(pt, platform.ButtonLeft, 1, mods); err != nil {
		return &Result{Text: fmt.Sprintf("left_click(%d, %d, mods=%v): %v", pt.X, pt.Y, mods, err), IsError: true}, nil
	}
	img, mime := settleAndMaybeShot(ctx, plat, params)
	return &Result{Text: fmt.Sprintf("left-clicked at (%d, %d) with modifiers %v", pt.X, pt.Y, mods), Image: img, MIMEType: mime}, nil
}
