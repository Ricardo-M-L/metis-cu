package tools

// find_text_on_screen — read-only complement to click_text. Returns
// every OCR'd region that matches an optional substring filter,
// without acting. Useful for the model to inspect what's actually on
// screen before picking a target — pairs naturally with the existing
// `cursor_position` and `screen_size` reconnaissance pattern.
//
// Backend dependency: same Platform.OCR() as click_text. TierRead-
// safe (vision only); no gate enforcement needed.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type findTextRegion struct {
	Text       string  `json:"text"`
	X          int     `json:"x"`
	Y          int     `json:"y"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	Confidence float64 `json:"confidence"`
}

type findTextReport struct {
	Query   string           `json:"query,omitempty"`
	Total   int              `json:"total"`
	Regions []findTextRegion `json:"regions"`
}

const findTextDefaultLimit = 200

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "find_text_on_screen",
			Description: "OCR the active screenshot and return every text region. " +
				"With `query`, filters case-insensitively to substring matches. " +
				"Read-only — no clicks, no gating. Use to reconnaissance the screen " +
				"before click_text or to verify a label appeared after an action. " +
				"Backend: tesseract on Linux; ErrNotImplemented on darwin/windows " +
				"pending native bindings.",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Optional substring filter (case-insensitive). Omit to return every recognised region (capped at 200 by default).",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Max regions to return. Defaults to 200 — set lower to keep the response compact when the screen is text-heavy.",
						"minimum":     1,
						"default":     findTextDefaultLimit,
					},
				},
				"additionalProperties": false,
			},
			Handler: handleFindTextOnScreen,
		})
	})
}

func handleFindTextOnScreen(_ context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	query, err := optionalString(params, "query", "")
	if err != nil {
		return &Result{Text: fmt.Sprintf("find_text_on_screen: %v", err), IsError: true}, nil
	}
	limit, err := optionalInt(params, "limit", findTextDefaultLimit)
	if err != nil {
		return &Result{Text: fmt.Sprintf("find_text_on_screen: %v", err), IsError: true}, nil
	}
	if limit <= 0 {
		limit = findTextDefaultLimit
	}
	img, err := plat.Screenshot()
	if err != nil {
		return &Result{Text: fmt.Sprintf("find_text_on_screen: screenshot: %v", err), IsError: true}, nil
	}
	regions, err := plat.OCR(img)
	if err != nil {
		if errors.Is(err, platform.ErrNotImplemented) {
			return &Result{Text: "find_text_on_screen: OCR backend not available on this platform yet (Linux: install tesseract-ocr; darwin/windows: pending native bindings)", IsError: true}, nil
		}
		return &Result{Text: fmt.Sprintf("find_text_on_screen: OCR: %v", err), IsError: true}, nil
	}
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]findTextRegion, 0, len(regions))
	for _, r := range regions {
		if q != "" && !strings.Contains(strings.ToLower(r.Text), q) {
			continue
		}
		out = append(out, findTextRegion{
			Text:       r.Text,
			X:          r.Bounds.Min.X,
			Y:          r.Bounds.Min.Y,
			Width:      r.Bounds.Dx(),
			Height:     r.Bounds.Dy(),
			Confidence: r.Confidence,
		})
		if len(out) >= limit {
			break
		}
	}
	body, err := json.Marshal(findTextReport{
		Query:   query,
		Total:   len(out),
		Regions: out,
	})
	if err != nil {
		return &Result{Text: fmt.Sprintf("find_text_on_screen: marshal: %v", err), IsError: true}, nil
	}
	return &Result{Text: string(body)}, nil
}
