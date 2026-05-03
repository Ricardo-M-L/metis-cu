//go:build darwin

package platform

// Clipboard. Real implementation (Sprint 2-C): NSPasteboard via
// golang.design/x/clipboard. The library's Init() must be called once
// per process before Read/Write — we wrap it with sync.Once so the
// first ClipboardRead/ClipboardWrite call lazily initializes, and any
// init failure (e.g. headless / no Quartz access) is cached so callers
// see a deterministic "clipboard init: ..." error instead of a panic.

import (
	"fmt"
	"sync"

	"golang.design/x/clipboard"
)

var (
	clipboardInitOnce sync.Once
	clipboardInitErr  error
)

// ensureClipboard runs clipboard.Init at most once and caches the
// outcome. clipboard.Read/Write on macOS can panic if Init hasn't been
// called or returned an error, so every consumer in this package must
// gate on this helper first.
func ensureClipboard() error {
	clipboardInitOnce.Do(func() {
		if err := clipboard.Init(); err != nil {
			clipboardInitErr = fmt.Errorf("clipboard init: %w", err)
		}
	})
	return clipboardInitErr
}

// ClipboardRead returns the current text content of the system
// clipboard. Empty clipboard or non-text content (image, file URL,
// etc.) returns "" with a nil error — the caller decides how to
// surface that distinction.
func (p *darwinPlatform) ClipboardRead() (string, error) {
	if err := ensureClipboard(); err != nil {
		return "", err
	}
	return string(clipboard.Read(clipboard.FmtText)), nil
}

// ClipboardWrite replaces the system clipboard with text. The
// underlying library's Write returns a "you've been overwritten"
// channel that we ignore — MCP semantics are fire-and-forget.
func (p *darwinPlatform) ClipboardWrite(text string) error {
	if err := ensureClipboard(); err != nil {
		return err
	}
	clipboard.Write(clipboard.FmtText, []byte(text))
	return nil
}
