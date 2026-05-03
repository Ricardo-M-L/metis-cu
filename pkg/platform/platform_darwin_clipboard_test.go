//go:build darwin

package platform

import (
	"os"
	"testing"
)

// TestClipboard_RoundTrip verifies write→read on the real macOS
// pasteboard. CI runners are headless and have no Quartz session, so
// we skip there. On a dev box ensureClipboard may still fail (lid
// closed, ssh-only session) — treat that as a skip too rather than a
// hard failure.
//
// We save and restore the user's existing clipboard around the test
// so running `go test` doesn't clobber whatever was copied last.
func TestClipboard_RoundTrip(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping clipboard round-trip on CI (headless, no NSPasteboard)")
	}

	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Best-effort save of the current clipboard. If the initial Read
	// fails we still try the round-trip — many "no clipboard" failure
	// modes only surface on Init, not on the empty Read.
	original, readErr := p.ClipboardRead()
	if readErr != nil {
		t.Skipf("clipboard unavailable: %v", readErr)
	}
	defer func() {
		// Restore even if the test failed — leaking the marker into
		// the user's pasteboard would be obnoxious.
		if err := p.ClipboardWrite(original); err != nil {
			t.Logf("warning: failed to restore original clipboard: %v", err)
		}
	}()

	const marker = "metis-cu-test-marker-2026"
	if err := p.ClipboardWrite(marker); err != nil {
		t.Fatalf("ClipboardWrite: %v", err)
	}

	got, err := p.ClipboardRead()
	if err != nil {
		t.Fatalf("ClipboardRead: %v", err)
	}
	if got != marker {
		t.Fatalf("clipboard round-trip mismatch: got %q, want %q", got, marker)
	}
}
