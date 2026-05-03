package tools

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name:        "write_clipboard",
			Description: "Replace the system clipboard with the provided text.",
			Schema: map[string]any{
				"type":     "object",
				"required": []string{"text"},
				"properties": map[string]any{
					"text": map[string]any{
						"type":        "string",
						"description": "UTF-8 text to place on the clipboard. Replaces any prior content.",
					},
				},
				"additionalProperties": false,
			},
			Handler: handleWriteClipboard,
		})
	})
}

func handleWriteClipboard(_ context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	text, err := requireString(params, "text")
	if err != nil {
		return &Result{Text: fmt.Sprintf("write_clipboard: %v", err), IsError: true}, nil
	}
	if err := plat.ClipboardWrite(text); err != nil {
		return &Result{Text: fmt.Sprintf("write_clipboard: %v", err), IsError: true}, nil
	}
	return &Result{Text: fmt.Sprintf("wrote %d bytes to clipboard", len(text))}, nil
}
