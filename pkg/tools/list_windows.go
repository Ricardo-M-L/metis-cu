package tools

// list_windows — Phase I-mini Tier-1 borrow. Pragmatic subset of the
// full accessibility-tree exposure originally scoped: returns
// [{app, title, x, y, width, height}] for every visible top-level
// window without needing per-OS a11y permissions or the accessibility
// bus that headless CI runners typically lack.
//
// Backend: Platform.ListWindows(). macOS via osascript, Linux via
// wmctrl, Windows currently returns ErrNotImplemented (the Win32
// EnumWindows path is queued as a follow-up).
//
// Pairs naturally with screen_size (display geometry), cursor_position
// (pointer state), and frontmost-app gating (the existing tier
// system) to give the model a full reconnaissance kit before acting.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type listWindowEntry struct {
	App    string `json:"app"`
	Title  string `json:"title"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type listWindowReport struct {
	Query   string            `json:"query,omitempty"`
	Total   int               `json:"total"`
	Windows []listWindowEntry `json:"windows"`
}

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "list_windows",
			Description: "Enumerate every visible top-level window across every running app. " +
				"Returns JSON: per-window app, title, and bounds in logical pixels. " +
				"With optional `query`, filters case-insensitively to substring matches " +
				"on either app OR title. Read-only — no tier gate. Backend: " +
				"osascript on macOS, wmctrl on Linux; Windows currently returns an " +
				"unavailable error (Win32 EnumWindows binding pending).",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Optional case-insensitive substring filter applied to both app and title. Omit to return every visible window.",
					},
				},
				"additionalProperties": false,
			},
			Handler: handleListWindows,
		})
	})
}

func handleListWindows(_ context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	query, err := optionalString(params, "query", "")
	if err != nil {
		return &Result{Text: fmt.Sprintf("list_windows: %v", err), IsError: true}, nil
	}
	wins, err := plat.ListWindows()
	if err != nil {
		if errors.Is(err, platform.ErrNotImplemented) {
			return &Result{Text: "list_windows: backend not available on this platform yet (Linux: install wmctrl; windows: pending Win32 EnumWindows binding)", IsError: true}, nil
		}
		return &Result{Text: fmt.Sprintf("list_windows: %v", err), IsError: true}, nil
	}
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]listWindowEntry, 0, len(wins))
	for _, w := range wins {
		if q != "" {
			matched := strings.Contains(strings.ToLower(w.App), q) ||
				strings.Contains(strings.ToLower(w.Title), q)
			if !matched {
				continue
			}
		}
		out = append(out, listWindowEntry{
			App:    w.App,
			Title:  w.Title,
			X:      w.Bounds.Min.X,
			Y:      w.Bounds.Min.Y,
			Width:  w.Bounds.Dx(),
			Height: w.Bounds.Dy(),
		})
	}
	body, err := json.Marshal(listWindowReport{
		Query:   query,
		Total:   len(out),
		Windows: out,
	})
	if err != nil {
		return &Result{Text: fmt.Sprintf("list_windows: marshal: %v", err), IsError: true}, nil
	}
	return &Result{Text: string(body)}, nil
}
