//go:build darwin

package platform

// Empirical triple-click verification (BUG-23).
//
// The fix routes count>=2 clicks through robotgo.MultiClick which sets
// kCGMouseEventClickState=count on the synthetic CGEvent. macOS apps
// inspect that field to decide whether back-to-back clicks are a
// double / triple gesture. Reading robotgo's source confirms the
// mechanism, but only an end-to-end test against a real text editor
// proves the click ACTUALLY selects a line.
//
// This test:
//   1. Launches TextEdit + opens a fresh untitled document via osascript
//   2. Types a known sentence
//   3. Moves the cursor over the sentence
//   4. Calls MouseClick(..., count=3)
//   5. Asks TextEdit for `selection's contents` via osascript
//   6. Verifies the selection length matches the line (single click
//      would return 0 chars, double would return one word, triple
//      should return the full line)
//
// Skipped by default — run only when explicitly enabled, since:
//   - It opens TextEdit and steals focus, disrupting the user
//   - It requires Accessibility permission for the test binary
//   - It's slow (osascript round-trips)
//
// Enable with: METIS_CU_BUG23_VERIFY=1 go test ./pkg/platform -run TripleClick

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestTripleClick_SelectsLineInTextEdit(t *testing.T) {
	if os.Getenv("METIS_CU_BUG23_VERIFY") == "" {
		t.Skip("set METIS_CU_BUG23_VERIFY=1 to run the BUG-23 GUI integration test")
	}
	if os.Getenv("CI") != "" {
		t.Skip("CI runners don't have a GUI session for TextEdit")
	}

	const sentence = "metis-cu BUG-23 verification line one"

	// Open TextEdit with a new doc and bring to front.
	if err := exec.Command("osascript", "-e",
		`tell application "TextEdit" to activate`,
		"-e", `tell application "TextEdit" to make new document`,
	).Run(); err != nil {
		t.Fatalf("open TextEdit: %v", err)
	}
	defer func() {
		// Close without saving so we don't leave an untitled doc behind.
		_ = exec.Command("osascript", "-e",
			`tell application "TextEdit" to close every document saving no`,
		).Run()
	}()
	time.Sleep(800 * time.Millisecond) // let TextEdit settle / take focus

	// Type the sentence by setting it directly via osascript — avoids
	// dependency on our Type() (which is a separate code path under
	// test by other suites) and avoids race with focus.
	if err := exec.Command("osascript", "-e",
		`tell application "TextEdit" to set text of front document to "`+sentence+`"`,
	).Run(); err != nil {
		t.Fatalf("set text: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// Find roughly where the text is on screen. TextEdit's default
	// window opens around (200, 200) on most setups; the text starts
	// around (250, 100) inside that window. Click at (450, 250)
	// (window-relative ~250,150) — well into the typed sentence.
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	pt := Point{X: 450, Y: 250}
	if err := p.MouseClick(pt, ButtonLeft, 3); err != nil {
		t.Fatalf("triple-click: %v", err)
	}
	time.Sleep(150 * time.Millisecond)

	// Query the selected text. A successful triple-click selects the
	// whole line; a single click selects nothing (cursor only).
	out, err := exec.Command("osascript", "-e",
		`tell application "TextEdit" to get the selection of front document as string`,
	).Output()
	if err != nil {
		// Fallback: AppleScript's `selection` API is awkward; try via
		// System Events for the focused text field.
		out, err = exec.Command("osascript", "-e",
			`tell application "System Events" to tell process "TextEdit" to get value of attribute "AXSelectedText" of text area 1 of scroll area 1 of window 1`,
		).Output()
		if err != nil {
			t.Fatalf("query selection: %v (fallback also failed)", err)
		}
	}
	selected := strings.TrimSpace(string(out))
	t.Logf("selected after triple-click: %q (length %d)", selected, len(selected))
	if !strings.Contains(selected, "metis-cu BUG-23") {
		t.Errorf("triple-click did not select the line — got %q (want substring of %q)",
			selected, sentence)
	}
}
