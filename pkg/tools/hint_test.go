package tools

import (
	"strings"
	"testing"
)

func TestAppendHint_EmptyHintIsNoOp(t *testing.T) {
	got := AppendHint("clicked at (10, 20)", "")
	if got != "clicked at (10, 20)" {
		t.Errorf("empty hint changed text: %q", got)
	}
}

func TestAppendHint_EmptyTextReturnsWrappedHint(t *testing.T) {
	got := AppendHint("", "use click_text instead")
	if !strings.Contains(got, HintTagOpen) || !strings.Contains(got, "use click_text instead") {
		t.Errorf("empty text + hint should produce wrapped hint; got %q", got)
	}
	if strings.HasPrefix(got, "\n") {
		t.Errorf("no leading newline expected; got %q", got)
	}
}

func TestAppendHint_NormalCase(t *testing.T) {
	got := AppendHint("scrolled at (10, 20) by (dx=0, dy=1)",
		"You used raw (x,y). Prefer modifiers field for ctrl+wheel.")
	if !strings.HasPrefix(got, "scrolled at (10, 20)") {
		t.Errorf("hint should append AFTER text; got %q", got)
	}
	if !strings.Contains(got, HintTagOpen) || !strings.Contains(got, HintTagClose) {
		t.Errorf("missing tag pair: %q", got)
	}
	hints := ExtractHints(got)
	if len(hints) != 1 || !strings.Contains(hints[0], "ctrl+wheel") {
		t.Errorf("ExtractHints round-trip failed: %v", hints)
	}
}

func TestStripHints_RemovesAllSections(t *testing.T) {
	in := "first " + HintTagOpen + "\nA\n" + HintTagClose + " middle " + HintTagOpen + "\nB\n" + HintTagClose + " last"
	got := StripHints(in)
	for _, must := range []string{"first", "middle", "last"} {
		if !strings.Contains(got, must) {
			t.Errorf("strip dropped %q: %q", must, got)
		}
	}
	for _, must := range []string{"A", "B", HintTagOpen, HintTagClose} {
		if strings.Contains(got, must) {
			t.Errorf("strip kept %q: %q", must, got)
		}
	}
}

// TestStripHints_UnterminatedLeavesContent: a stray open tag with no
// close shouldn't swallow trailing text — defensive against a future
// tool that emits a malformed hint.
func TestStripHints_UnterminatedLeavesContent(t *testing.T) {
	in := "before " + HintTagOpen + " important trailing content"
	got := StripHints(in)
	if !strings.Contains(got, "trailing content") {
		t.Errorf("trailing content lost: %q", got)
	}
}

func TestExtractHints_OrderPreserved(t *testing.T) {
	in := AppendHint(AppendHint("base", "first hint"), "second hint")
	hints := ExtractHints(in)
	if len(hints) != 2 || hints[0] != "first hint" || hints[1] != "second hint" {
		t.Errorf("ExtractHints order/content wrong: %v", hints)
	}
}
