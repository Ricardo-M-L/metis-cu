package tools

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "cursor_position",
			Description: "Return the current mouse cursor location in logical pixels " +
				"(top-left origin, matches `screenshot` coordinates). No tier gate — " +
				"reading the cursor position is a vision operation, not input.",
			Schema: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
			Handler: handleCursorPosition,
		})
	})
}

func handleCursorPosition(_ context.Context, plat platform.Platform, _ map[string]any) (*Result, error) {
	pt, err := plat.CursorPosition()
	if err != nil {
		return &Result{Text: fmt.Sprintf("cursor_position: %v", err), IsError: true}, nil
	}
	// Formatted as JSON-ish text so the LLM gets a clean structured
	// payload — mcp-go does not have a native "structured object" result
	// shape, so adapters that return small key/value data emit it as
	// text the model can parse.
	return &Result{Text: fmt.Sprintf(`{"x":%d,"y":%d}`, pt.X, pt.Y)}, nil
}
