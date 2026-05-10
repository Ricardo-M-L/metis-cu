package tools

import (
	"context"
	"encoding/json"
	"errors"
	"image"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type fakeScreenSizePlat struct {
	stubPlat
	count       int
	bounds      map[int]image.Rectangle
	countErr    error
	boundsErrAt map[int]error
}

func (p *fakeScreenSizePlat) DisplayCount() (int, error) {
	if p.countErr != nil {
		return 0, p.countErr
	}
	return p.count, nil
}

func (p *fakeScreenSizePlat) DisplayBounds(idx int) (image.Rectangle, error) {
	if e, ok := p.boundsErrAt[idx]; ok {
		return image.Rectangle{}, e
	}
	if r, ok := p.bounds[idx]; ok {
		return r, nil
	}
	return image.Rectangle{}, errors.New("no such display")
}

func TestScreenSize_OK(t *testing.T) {
	p := &fakeScreenSizePlat{
		count: 2,
		bounds: map[int]image.Rectangle{
			0: image.Rect(0, 0, 1440, 900),
			1: image.Rect(1440, 0, 1440+1920, 1080),
		},
	}
	res, err := handleScreenSize(context.Background(), p, nil)
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	var got screenSizeReport
	if err := json.Unmarshal([]byte(res.Text), &got); err != nil {
		t.Fatalf("Result.Text not JSON: %v\n--- raw:\n%s", err, res.Text)
	}
	if len(got.Displays) != 2 {
		t.Fatalf("displays len: got %d, want 2; raw: %s", len(got.Displays), res.Text)
	}
	if got.Displays[0].Width != 1440 || got.Displays[0].Height != 900 {
		t.Errorf("display[0] = %+v, want 1440x900", got.Displays[0])
	}
	if got.Displays[1].X != 1440 || got.Displays[1].Width != 1920 {
		t.Errorf("display[1] = %+v, want X=1440 Width=1920", got.Displays[1])
	}
	if got.DisplayWidthPx != 1440 {
		t.Errorf("DisplayWidthPx = %d, want 1440 (active display)", got.DisplayWidthPx)
	}
	if got.ScreenshotMaxW != DefaultScreenshotMaxW {
		t.Errorf("ScreenshotMaxW = %d, want default %d", got.ScreenshotMaxW, DefaultScreenshotMaxW)
	}
	if got.CoordSpaceNote == "" {
		t.Error("CoordSpaceNote should not be empty — the model needs the canvas hint")
	}
}

func TestScreenSize_NoDisplays(t *testing.T) {
	p := &fakeScreenSizePlat{count: 0}
	res, _ := handleScreenSize(context.Background(), p, nil)
	if !res.IsError {
		t.Fatal("expected IsError when DisplayCount returns 0")
	}
}

func TestScreenSize_DisplayCountError(t *testing.T) {
	p := &fakeScreenSizePlat{countErr: platform.ErrNotImplemented}
	res, _ := handleScreenSize(context.Background(), p, nil)
	if !res.IsError {
		t.Fatal("expected IsError when DisplayCount errors")
	}
}

// TestScreenSize_PartialBoundsFailure: one display can't be probed
// but the others succeed — the call still returns a usable JSON
// instead of failing the whole tool.
func TestScreenSize_PartialBoundsFailure(t *testing.T) {
	p := &fakeScreenSizePlat{
		count: 2,
		bounds: map[int]image.Rectangle{
			0: image.Rect(0, 0, 1024, 768),
		},
		boundsErrAt: map[int]error{1: errors.New("display dropped off the bus")},
	}
	res, err := handleScreenSize(context.Background(), p, nil)
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected partial success, got IsError: %s", res.Text)
	}
	var got screenSizeReport
	_ = json.Unmarshal([]byte(res.Text), &got)
	if len(got.Displays) != 1 || got.Displays[0].Width != 1024 {
		t.Errorf("expected 1 display 1024 wide; got %+v", got.Displays)
	}
}
