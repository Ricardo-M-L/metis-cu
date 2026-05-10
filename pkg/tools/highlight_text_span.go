package tools

// highlight_text_span — Tier-1 borrow from Agent-S
// (`gui_agents/s2_5/agents/grounding.py:502-521`). Selects on-screen
// text by content, not by coordinates: model emits start_phrase and
// end_phrase (as substrings of words visible on screen), tool OCRs,
// finds the first word matching start at the leftmost row and the
// last word matching end below it, then drag-selects from start's
// left edge to end's right edge.
//
// Useful for "highlight the second paragraph", "select from 'Dear'
// to 'Sincerely'" — operations that are hard to express with raw
// drag coordinates because the model can't precisely measure word
// positions in a downsampled screenshot.
//
// Backend: same Platform.OCR + Platform.MouseDrag the existing tools
// use. Routes through gateOrDeny("left_click_drag") for tier parity.

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "highlight_text_span",
			Description: "Select on-screen text by content. " +
				"Provide `start_phrase` (matched against the FIRST word in the span) and " +
				"`end_phrase` (matched against the LAST word). The tool OCRs, finds the " +
				"matching word boundaries, and drag-selects between them. Both phrases " +
				"are case-insensitive substring matches. Backend depends on " +
				"Platform.OCR — same caveats as click_text (Linux: tesseract; " +
				"darwin/windows: pending native bindings).",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"start_phrase": map[string]any{
						"type":        "string",
						"description": "Word at the start of the span (case-insensitive substring match).",
						"minLength":   1,
					},
					"end_phrase": map[string]any{
						"type":        "string",
						"description": "Word at the end of the span (case-insensitive substring match). Must appear AFTER start_phrase in document order or the call errors.",
						"minLength":   1,
					},
					"return_screenshot": returnScreenshotSchema(),
				},
				"required":             []string{"start_phrase", "end_phrase"},
				"additionalProperties": false,
			},
			Handler: handleHighlightTextSpan,
		})
	})
}

func handleHighlightTextSpan(ctx context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	startQ, err := requireString(params, "start_phrase")
	if err != nil {
		return &Result{Text: fmt.Sprintf("highlight_text_span: %v", err), IsError: true}, nil
	}
	endQ, err := requireString(params, "end_phrase")
	if err != nil {
		return &Result{Text: fmt.Sprintf("highlight_text_span: %v", err), IsError: true}, nil
	}
	if denied, deny := gateOrDeny(plat, "left_click_drag"); deny {
		return denied, nil
	}
	img, err := plat.Screenshot()
	if err != nil {
		return &Result{Text: fmt.Sprintf("highlight_text_span: screenshot: %v", err), IsError: true}, nil
	}
	regions, err := plat.OCR(img)
	if err != nil {
		if errors.Is(err, platform.ErrNotImplemented) {
			return &Result{Text: "highlight_text_span: OCR backend not available on this platform yet (Linux: install tesseract-ocr; darwin/windows: pending native bindings)", IsError: true}, nil
		}
		return &Result{Text: fmt.Sprintf("highlight_text_span: OCR: %v", err), IsError: true}, nil
	}
	startIdx := firstMatchIndex(regions, startQ)
	if startIdx < 0 {
		return &Result{Text: fmt.Sprintf("highlight_text_span: no on-screen text matched start_phrase %q", startQ), IsError: true}, nil
	}
	endIdx := lastMatchIndex(regions[startIdx+1:], endQ)
	if endIdx < 0 {
		return &Result{Text: fmt.Sprintf("highlight_text_span: no on-screen text matched end_phrase %q AFTER start_phrase", endQ), IsError: true}, nil
	}
	endIdx += startIdx + 1 // un-shift the slice offset

	startBox := regions[startIdx].Bounds
	endBox := regions[endIdx].Bounds
	from := platform.Point{X: startBox.Min.X, Y: (startBox.Min.Y + startBox.Max.Y) / 2}
	to := platform.Point{X: endBox.Max.X, Y: (endBox.Min.Y + endBox.Max.Y) / 2}

	if err := plat.MouseDrag(ctx, from, to, platform.ButtonLeft); err != nil {
		return &Result{Text: fmt.Sprintf("highlight_text_span: drag (%d,%d)→(%d,%d): %v", from.X, from.Y, to.X, to.Y, err), IsError: true}, nil
	}
	imgB64, mime := settleAndMaybeShot(ctx, plat, params)
	return &Result{
		Text:     fmt.Sprintf("selected from %q at (%d,%d) to %q at (%d,%d)", regions[startIdx].Text, from.X, from.Y, regions[endIdx].Text, to.X, to.Y),
		Image:    imgB64,
		MIMEType: mime,
	}, nil
}

func firstMatchIndex(regions []platform.OCRResult, q string) int {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return -1
	}
	for i, r := range regions {
		if strings.Contains(strings.ToLower(r.Text), q) {
			return i
		}
	}
	return -1
}

func lastMatchIndex(regions []platform.OCRResult, q string) int {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return -1
	}
	for i := len(regions) - 1; i >= 0; i-- {
		if strings.Contains(strings.ToLower(regions[i].Text), q) {
			return i
		}
	}
	return -1
}
