package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type fakeTypePlat struct {
	stubPlat
	typeErr error

	// Records for the BUG-22 paste path so tests can assert the
	// snapshot/restore + KeyPress(cmd+v) sequence actually fires.
	snapshotCalls   int
	restoreCalls    []platform.ClipboardSnapshot
	clipboardWrites []string
	keyPresses      []string
	priorClipboard  string
}

func (p *fakeTypePlat) Type(_ context.Context, text string) error { return p.typeErr }

func (p *fakeTypePlat) ClipboardSnapshot() platform.ClipboardSnapshot {
	p.snapshotCalls++
	return platform.ClipboardSnapshot{Text: p.priorClipboard}
}

func (p *fakeTypePlat) ClipboardRestore(s platform.ClipboardSnapshot) error {
	p.restoreCalls = append(p.restoreCalls, s)
	return nil
}

func (p *fakeTypePlat) ClipboardWrite(text string) error {
	p.clipboardWrites = append(p.clipboardWrites, text)
	return nil
}

func (p *fakeTypePlat) KeyPress(combo string) error {
	p.keyPresses = append(p.keyPresses, combo)
	return nil
}

func TestType_OK(t *testing.T) {
	p := &fakeTypePlat{}
	params := map[string]any{"text": "hello, world"}
	res, err := handleTypeText(context.Background(), p, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
}

func TestType_EmptyText(t *testing.T) {
	p := &fakeTypePlat{}
	params := map[string]any{"text": ""}
	res, _ := handleTypeText(context.Background(), p, params)
	// Empty text is a no-op per spec — should succeed without error
	if res.IsError {
		t.Fatalf("empty text should be a no-op: %s", res.Text)
	}
}

func TestType_MissingText(t *testing.T) {
	p := &fakeTypePlat{}
	res, _ := handleTypeText(context.Background(), p, map[string]any{})
	if !res.IsError {
		t.Fatal("expected IsError when text is missing")
	}
}

func TestType_UnicodeChars(t *testing.T) {
	p := &fakeTypePlat{}
	params := map[string]any{"text": "你好世界 🎉"}
	res, err := handleTypeText(context.Background(), p, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError for unicode text: %s", res.Text)
	}
}

func TestType_PlatformError(t *testing.T) {
	p := &fakeTypePlat{typeErr: errors.New("type failed")}
	params := map[string]any{"text": "test"}
	res, _ := handleTypeText(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when Type fails")
	}
}

// TestType_LongTextPastes (BUG-22): a string >= the threshold should
// route through ClipboardSnapshot → ClipboardWrite → KeyPress("cmd+v")
// → ClipboardRestore. Asserts each step fires in order with the
// right payload, and Type() (the per-key path) is NOT invoked.
func TestType_LongTextPastes(t *testing.T) {
	long := strings.Repeat("a", DefaultTypePasteThreshold+10)
	p := &fakeTypePlat{priorClipboard: "user's prior copy"}
	params := map[string]any{"text": long}
	res, err := handleTypeText(context.Background(), p, params)
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected paste-path success, got IsError: %s", res.Text)
	}
	if p.snapshotCalls != 1 {
		t.Errorf("expected 1 ClipboardSnapshot call, got %d", p.snapshotCalls)
	}
	if len(p.clipboardWrites) != 1 || p.clipboardWrites[0] != long {
		t.Errorf("expected ClipboardWrite to receive the long text, got %v", p.clipboardWrites)
	}
	if len(p.keyPresses) != 1 || p.keyPresses[0] != "cmd+v" {
		t.Errorf("expected KeyPress(cmd+v), got %v", p.keyPresses)
	}
	if len(p.restoreCalls) != 1 || p.restoreCalls[0].Text != "user's prior copy" {
		t.Errorf("expected ClipboardRestore with prior content, got %v", p.restoreCalls)
	}
	if !strings.Contains(res.Text, "via paste") {
		t.Errorf("expected 'via paste' in result text, got: %s", res.Text)
	}
}

// TestType_ShortTextStaysOnPerKeyPath: under-threshold strings must
// NOT touch the clipboard — that's the BUG-22 fix's whole point of
// being conditional.
func TestType_ShortTextStaysOnPerKeyPath(t *testing.T) {
	p := &fakeTypePlat{}
	params := map[string]any{"text": "hi"}
	res, err := handleTypeText(context.Background(), p, params)
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected per-key success, got IsError: %s", res.Text)
	}
	if p.snapshotCalls != 0 || len(p.clipboardWrites) != 0 || len(p.keyPresses) != 0 {
		t.Errorf("short text should not touch clipboard: snapshot=%d writes=%v keys=%v",
			p.snapshotCalls, p.clipboardWrites, p.keyPresses)
	}
}

// TestType_ThresholdHonoursRegistryOverride: when registry sets a
// custom threshold via SetTypePasteThreshold, the boundary moves.
// Confirms the DD-3 config plumbing wires through.
func TestType_ThresholdHonoursRegistryOverride(t *testing.T) {
	p := &fakeTypePlat{}
	r := &Registry{plat: p, TypePasteThreshold: 5}
	ctx := context.WithValue(context.Background(), registryKey{}, r)
	res, _ := handleTypeText(ctx, p, map[string]any{"text": "hello!"}) // 6 runes >= 5
	if res.IsError {
		t.Fatalf("expected paste-path success, got IsError: %s", res.Text)
	}
	if p.snapshotCalls != 1 {
		t.Errorf("expected paste path triggered by registry threshold, got snapshotCalls=%d", p.snapshotCalls)
	}
}
