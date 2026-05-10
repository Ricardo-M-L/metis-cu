package tools

import (
	"fmt"
	"strconv"
)

// asInt coerces a JSON value into int. JSON numbers decode to float64
// by default; LLMs occasionally hand back stringified ints. Both are
// accepted, anything else is rejected.
func asInt(v any) (int, error) {
	switch x := v.(type) {
	case int:
		return x, nil
	case int32:
		return int(x), nil
	case int64:
		return int(x), nil
	case float64:
		if x != float64(int(x)) {
			return 0, fmt.Errorf("not an integer: %v", v)
		}
		return int(x), nil
	case float32:
		if x != float32(int(x)) {
			return 0, fmt.Errorf("not an integer: %v", v)
		}
		return int(x), nil
	case string:
		i, err := strconv.Atoi(x)
		if err != nil {
			return 0, fmt.Errorf("not an integer: %q", x)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("expected number, got %T", v)
	}
}

// asString coerces v into string. JSON strings decode to string already;
// callers without a default value use this to provide a uniform error.
func asString(v any) (string, error) {
	if s, ok := v.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("expected string, got %T", v)
}

// asStringSlice coerces v into []string. Accepts []any of strings (the
// JSON-decoded shape) and []string (programmatic callers).
func asStringSlice(v any) ([]string, error) {
	switch x := v.(type) {
	case []string:
		return x, nil
	case []any:
		out := make([]string, 0, len(x))
		for i, item := range x {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("element %d: expected string, got %T", i, item)
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected array of strings, got %T", v)
	}
}

// requireInt extracts a required integer field; missing → error.
func requireInt(params map[string]any, key string) (int, error) {
	v, ok := params[key]
	if !ok {
		return 0, fmt.Errorf("missing required field: %s", key)
	}
	return asInt(v)
}

// requireString extracts a required string field; missing → error.
func requireString(params map[string]any, key string) (string, error) {
	v, ok := params[key]
	if !ok {
		return "", fmt.Errorf("missing required field: %s", key)
	}
	return asString(v)
}

// optionalInt returns the int at key, or def if absent.
func optionalInt(params map[string]any, key string, def int) (int, error) {
	v, ok := params[key]
	if !ok {
		return def, nil
	}
	return asInt(v)
}

// optionalString returns the string at key, or def if absent.
func optionalString(params map[string]any, key string, def string) (string, error) {
	v, ok := params[key]
	if !ok {
		return def, nil
	}
	return asString(v)
}

// optionalBool returns the bool at key, or def if absent. Accepts the
// usual JSON bool, plus the string forms "true"/"false" so callers
// hand-writing TOML / shell args don't have to know which serialisation
// the wire uses today.
func optionalBool(params map[string]any, key string, def bool) (bool, error) {
	v, ok := params[key]
	if !ok {
		return def, nil
	}
	switch x := v.(type) {
	case bool:
		return x, nil
	case string:
		switch x {
		case "true", "True", "TRUE", "1", "yes":
			return true, nil
		case "false", "False", "FALSE", "0", "no", "":
			return false, nil
		default:
			return false, fmt.Errorf("expected bool, got %q", x)
		}
	default:
		return false, fmt.Errorf("expected bool, got %T", v)
	}
}

// asFloat coerces v into float64. Accepts JSON numbers (which decode to
// float64), int variants (so callers passing programmatic values aren't
// surprised), and stringified numbers.
func asFloat(v any) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case float32:
		return float64(x), nil
	case int:
		return float64(x), nil
	case int32:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case string:
		f, err := strconv.ParseFloat(x, 64)
		if err != nil {
			return 0, fmt.Errorf("not a number: %q", x)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("expected number, got %T", v)
	}
}

// requireFloat extracts a required float field; missing → error.
func requireFloat(params map[string]any, key string) (float64, error) {
	v, ok := params[key]
	if !ok {
		return 0, fmt.Errorf("missing required field: %s", key)
	}
	return asFloat(v)
}

// Rect is a logical-pixel rectangle used by `zoom` and any future tool
// that takes a {x,y,w,h} sub-object. Kept separate from platform.Point
// because the platform layer doesn't deal in sized regions today.
type Rect struct {
	X, Y, W, H int
}

// requireRect extracts a {x,y,w,h} sub-object from params under `key`.
// Same shape convention as requirePoint but with width/height.
func requireRect(params map[string]any, key string) (Rect, error) {
	v, ok := params[key]
	if !ok {
		return Rect{}, fmt.Errorf("missing required field: %s", key)
	}
	m, ok := v.(map[string]any)
	if !ok {
		return Rect{}, fmt.Errorf("%s: expected object, got %T", key, v)
	}
	x, err := requireInt(m, "x")
	if err != nil {
		return Rect{}, fmt.Errorf("%s.x: %w", key, err)
	}
	y, err := requireInt(m, "y")
	if err != nil {
		return Rect{}, fmt.Errorf("%s.y: %w", key, err)
	}
	w, err := requireInt(m, "w")
	if err != nil {
		return Rect{}, fmt.Errorf("%s.w: %w", key, err)
	}
	h, err := requireInt(m, "h")
	if err != nil {
		return Rect{}, fmt.Errorf("%s.h: %w", key, err)
	}
	if w <= 0 || h <= 0 {
		return Rect{}, fmt.Errorf("%s: w and h must be positive (got w=%d, h=%d)", key, w, h)
	}
	return Rect{X: x, Y: y, W: w, H: h}, nil
}
