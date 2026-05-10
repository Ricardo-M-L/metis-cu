package tools

import (
	"context"
	"image"
	"strings"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type fakeHighlightPlat struct {
	fakeOCRPlat
	dragFrom platform.Point
	dragTo   platform.Point
	dragBtn  platform.Button
	dragHits int
	dragErr  error
}

func (p *fakeHighlightPlat) MouseDrag(_ context.Context, from, to platform.Point, btn platform.Button) error {
	p.dragFrom = from
	p.dragTo = to
	p.dragBtn = btn
	p.dragHits++
	return p.dragErr
}

func TestHighlightTextSpan_NormalCase(t *testing.T) {
	p := &fakeHighlightPlat{
		fakeOCRPlat: fakeOCRPlat{
			regions: []platform.OCRResult{
				{Text: "Dear", Bounds: image.Rect(10, 100, 60, 130)},
				{Text: "John,", Bounds: image.Rect(70, 100, 130, 130)},
				{Text: "It", Bounds: image.Rect(10, 200, 30, 230)},
				{Text: "was", Bounds: image.Rect(40, 200, 80, 230)},
				{Text: "great.", Bounds: image.Rect(90, 200, 150, 230)},
				{Text: "Sincerely,", Bounds: image.Rect(10, 300, 130, 330)},
			},
		},
	}
	res, err := handleHighlightTextSpan(context.Background(), p, map[string]any{
		"start_phrase": "Dear",
		"end_phrase":   "Sincerely",
	})
	if err != nil {
		t.Fatalf("transport: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if p.dragHits != 1 {
		t.Errorf("MouseDrag called %d times; want 1", p.dragHits)
	}
	// from = left edge of "Dear" at row centre y=115
	if p.dragFrom.X != 10 || p.dragFrom.Y != 115 {
		t.Errorf("dragFrom = %+v; want (10, 115)", p.dragFrom)
	}
	// to = right edge of "Sincerely," at row centre y=315
	if p.dragTo.X != 130 || p.dragTo.Y != 315 {
		t.Errorf("dragTo = %+v; want (130, 315)", p.dragTo)
	}
	if p.dragBtn != platform.ButtonLeft {
		t.Errorf("dragBtn = %v; want left", p.dragBtn)
	}
}

func TestHighlightTextSpan_StartNotFound(t *testing.T) {
	p := &fakeHighlightPlat{fakeOCRPlat: fakeOCRPlat{
		regions: []platform.OCRResult{{Text: "World", Bounds: image.Rect(0, 0, 50, 20)}},
	}}
	res, _ := handleHighlightTextSpan(context.Background(), p, map[string]any{
		"start_phrase": "Hello",
		"end_phrase":   "Bye",
	})
	if !res.IsError {
		t.Fatal("expected IsError when start_phrase not found")
	}
	if !strings.Contains(res.Text, "start_phrase") {
		t.Errorf("expected start-phrase phrasing in error; got %s", res.Text)
	}
	if p.dragHits != 0 {
		t.Errorf("MouseDrag called %d times; want 0 on start-not-found", p.dragHits)
	}
}

// TestHighlightTextSpan_EndBeforeStart: end_phrase appears in OCR but
// only BEFORE start_phrase — should error rather than drag-select
// backwards.
func TestHighlightTextSpan_EndBeforeStart(t *testing.T) {
	p := &fakeHighlightPlat{fakeOCRPlat: fakeOCRPlat{
		regions: []platform.OCRResult{
			{Text: "End", Bounds: image.Rect(0, 0, 30, 20)},
			{Text: "Start", Bounds: image.Rect(40, 0, 90, 20)},
		},
	}}
	res, _ := handleHighlightTextSpan(context.Background(), p, map[string]any{
		"start_phrase": "Start",
		"end_phrase":   "End",
	})
	if !res.IsError {
		t.Fatal("expected IsError when end appears only before start")
	}
}

func TestHighlightTextSpan_OCRBackendNotAvailable(t *testing.T) {
	p := &fakeHighlightPlat{fakeOCRPlat: fakeOCRPlat{ocrErr: platform.ErrNotImplemented}}
	res, _ := handleHighlightTextSpan(context.Background(), p, map[string]any{
		"start_phrase": "x",
		"end_phrase":   "y",
	})
	if !res.IsError {
		t.Fatal("expected IsError when OCR returns ErrNotImplemented")
	}
}
