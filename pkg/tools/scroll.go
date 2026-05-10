package tools

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		props := xySchema()
		props["dx"] = map[string]any{
			"type":        "integer",
			"description": "Horizontal wheel ticks. Positive = right.",
		}
		props["dy"] = map[string]any{
			"type":        "integer",
			"description": "Vertical wheel ticks. Positive = down (matches Anthropic's spec; OS natural-scroll setting is bypassed — these are raw wheel events).",
		}
		props["modifiers"] = map[string]any{
			"type":        "array",
			"items":       map[string]any{"type": "string", "enum": []string{"cmd", "ctrl", "alt", "shift"}},
			"description": "Optional modifiers to hold while scrolling: ctrl+wheel = zoom in most apps, shift+wheel = horizontal scroll, alt+wheel = step-by-pixel in some image editors. `cmd` is auto-translated to `ctrl` on Linux/Windows.",
		}
		props["return_screenshot"] = returnScreenshotSchema()
		r.register(Spec{
			Name: "scroll",
			Description: "Move to (x, y) and emit (dx, dy) wheel ticks. Positive dy scrolls down. " +
				"One tick is one notch on a physical scroll wheel — for inertial / momentum " +
				"scrolling emit a stream of small ticks rather than one giant one. " +
				"Optional `modifiers` array holds ctrl/shift/alt/cmd while scrolling " +
				"(ctrl+wheel = zoom, shift+wheel = horizontal).",
			Schema: map[string]any{
				"type":                 "object",
				"properties":           props,
				"required":             []string{"x", "y", "dx", "dy"},
				"additionalProperties": false,
			},
			Handler: handleScroll,
		})
	})
}

func handleScroll(ctx context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	pt, err := requireXY(params)
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid params: %v", err), IsError: true}, nil
	}
	dx, err := requireInt(params, "dx")
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid dx: %v", err), IsError: true}, nil
	}
	dy, err := requireInt(params, "dy")
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid dy: %v", err), IsError: true}, nil
	}
	mods, err := optionalModifiers(params)
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid modifiers: %v", err), IsError: true}, nil
	}
	if denied, deny := gateOrDeny(plat, "scroll"); deny {
		return denied, nil
	}
	if len(mods) > 0 {
		if err := plat.ScrollWithModifiers(pt, dx, dy, mods); err != nil {
			return &Result{Text: fmt.Sprintf("scroll(%d, %d, dx=%d, dy=%d, mods=%v): %v", pt.X, pt.Y, dx, dy, mods, err), IsError: true}, nil
		}
		img, mime := settleAndMaybeShot(ctx, plat, params)
		return &Result{Text: fmt.Sprintf("scrolled at (%d, %d) by (dx=%d, dy=%d) holding %v", pt.X, pt.Y, dx, dy, mods), Image: img, MIMEType: mime}, nil
	}
	if err := plat.Scroll(pt, dx, dy); err != nil {
		return &Result{Text: fmt.Sprintf("scroll(%d, %d, dx=%d, dy=%d): %v", pt.X, pt.Y, dx, dy, err), IsError: true}, nil
	}
	img, mime := settleAndMaybeShot(ctx, plat, params)
	return &Result{Text: fmt.Sprintf("scrolled at (%d, %d) by (dx=%d, dy=%d)", pt.X, pt.Y, dx, dy), Image: img, MIMEType: mime}, nil
}
