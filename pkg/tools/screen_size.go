package tools

// screen_size — Tier-1 borrow from Anthropic's official computer-use
// reference (`tools/computer.py:78-111` ComputerToolOptions). The
// reference declares `display_width_px` / `display_height_px` to the
// model so coordinate emissions land in the right canvas. metis-cu's
// downsampled screenshots (1280×800 cap) leave the click-coord space
// ambiguous; this tool fills that gap by exposing the LOGICAL pixel
// dimensions of every attached display, plus the active one's index
// and the current screenshot downsample cap.
//
// Output is JSON (DD-6 pattern, mirrors list_granted_applications) so
// the model can parse it deterministically rather than scraping a
// human sentence.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// displayInfo is the per-display entry in the screen_size JSON
// payload. Width / Height are LOGICAL pixels — the same coordinate
// space MouseMove / MouseClick / ScrollTo operate in. X / Y are the
// display's origin in the global virtual desktop (0,0 = primary
// upper-left; secondary monitors land at +primary.Width or similar
// per OS convention).
type displayInfo struct {
	Index  int `json:"index"`
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// screenSizeReport is the marshaled JSON shape. `displays` lists every
// attached monitor; `active` is the one Screenshot() will capture next
// (settable via switch_display). screenshot_max_w/h surface the
// downsample cap so the model can compute the scale factor between
// what it sees in PNG bytes and what coordinate space its clicks
// actually use (clicks are in display-native logical px, not PNG px).
type screenSizeReport struct {
	Active          int           `json:"active"`
	Displays        []displayInfo `json:"displays"`
	ScreenshotMaxW  int           `json:"screenshot_max_w"`
	ScreenshotMaxH  int           `json:"screenshot_max_h"`
	CoordSpaceNote  string        `json:"coord_space_note"`
	DisplayWidthPx  int           `json:"display_width_px"`  // active display
	DisplayHeightPx int           `json:"display_height_px"` // active display
}

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "screen_size",
			Description: "Return geometry of every attached display: per-display index/origin/width/height " +
				"(logical pixels, the same coord space mouse tools use), the currently active display index, " +
				"and the screenshot downsample cap. Call once at session start so the model knows what " +
				"canvas to emit click coordinates into — Anthropic's `display_width_px` / " +
				"`display_height_px` contract translated for the MCP wire.",
			Schema:  noArgsSchema(),
			Handler: handleScreenSize,
		})
	})
}

func handleScreenSize(ctx context.Context, plat platform.Platform, _ map[string]any) (*Result, error) {
	n, err := plat.DisplayCount()
	if err != nil {
		return &Result{Text: fmt.Sprintf("screen_size: %v", err), IsError: true}, nil
	}
	if n <= 0 {
		return &Result{Text: "screen_size: no active displays", IsError: true}, nil
	}
	displays := make([]displayInfo, 0, n)
	for i := 0; i < n; i++ {
		b, err := plat.DisplayBounds(i)
		if err != nil {
			// Skip a single bad display rather than failing the whole
			// call — the active one being healthy is what most callers
			// actually need.
			continue
		}
		displays = append(displays, displayInfo{
			Index:  i,
			X:      b.Min.X,
			Y:      b.Min.Y,
			Width:  b.Dx(),
			Height: b.Dy(),
		})
	}
	if len(displays) == 0 {
		return &Result{Text: "screen_size: no readable displays", IsError: true}, nil
	}
	// Active display: the one Screenshot() will capture next. Platform
	// keeps it as internal state; we don't expose a getter today, so
	// we report 0 (the default). When SwitchDisplay is added to the
	// model's available toolset this should grow a Platform.ActiveDisplay
	// accessor — punted to follow-up since the existing code path always
	// boots with activeDisplay=0.
	active := 0
	maxW, maxH := screenshotLimits(ctx)
	report := screenSizeReport{
		Active:          active,
		Displays:        displays,
		ScreenshotMaxW:  maxW,
		ScreenshotMaxH:  maxH,
		DisplayWidthPx:  displays[0].Width,
		DisplayHeightPx: displays[0].Height,
		CoordSpaceNote:  "Click coordinates are LOGICAL pixels of the active display — NOT pixels of the downsampled screenshot. If the screenshot was downsampled (image dim != display_width/height_px), scale up before emitting clicks.",
	}
	out, err := json.Marshal(report)
	if err != nil {
		return &Result{Text: fmt.Sprintf("screen_size: marshal: %v", err), IsError: true}, nil
	}
	return &Result{Text: string(out)}, nil
}
