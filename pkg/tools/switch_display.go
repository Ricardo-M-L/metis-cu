package tools

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "switch_display",
			Description: "Set the active display index for subsequent `screenshot` / " +
				"`zoom` calls. 0-based; out-of-range index returns an error. Sticky " +
				"until called again — `screenshot` accepts a one-shot `display` " +
				"override that does not modify the active selection.",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"display": map[string]any{
						"type":        "integer",
						"minimum":     0,
						"description": "0-based display index. Use `screenshot` afterwards to capture this display.",
					},
				},
				"required":             []string{"display"},
				"additionalProperties": false,
			},
			Handler: handleSwitchDisplay,
		})
	})
}

func handleSwitchDisplay(_ context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	idx, err := requireInt(params, "display")
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid display: %v", err), IsError: true}, nil
	}
	if err := plat.SwitchDisplay(idx); err != nil {
		return &Result{Text: fmt.Sprintf("switch_display(%d): %v", idx, err), IsError: true}, nil
	}
	return &Result{Text: fmt.Sprintf(`{"ok":true,"display":%d}`, idx)}, nil
}
