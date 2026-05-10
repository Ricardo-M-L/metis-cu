package tools

import (
	"context"
	"errors"
	"testing"
)

type fakeKeyPlat struct {
	stubPlat
	keyErr error
}

func (p *fakeKeyPlat) KeyPress(combo string) error { return p.keyErr }

func TestKey_OK(t *testing.T) {
	p := &fakeKeyPlat{}
	params := map[string]any{"combo": "cmd+a"}
	res, err := handleKey(context.Background(), p, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
}

func TestKey_MissingCombo(t *testing.T) {
	p := &fakeKeyPlat{}
	res, _ := handleKey(context.Background(), p, map[string]any{})
	if !res.IsError {
		t.Fatal("expected IsError when combo is missing")
	}
}

func TestKey_PlatformError(t *testing.T) {
	p := &fakeKeyPlat{keyErr: errors.New("key press failed")}
	params := map[string]any{"combo": "esc"}
	res, _ := handleKey(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when KeyPress fails")
	}
}
