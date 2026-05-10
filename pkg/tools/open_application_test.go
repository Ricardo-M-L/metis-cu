package tools

import (
	"context"
	"errors"
	"testing"
)

type fakeOpenAppPlat struct {
	stubPlat
	openErr error
}

func (p *fakeOpenAppPlat) OpenApplication(_ context.Context, name string) error {
	return p.openErr
}

func TestOpenApplication_OK(t *testing.T) {
	p := &fakeOpenAppPlat{}
	params := map[string]any{"name": "Safari"}
	res, err := handleOpenApplication(context.Background(), p, params)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
}

func TestOpenApplication_MissingName(t *testing.T) {
	p := &fakeOpenAppPlat{}
	res, _ := handleOpenApplication(context.Background(), p, map[string]any{})
	if !res.IsError {
		t.Fatal("expected IsError when name is missing")
	}
}

func TestOpenApplication_PlatformError(t *testing.T) {
	p := &fakeOpenAppPlat{openErr: errors.New("application not found")}
	params := map[string]any{"name": "NonExistentApp"}
	res, _ := handleOpenApplication(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when OpenApplication fails")
	}
}
