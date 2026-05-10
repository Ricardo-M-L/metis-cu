package tools

import (
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// requirePoint extracts a {x:int, y:int} sub-object from params under
// `key`. left_click_drag uses this for both `from` and `to`. Failures
// propagate as plain errors so the calling adapter wraps them in the
// usual "invalid <field>: ..." Result.
func requirePoint(params map[string]any, key string) (platform.Point, error) {
	v, ok := params[key]
	if !ok {
		return platform.Point{}, fmt.Errorf("missing required field: %s", key)
	}
	m, ok := v.(map[string]any)
	if !ok {
		return platform.Point{}, fmt.Errorf("%s: expected object, got %T", key, v)
	}
	x, err := requireInt(m, "x")
	if err != nil {
		return platform.Point{}, fmt.Errorf("%s.x: %w", key, err)
	}
	y, err := requireInt(m, "y")
	if err != nil {
		return platform.Point{}, fmt.Errorf("%s.y: %w", key, err)
	}
	return platform.Point{X: x, Y: y}, nil
}

// requireXY extracts the common (x, y) integer pair from a flat
// params object — the shape used by every mouse tool except
// left_click_drag.
func requireXY(params map[string]any) (platform.Point, error) {
	x, err := requireInt(params, "x")
	if err != nil {
		return platform.Point{}, err
	}
	y, err := requireInt(params, "y")
	if err != nil {
		return platform.Point{}, err
	}
	return platform.Point{X: x, Y: y}, nil
}

// pointSchema is the JSON Schema fragment for a {x:int, y:int} pair.
// Reused by left_click_drag's from / to declarations so both endpoints
// stay in sync.
func pointSchema(desc string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"x": map[string]any{"type": "integer", "description": "X coordinate (logical pixels, top-left origin)."},
			"y": map[string]any{"type": "integer", "description": "Y coordinate (logical pixels, top-left origin)."},
		},
		"required":             []string{"x", "y"},
		"additionalProperties": false,
		"description":          desc,
	}
}

// xySchema returns the property map shared by every flat-(x,y) tool.
// Tools spread the returned map into their own `properties` block —
// using a helper keeps the schemas consistent (same descriptions,
// same minimum/integer constraints) without per-tool copy-paste.
func xySchema() map[string]any {
	return map[string]any{
		"x": map[string]any{
			"type":        "integer",
			"description": "X coordinate (logical pixels, top-left origin).",
		},
		"y": map[string]any{
			"type":        "integer",
			"description": "Y coordinate (logical pixels, top-left origin).",
		},
	}
}

// optionalModifiers parses the schema-declared `modifiers` array into
// a string slice the platform layer can consume. Missing key returns
// (nil, nil); empty array returns (nil, nil) — both treated as "no
// modifiers". Validates each entry against the documented enum so a
// hallucinated "ctrlx" surfaces as a clear error rather than getting
// passed to robotgo (which would silently drop unknown names).
//
// Replaces the old rejectModifiers (BUG-21): now that the platform
// supports MouseClickWithModifiers, the tool layer just hands them
// through after validation.
func optionalModifiers(params map[string]any) ([]string, error) {
	v, ok := params["modifiers"]
	if !ok {
		return nil, nil
	}
	mods, err := asStringSlice(v)
	if err != nil {
		return nil, err
	}
	if len(mods) == 0 {
		return nil, nil
	}
	for _, m := range mods {
		switch m {
		case "cmd", "ctrl", "alt", "shift":
		default:
			return nil, fmt.Errorf("modifier %q not in {cmd, ctrl, alt, shift}", m)
		}
	}
	return mods, nil
}
