package tools

import (
	"context"
	"errors"
	"testing"
)

type fakeClipboardReadPlat struct {
	stubPlat
	readErr   error
	readValue string
}

func (p *fakeClipboardReadPlat) ClipboardRead() (string, error) {
	return p.readValue, p.readErr
}

func TestReadClipboard_OK(t *testing.T) {
	p := &fakeClipboardReadPlat{readValue: "hello from clipboard"}
	res, err := handleReadClipboard(context.Background(), p, nil)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if res.Text != "hello from clipboard" {
		t.Errorf("got %q, want %q", res.Text, "hello from clipboard")
	}
}

func TestReadClipboard_EmptyClipboard(t *testing.T) {
	p := &fakeClipboardReadPlat{readValue: ""}
	res, err := handleReadClipboard(context.Background(), p, nil)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("empty clipboard should not be an error: %s", res.Text)
	}
	// BUG-15: empty / non-text payload returns a descriptive sentinel
	// rather than "" so the model can distinguish "empty" from a copy
	// that silently dropped non-text content.
	if res.Text != "(clipboard empty or holds non-text content)" {
		t.Errorf("got %q, want sentinel describing empty/non-text", res.Text)
	}
}

func TestReadClipboard_PlatformError(t *testing.T) {
	p := &fakeClipboardReadPlat{readErr: errors.New("clipboard read failed")}
	res, _ := handleReadClipboard(context.Background(), p, nil)
	if !res.IsError {
		t.Fatal("expected IsError when ClipboardRead fails")
	}
}
