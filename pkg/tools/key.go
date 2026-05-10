package tools

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "key",
			Description: "Press a single key combo and release it. " +
				"Combo is a '+'-joined string where the final segment " +
				"is the key and any preceding segments are modifiers " +
				"(`cmd`, `ctrl`, `alt`, `shift`). Examples: `\"a\"`, " +
				"`\"cmd+shift+a\"`, `\"esc\"`, `\"f11\"`. Frontmost-app " +
				"tier `click` apps reject this — use `left_click` only.",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"combo": map[string]any{
						"type":        "string",
						"description": "Key combo, '+'-joined. Final segment is the key, preceding segments are modifiers.",
						"minLength":   1,
					},
				},
				"required":             []string{"combo"},
				"additionalProperties": false,
			},
			Handler: handleKey,
		})
	})
}

func handleKey(_ context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	combo, err := requireString(params, "combo")
	if err != nil {
		return &Result{Text: fmt.Sprintf("key: %v", err), IsError: true}, nil
	}
	if denied, deny := gateOrDeny(plat, "key"); deny {
		return denied, nil
	}
	if err := plat.KeyPress(combo); err != nil {
		return &Result{Text: fmt.Sprintf("key(%q): %v", combo, err), IsError: true}, nil
	}
	return &Result{Text: fmt.Sprintf("pressed: %s", combo)}, nil
}
