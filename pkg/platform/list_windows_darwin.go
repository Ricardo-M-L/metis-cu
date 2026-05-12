//go:build darwin

package platform

// ListWindows on macOS — shells out to osascript and asks System
// Events to enumerate every visible window of every visible process.
// Same osascript path FrontmostApp / Confirm already use, so no new
// permission grant beyond what's already required.
//
// Output format: one window per line, tab-separated:
//
//	app<TAB>title<TAB>x<TAB>y<TAB>w<TAB>h
//
// Processes that throw on access (sandboxed daemons, helper apps)
// are skipped via the per-process try block — better than failing the
// whole enumeration on the first uncooperative process.

import (
	"context"
	"fmt"
	"image"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const listWindowsTimeout = 3 * time.Second

const listWindowsScript = `tell application "System Events"
	set output to {}
	repeat with proc in (every process whose visible is true)
		try
			set procName to name of proc
			repeat with win in (every window of proc)
				try
					set winTitle to name of win
					set pos to position of win
					set sz to size of win
					set end of output to procName & tab & winTitle & tab & (item 1 of pos) & tab & (item 2 of pos) & tab & (item 1 of sz) & tab & (item 2 of sz)
				end try
			end repeat
		end try
	end repeat
	set AppleScript's text item delimiters to linefeed
	return output as text
end tell`

func (p *darwinPlatform) ListWindows() ([]WindowInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listWindowsTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "osascript", "-e", listWindowsScript)
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("list_windows: osascript timed out after %s (System Events may be wedged)", listWindowsTimeout)
		}
		return nil, fmt.Errorf("list_windows: osascript: %w", err)
	}
	return parseListWindowsTSV(string(out)), nil
}

func parseListWindowsTSV(s string) []WindowInfo {
	var out []WindowInfo
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 6 {
			continue
		}
		x, errX := strconv.Atoi(strings.TrimSpace(f[2]))
		y, errY := strconv.Atoi(strings.TrimSpace(f[3]))
		w, errW := strconv.Atoi(strings.TrimSpace(f[4]))
		h, errH := strconv.Atoi(strings.TrimSpace(f[5]))
		if errX != nil || errY != nil || errW != nil || errH != nil {
			continue
		}
		out = append(out, WindowInfo{
			App:    strings.TrimSpace(f[0]),
			Title:  strings.TrimSpace(f[1]),
			Bounds: image.Rect(x, y, x+w, y+h),
		})
	}
	return out
}
