package platform

// parseKeyCombo is shared across all OS keyboard implementations
// (darwin / linux / windows). The combo grammar is the same on every
// platform — '+'-joined, last segment is the key, preceding segments
// are modifiers — and the lowercase normalization keeps the lookup
// against robotgo's name table consistent.
//
// Returns []any (rather than []string) because robotgo's KeyTap /
// KeyToggle take variadic any so the slice can be spread directly.

import (
	"fmt"
	"strings"
)

func parseKeyCombo(combo string) (key string, mods []any, err error) {
	combo = strings.TrimSpace(combo)
	if combo == "" {
		return "", nil, fmt.Errorf("empty key combo")
	}
	parts := strings.Split(combo, "+")
	for i, part := range parts {
		parts[i] = strings.ToLower(strings.TrimSpace(part))
	}
	key = parts[len(parts)-1]
	if key == "" {
		return "", nil, fmt.Errorf("empty key in combo %q", combo)
	}
	if len(parts) == 1 {
		return key, nil, nil
	}
	mods = make([]any, 0, len(parts)-1)
	for _, m := range parts[:len(parts)-1] {
		if m == "" {
			return "", nil, fmt.Errorf("empty modifier in combo %q", combo)
		}
		mods = append(mods, m)
	}
	return key, mods, nil
}
