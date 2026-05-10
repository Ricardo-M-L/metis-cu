//go:build windows

package platform

// Windows clipboard via golang.design/x/clipboard (Win32
// OpenClipboard/SetClipboardData/CloseClipboard under the hood). Same
// lazy-init pattern as darwin / linux.

import (
	"fmt"
	"sync"

	"golang.design/x/clipboard"
)

var (
	clipboardInitOnceWindows sync.Once
	clipboardInitErrWindows  error
)

func ensureClipboardWindows() error {
	clipboardInitOnceWindows.Do(func() {
		if err := clipboard.Init(); err != nil {
			clipboardInitErrWindows = fmt.Errorf("clipboard init: %w", err)
		}
	})
	return clipboardInitErrWindows
}

func (p *windowsPlatform) ClipboardRead() (string, error) {
	if err := ensureClipboardWindows(); err != nil {
		return "", err
	}
	return string(clipboard.Read(clipboard.FmtText)), nil
}

func (p *windowsPlatform) ClipboardWrite(text string) error {
	if err := ensureClipboardWindows(); err != nil {
		return err
	}
	clipboard.Write(clipboard.FmtText, []byte(text))
	return nil
}

// ClipboardSnapshot / Restore: see darwin twin (DD-1).
func (p *windowsPlatform) ClipboardSnapshot() ClipboardSnapshot {
	if err := ensureClipboardWindows(); err != nil {
		return ClipboardSnapshot{Empty: true}
	}
	raw := clipboard.Read(clipboard.FmtText)
	if raw == nil {
		return ClipboardSnapshot{Empty: true}
	}
	return ClipboardSnapshot{Text: string(raw)}
}

func (p *windowsPlatform) ClipboardRestore(s ClipboardSnapshot) error {
	if s.Empty {
		return nil
	}
	return p.ClipboardWrite(s.Text)
}
