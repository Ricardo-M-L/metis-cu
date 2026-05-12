package tools

import (
	"context"
	"encoding/json"
	"errors"
	"image"
	"strings"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type fakeListWindowsPlat struct {
	stubPlat
	wins []platform.WindowInfo
	err  error
}

func (p *fakeListWindowsPlat) ListWindows() ([]platform.WindowInfo, error) {
	return p.wins, p.err
}

func TestListWindows_NoFilter(t *testing.T) {
	p := &fakeListWindowsPlat{wins: []platform.WindowInfo{
		{App: "Safari", Title: "Tab 1 — Apple", Bounds: image.Rect(0, 0, 1440, 900)},
		{App: "Code", Title: "metis-cu", Bounds: image.Rect(100, 50, 1700, 1100)},
	}}
	res, err := handleListWindows(context.Background(), p, nil)
	if err != nil {
		t.Fatalf("transport: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	var rep listWindowReport
	if err := json.Unmarshal([]byte(res.Text), &rep); err != nil {
		t.Fatalf("Result.Text not JSON: %v\n%s", err, res.Text)
	}
	if rep.Total != 2 || len(rep.Windows) != 2 {
		t.Errorf("expected 2 windows; got total=%d, len=%d", rep.Total, len(rep.Windows))
	}
}

func TestListWindows_FilterMatchesAppOrTitle(t *testing.T) {
	p := &fakeListWindowsPlat{wins: []platform.WindowInfo{
		{App: "Safari", Title: "Apple", Bounds: image.Rect(0, 0, 100, 100)},
		{App: "Code", Title: "metis-cu — Phase I", Bounds: image.Rect(0, 0, 100, 100)},
		{App: "Terminal", Title: "ricardo@host", Bounds: image.Rect(0, 0, 100, 100)},
	}}
	// "code" matches App; expect 1 window.
	res, _ := handleListWindows(context.Background(), p, map[string]any{"query": "code"})
	var rep listWindowReport
	_ = json.Unmarshal([]byte(res.Text), &rep)
	if rep.Total != 1 || rep.Windows[0].App != "Code" {
		t.Errorf("filter on App failed: %+v", rep)
	}
	// "Phase" matches Title; expect 1 window.
	res, _ = handleListWindows(context.Background(), p, map[string]any{"query": "Phase"})
	_ = json.Unmarshal([]byte(res.Text), &rep)
	if rep.Total != 1 || rep.Windows[0].App != "Code" {
		t.Errorf("filter on Title failed: %+v", rep)
	}
}

func TestListWindows_BackendNotAvailable(t *testing.T) {
	p := &fakeListWindowsPlat{err: platform.ErrNotImplemented}
	res, _ := handleListWindows(context.Background(), p, nil)
	if !res.IsError {
		t.Fatal("expected IsError when backend returns ErrNotImplemented")
	}
	if !strings.Contains(res.Text, "backend not available") {
		t.Errorf("expected unavailable phrasing; got %s", res.Text)
	}
}

func TestListWindows_BackendError(t *testing.T) {
	p := &fakeListWindowsPlat{err: errors.New("wmctrl died")}
	res, _ := handleListWindows(context.Background(), p, nil)
	if !res.IsError {
		t.Fatal("expected IsError on backend error")
	}
}
