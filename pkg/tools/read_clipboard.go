package tools

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "read_clipboard",
			Description: "Read the system clipboard. Returns the current text " +
				"content, or an empty string if the clipboard is empty or holds " +
				"non-text data.",
			Schema: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
			Handler: handleReadClipboard,
		})
	})
}

func handleReadClipboard(_ context.Context, plat platform.Platform, _ map[string]any) (*Result, error) {
	text, err := plat.ClipboardRead()
	if err != nil {
		return &Result{Text: fmt.Sprintf("read_clipboard: %v", err), IsError: true}, nil
	}
	return &Result{Text: text}, nil
}
