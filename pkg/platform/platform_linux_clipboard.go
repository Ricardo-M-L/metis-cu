//go:build linux

package platform

// Linux clipboard via golang.design/x/clipboard. Same lazy-init pattern
// as darwin (the library shares one Init() across all platforms).
// Under X11 the library uses XCB; under XWayland it works transparently;
// native Wayland may need wl-clipboard-rs (not bundled).

import (
	"fmt"
	"sync"

	"golang.design/x/clipboard"
)

var (
	clipboardInitOnceLinux sync.Once
	clipboardInitErrLinux  error
)

func ensureClipboardLinux() error {
	clipboardInitOnceLinux.Do(func() {
		if err := clipboard.Init(); err != nil {
			clipboardInitErrLinux = fmt.Errorf("clipboard init: %w", err)
		}
	})
	return clipboardInitErrLinux
}

func (p *linuxPlatform) ClipboardRead() (string, error) {
	if err := ensureClipboardLinux(); err != nil {
		return "", err
	}
	return string(clipboard.Read(clipboard.FmtText)), nil
}

func (p *linuxPlatform) ClipboardWrite(text string) error {
	if err := ensureClipboardLinux(); err != nil {
		return err
	}
	clipboard.Write(clipboard.FmtText, []byte(text))
	return nil
}

// ClipboardSnapshot / Restore: see darwin twin (DD-1).
func (p *linuxPlatform) ClipboardSnapshot() ClipboardSnapshot {
	if err := ensureClipboardLinux(); err != nil {
		return ClipboardSnapshot{Empty: true}
	}
	raw := clipboard.Read(clipboard.FmtText)
	if raw == nil {
		return ClipboardSnapshot{Empty: true}
	}
	return ClipboardSnapshot{Text: string(raw)}
}

func (p *linuxPlatform) ClipboardRestore(s ClipboardSnapshot) error {
	if s.Empty {
		return nil
	}
	return p.ClipboardWrite(s.Text)
}
