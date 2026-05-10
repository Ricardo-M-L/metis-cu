package tools

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type fakeOCRPlat struct {
	stubPlat
	ocrErr     error
	regions    []platform.OCRResult
	clickedAt  platform.Point
	clickedBtn platform.Button
	clickCount int
	shotErr    error
}

func (p *fakeOCRPlat) Screenshot() (image.Image, error) {
	if p.shotErr != nil {
		return nil, p.shotErr
	}
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	img.Set(0, 0, color.RGBA{R: 1, A: 255})
	return img, nil
}

func (p *fakeOCRPlat) OCR(_ image.Image) ([]platform.OCRResult, error) {
	if p.ocrErr != nil {
		return nil, p.ocrErr
	}
	return p.regions, nil
}

func (p *fakeOCRPlat) MouseClick(pt platform.Point, btn platform.Button, _ int) error {
	p.clickedAt = pt
	p.clickedBtn = btn
	p.clickCount++
	return nil
}

func TestClickText_SingleMatch_AutoClicks(t *testing.T) {
	p := &fakeOCRPlat{
		regions: []platform.OCRResult{
			{Text: "Submit", Bounds: image.Rect(100, 200, 200, 240), Confidence: 0.9},
			{Text: "Cancel", Bounds: image.Rect(300, 200, 400, 240), Confidence: 0.9},
		},
	}
	res, err := handleClickText(context.Background(), p, map[string]any{"text": "submit"})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if p.clickCount != 1 {
		t.Errorf("MouseClick called %d times; want 1", p.clickCount)
	}
	// Centre of (100,200)-(200,240) is (150, 220).
	if p.clickedAt.X != 150 || p.clickedAt.Y != 220 {
		t.Errorf("click landed at %+v; want (150, 220)", p.clickedAt)
	}
	if p.clickedBtn != platform.ButtonLeft {
		t.Errorf("click button = %v; want left", p.clickedBtn)
	}
}

// TestClickText_MultiMatch_ReturnsCandidates: ambiguous match returns
// IsError + a JSON list with rank indices the model uses to retry.
func TestClickText_MultiMatch_ReturnsCandidates(t *testing.T) {
	p := &fakeOCRPlat{
		regions: []platform.OCRResult{
			{Text: "OK", Bounds: image.Rect(10, 10, 30, 30)},
			{Text: "OK", Bounds: image.Rect(100, 100, 120, 120)},
			{Text: "OK", Bounds: image.Rect(200, 200, 220, 220)},
		},
	}
	res, err := handleClickText(context.Background(), p, map[string]any{"text": "OK"})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError on multi-match; got %s", res.Text)
	}
	if p.clickCount != 0 {
		t.Errorf("MouseClick called %d times on ambiguous match; want 0", p.clickCount)
	}
	var amb clickTextAmbiguous
	if err := json.Unmarshal([]byte(res.Text), &amb); err != nil {
		t.Fatalf("Result.Text not JSON: %v\n--- raw:\n%s", err, res.Text)
	}
	if len(amb.Candidates) != 3 {
		t.Fatalf("candidates len = %d; want 3", len(amb.Candidates))
	}
	for i, c := range amb.Candidates {
		if c.Rank != i {
			t.Errorf("candidate[%d].Rank = %d; want %d", i, c.Rank, i)
		}
	}
}

// TestClickText_MultiMatch_ExplicitOccurrence: caller passes
// occurrence=1 → click the second match without ambiguity error.
func TestClickText_MultiMatch_ExplicitOccurrence(t *testing.T) {
	p := &fakeOCRPlat{
		regions: []platform.OCRResult{
			{Text: "OK", Bounds: image.Rect(10, 10, 30, 30)},
			{Text: "OK", Bounds: image.Rect(100, 100, 120, 120)},
		},
	}
	res, _ := handleClickText(context.Background(), p, map[string]any{"text": "OK", "occurrence": 1})
	if res.IsError {
		t.Fatalf("expected success with explicit occurrence; got %s", res.Text)
	}
	if p.clickedAt.X != 110 || p.clickedAt.Y != 110 {
		t.Errorf("click landed at %+v; want centre of second match (110, 110)", p.clickedAt)
	}
}

func TestClickText_NoMatch(t *testing.T) {
	p := &fakeOCRPlat{regions: []platform.OCRResult{{Text: "Submit", Bounds: image.Rect(0, 0, 10, 10)}}}
	res, _ := handleClickText(context.Background(), p, map[string]any{"text": "nope"})
	if !res.IsError {
		t.Fatal("expected IsError on no-match")
	}
	if !strings.Contains(res.Text, "no on-screen text matched") {
		t.Errorf("expected no-match phrasing; got %s", res.Text)
	}
}

func TestClickText_OCRBackendNotAvailable(t *testing.T) {
	p := &fakeOCRPlat{ocrErr: platform.ErrNotImplemented}
	res, _ := handleClickText(context.Background(), p, map[string]any{"text": "Submit"})
	if !res.IsError {
		t.Fatal("expected IsError when OCR returns ErrNotImplemented")
	}
	if !strings.Contains(res.Text, "OCR backend not available") {
		t.Errorf("expected platform-not-supported phrasing; got %s", res.Text)
	}
}

func TestClickText_OccurrenceOutOfRange(t *testing.T) {
	p := &fakeOCRPlat{regions: []platform.OCRResult{{Text: "Submit", Bounds: image.Rect(0, 0, 10, 10)}}}
	res, _ := handleClickText(context.Background(), p, map[string]any{"text": "Submit", "occurrence": 5})
	if !res.IsError {
		t.Fatal("expected IsError when occurrence >= match count")
	}
}

// TestClickText_ButtonRoute: button=right routes to MouseClick with
// platform.ButtonRight.
func TestClickText_ButtonRoute(t *testing.T) {
	p := &fakeOCRPlat{regions: []platform.OCRResult{{Text: "Menu", Bounds: image.Rect(50, 50, 100, 80)}}}
	_, _ = handleClickText(context.Background(), p, map[string]any{"text": "Menu", "button": "right"})
	if p.clickedBtn != platform.ButtonRight {
		t.Errorf("button routing failed: got %v, want right", p.clickedBtn)
	}
}

func TestFindTextOnScreen_FilterAndPagination(t *testing.T) {
	p := &fakeOCRPlat{
		regions: []platform.OCRResult{
			{Text: "Hello", Bounds: image.Rect(0, 0, 50, 20)},
			{Text: "World", Bounds: image.Rect(60, 0, 110, 20)},
			{Text: "Hello", Bounds: image.Rect(0, 30, 50, 50)},
		},
	}
	res, err := handleFindTextOnScreen(context.Background(), p, map[string]any{"query": "hello"})
	if err != nil {
		t.Fatalf("transport: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	var rep findTextReport
	if err := json.Unmarshal([]byte(res.Text), &rep); err != nil {
		t.Fatalf("Result.Text not JSON: %v\n%s", err, res.Text)
	}
	if rep.Total != 2 || len(rep.Regions) != 2 {
		t.Fatalf("expected 2 hits; got total=%d len=%d body=%s", rep.Total, len(rep.Regions), res.Text)
	}
	for _, r := range rep.Regions {
		if r.Text != "Hello" {
			t.Errorf("filter leaked: %q in result", r.Text)
		}
	}
}

func TestFindTextOnScreen_NoQueryReturnsAll(t *testing.T) {
	p := &fakeOCRPlat{
		regions: []platform.OCRResult{
			{Text: "A", Bounds: image.Rect(0, 0, 10, 10)},
			{Text: "B", Bounds: image.Rect(0, 0, 10, 10)},
			{Text: "C", Bounds: image.Rect(0, 0, 10, 10)},
		},
	}
	res, _ := handleFindTextOnScreen(context.Background(), p, nil)
	var rep findTextReport
	_ = json.Unmarshal([]byte(res.Text), &rep)
	if rep.Total != 3 {
		t.Errorf("expected 3 regions with no query; got %d", rep.Total)
	}
}

func TestFindTextOnScreen_OCRBackendNotAvailable(t *testing.T) {
	p := &fakeOCRPlat{ocrErr: platform.ErrNotImplemented}
	res, _ := handleFindTextOnScreen(context.Background(), p, map[string]any{"query": "x"})
	if !res.IsError {
		t.Fatal("expected IsError when OCR returns ErrNotImplemented")
	}
}
