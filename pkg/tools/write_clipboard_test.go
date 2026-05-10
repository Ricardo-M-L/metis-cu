package tools

import (
	"context"
	"errors"
	"testing"
)

type fakeClipboardWritePlat struct {
	stubPlat
	writeErr error
}

func (p *fakeClipboardWritePlat) ClipboardWrite(text string) error { return p.writeErr }

func TestWriteClipboard_OK(t *testing.T) {
	p := &fakeClipboardWritePlat{}
	params := map[string]any{"text": "hello, clipboard"}
	res, err := handleWriteClipboard(context.Background(), p, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
}

func TestWriteClipboard_EmptyText(t *testing.T) {
	p := &fakeClipboardWritePlat{}
	params := map[string]any{"text": ""}
	res, err := handleWriteClipboard(context.Background(), p, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("empty text should be allowed: %s", res.Text)
	}
}

func TestWriteClipboard_MissingText(t *testing.T) {
	p := &fakeClipboardWritePlat{}
	res, _ := handleWriteClipboard(context.Background(), p, map[string]any{})
	if !res.IsError {
		t.Fatal("expected IsError when text is missing")
	}
}

func TestWriteClipboard_PlatformError(t *testing.T) {
	p := &fakeClipboardWritePlat{writeErr: errors.New("clipboard write failed")}
	params := map[string]any{"text": "test"}
	res, _ := handleWriteClipboard(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when ClipboardWrite fails")
	}
}
