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
			"description": "Modifier keys to hold during the click. NOT yet wired in Phase 2-A — passing a non-empty array returns an error. Will be supported in a follow-up phase.",
		}
		r.register(Spec{
			Name: "left_click",
			Description: "Single left-click at (x, y). The cursor is moved to the target first. " +
				"Modifier keys are accepted in the schema for forward-compat but rejected at " +
				"runtime in Phase 2-A.",
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

func handleLeftClick(_ context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	pt, err := requireXY(params)
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid params: %v", err), IsError: true}, nil
	}
	if rejected, msg := rejectModifiers(params); rejected {
		return &Result{Text: msg, IsError: true}, nil
	}
	if err := plat.MouseClick(pt, platform.ButtonLeft, 1); err != nil {
		return &Result{Text: fmt.Sprintf("left_click(%d, %d): %v", pt.X, pt.Y, err), IsError: true}, nil
	}
	return &Result{Text: fmt.Sprintf("left-clicked at (%d, %d)", pt.X, pt.Y)}, nil
}
