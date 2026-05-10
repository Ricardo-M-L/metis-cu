package tools

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// readClipboardMaxBytes is the legacy alias kept so the schema
// description compiles without re-templating. Runtime cap comes from
// clipboardMaxBytesFor(ctx) which honours the Registry override.
const readClipboardMaxBytes = DefaultClipboardMaxBytes

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "read_clipboard",
			Description: "Read the system clipboard text. Returns the current text " +
				"content; if the clipboard is empty or holds non-text data (image, " +
				"file URL, etc.) the result text describes that explicitly so the " +
				"model can distinguish empty-vs-non-text. Capped at 64KB; longer " +
				"payloads are truncated with a count of total bytes.",
			Schema:  noArgsSchema(),
			Handler: handleReadClipboard,
		})
	})
}

func handleReadClipboard(ctx context.Context, plat platform.Platform, _ map[string]any) (*Result, error) {
	text, err := plat.ClipboardRead()
	if err != nil {
		return &Result{Text: fmt.Sprintf("read_clipboard: %v", err), IsError: true}, nil
	}
	if text == "" {
		// BUG-15: distinguish "actually empty" from "non-text payload" so
		// the model doesn't think a copy-from-image silently failed.
		return &Result{Text: "(clipboard empty or holds non-text content)"}, nil
	}
	cap := clipboardMaxBytesFor(ctx)
	if len(text) > cap {
		truncated := text[:cap]
		return &Result{
			Text: fmt.Sprintf("%s\n\n…(truncated; total %d bytes, showing first %d)", truncated, len(text), cap),
		}, nil
	}
	return &Result{Text: text}, nil
}
