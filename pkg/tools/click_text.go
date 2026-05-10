package tools

// click_text — Tier-1 borrow from open-interpreter (`mouse.py:44-204`)
// + self-operating-computer's OCR-anchored click. The model emits a
// label ("Submit", "OK"); the tool screenshots, OCRs, finds the
// matching word region, and clicks its center. Eliminates the worst
// failure mode of raw (x,y) clicking — model misreads a downsampled
// screenshot and lands a button-width off.
//
// Multi-match contract: when N candidates >= 2, returns IsError with
// a JSON list of (rank, x, y, text). Model retries with `occurrence`
// to pick a specific one. Single match auto-clicks. Zero matches is
// a clear "no text matched query" error.
//
// Backend dependency: Platform.OCR(). Linux ships with tesseract
// shell-out; darwin / windows currently return ErrNotImplemented and
// surface a clean "OCR backend not available" message.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type clickTextCandidate struct {
	Rank   int    `json:"rank"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Text   string `json:"text"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type clickTextAmbiguous struct {
	Query      string               `json:"query"`
	Candidates []clickTextCandidate `json:"candidates"`
	Hint       string               `json:"hint"`
}

func init() {
	addRegistration(func(r *Registry) {
		props := map[string]any{
			"text": map[string]any{
				"type":        "string",
				"description": "Substring to match against on-screen text (case-insensitive). Match is per-OCR-word; for phrases match the most distinctive word.",
				"minLength":   1,
			},
			"occurrence": map[string]any{
				"type":        "integer",
				"description": "Which match to click when multiple candidates exist. 0-based; default 0 (first). Set after a previous call returned an ambiguous-match list.",
				"minimum":     0,
				"default":     0,
			},
			"button": map[string]any{
				"type":        "string",
				"enum":        []string{"left", "right", "middle"},
				"default":     "left",
				"description": "Which mouse button to click with. Defaults to left.",
			},
			"return_screenshot": returnScreenshotSchema(),
		}
		r.register(Spec{
			Name: "click_text",
			Description: "Find on-screen text and click its centre. " +
				"Higher accuracy than raw (x,y) clicking when a label is visible. " +
				"On multiple matches, returns IsError + a JSON candidate list with " +
				"`rank` indices; retry with `occurrence=<rank>`. Backend: OCR via " +
				"the active platform's text recogniser (Linux: tesseract; darwin/" +
				"windows: ErrNotImplemented pending native bindings).",
			Schema: map[string]any{
				"type":                 "object",
				"properties":           props,
				"required":             []string{"text"},
				"additionalProperties": false,
			},
			Handler: handleClickText,
		})
	})
}

func handleClickText(ctx context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	query, err := requireString(params, "text")
	if err != nil {
		return &Result{Text: fmt.Sprintf("click_text: %v", err), IsError: true}, nil
	}
	occurrence, err := optionalInt(params, "occurrence", 0)
	if err != nil {
		return &Result{Text: fmt.Sprintf("click_text: %v", err), IsError: true}, nil
	}
	if occurrence < 0 {
		return &Result{Text: fmt.Sprintf("click_text: occurrence must be >= 0, got %d", occurrence), IsError: true}, nil
	}
	buttonStr, err := optionalString(params, "button", "left")
	if err != nil {
		return &Result{Text: fmt.Sprintf("click_text: %v", err), IsError: true}, nil
	}
	btn, err := parseClickButton(buttonStr)
	if err != nil {
		return &Result{Text: fmt.Sprintf("click_text: %v", err), IsError: true}, nil
	}
	// Use the same gate as left_click — clicking is a TierClick action
	// regardless of how the coordinate was derived.
	if denied, deny := gateOrDeny(plat, "left_click"); deny {
		return denied, nil
	}
	img, err := plat.Screenshot()
	if err != nil {
		return &Result{Text: fmt.Sprintf("click_text: screenshot: %v", err), IsError: true}, nil
	}
	regions, err := plat.OCR(img)
	if err != nil {
		if errors.Is(err, platform.ErrNotImplemented) {
			return &Result{Text: "click_text: OCR backend not available on this platform yet (Linux: install tesseract-ocr; darwin/windows: pending native bindings)", IsError: true}, nil
		}
		return &Result{Text: fmt.Sprintf("click_text: OCR: %v", err), IsError: true}, nil
	}
	candidates := matchCandidates(regions, query)
	if len(candidates) == 0 {
		return &Result{Text: fmt.Sprintf("click_text: no on-screen text matched %q", query), IsError: true}, nil
	}
	if len(candidates) > 1 && occurrence == 0 && !explicitOccurrence(params) {
		body, _ := json.Marshal(clickTextAmbiguous{
			Query:      query,
			Candidates: candidates,
			Hint:       "multiple matches — re-call with `occurrence=<rank>` to pick one",
		})
		return &Result{Text: string(body), IsError: true}, nil
	}
	if occurrence >= len(candidates) {
		return &Result{Text: fmt.Sprintf("click_text: occurrence %d out of range (have %d matches for %q)", occurrence, len(candidates), query), IsError: true}, nil
	}
	pick := candidates[occurrence]
	if err := plat.MouseClick(platform.Point{X: pick.X, Y: pick.Y}, btn, 1); err != nil {
		return &Result{Text: fmt.Sprintf("click_text: MouseClick(%d, %d): %v", pick.X, pick.Y, err), IsError: true}, nil
	}
	imgB64, mime := settleAndMaybeShot(ctx, plat, params)
	return &Result{
		Text:     fmt.Sprintf("clicked %q at (%d, %d) [match %d of %d]", pick.Text, pick.X, pick.Y, occurrence, len(candidates)),
		Image:    imgB64,
		MIMEType: mime,
	}, nil
}

// matchCandidates returns OCR regions whose lowercased text contains
// the lowercased query, in document order (top-to-bottom, left-to-
// right tie break). Each candidate carries its 0-based rank for the
// ambiguous-match retry contract.
func matchCandidates(regions []platform.OCRResult, query string) []clickTextCandidate {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	type sortable struct {
		r  platform.OCRResult
		ix int
	}
	var hits []sortable
	for i, r := range regions {
		if strings.Contains(strings.ToLower(r.Text), q) {
			hits = append(hits, sortable{r: r, ix: i})
		}
	}
	// Stable order from OCR engine — tesseract emits left-to-right,
	// top-to-bottom already, so preserve as-is.
	out := make([]clickTextCandidate, 0, len(hits))
	for rank, h := range hits {
		mid := h.r.Bounds.Min.Add(h.r.Bounds.Max.Sub(h.r.Bounds.Min).Div(2))
		out = append(out, clickTextCandidate{
			Rank:   rank,
			X:      mid.X,
			Y:      mid.Y,
			Text:   h.r.Text,
			Width:  h.r.Bounds.Dx(),
			Height: h.r.Bounds.Dy(),
		})
	}
	return out
}

// explicitOccurrence is true when the caller passed `occurrence` in
// params — used to distinguish "I want the first match" (occurrence=0
// supplied) from "I haven't decided yet, use the default 0". Without
// this distinction a follow-up retry that says occurrence=0 explicitly
// would still get the ambiguous-match branch.
func explicitOccurrence(params map[string]any) bool {
	_, ok := params["occurrence"]
	return ok
}

// parseClickButton maps the wire string to a platform.Button. Bad
// values yield a typed error that bubbles up as IsError.
func parseClickButton(s string) (platform.Button, error) {
	switch s {
	case "", "left":
		return platform.ButtonLeft, nil
	case "right":
		return platform.ButtonRight, nil
	case "middle":
		return platform.ButtonMiddle, nil
	default:
		return 0, fmt.Errorf("button %q not in {left, right, middle}", s)
	}
}
