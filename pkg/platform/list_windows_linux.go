//go:build linux

package platform

// ListWindows on Linux — shells out to `wmctrl -lpG` and parses its
// space-separated output:
//
//	0x05400003 -1 1234 0 0 1920 1080 hostname app-class - Title
//	         ^   ^  ^  ^ ^ ^    ^    ^        ^         ^
//	         id  ws pid x y w   h    host     class     title (rest)
//
// wmctrl ships in most desktop distros (apt install wmctrl). When
// it's not on PATH we return ErrNotImplemented so the caller surfaces
// a clean "list_windows backend not available" message.
//
// A future enhancement would use raw X11 ICCCM properties directly,
// removing the wmctrl dependency at the cost of ~150 lines of XCB
// marshaling. Shell-out is the pragmatic baseline.

import (
	"bufio"
	"context"
	"fmt"
	"image"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func (p *linuxPlatform) ListWindows() ([]WindowInfo, error) {
	if _, err := exec.LookPath("wmctrl"); err != nil {
		return nil, fmt.Errorf("list_windows: wmctrl not on PATH (apt install wmctrl): %w", ErrNotImplemented)
	}
	ctx, cancel := context.WithTimeout(context.Background(), listWindowsLinuxTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "wmctrl", "-lpG")
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("list_windows: wmctrl timed out after %s", listWindowsLinuxTimeout)
		}
		return nil, fmt.Errorf("list_windows: wmctrl: %w", err)
	}
	return parseWmctrlOutput(string(out)), nil
}

const listWindowsLinuxTimeout = 2 * time.Second

// parseWmctrlOutput splits each line into fields with wmctrl's
// 8-field "id workspace pid x y w h host" prefix, then takes
// everything else as `class - title` joined by ` - `. Lines that
// don't parse are silently skipped — wmctrl occasionally emits
// non-window status lines on some distros and we'd rather miss one
// than fail the whole enumeration.
func parseWmctrlOutput(s string) []WindowInfo {
	var out []WindowInfo
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		x, errX := strconv.Atoi(fields[3])
		y, errY := strconv.Atoi(fields[4])
		w, errW := strconv.Atoi(fields[5])
		h, errH := strconv.Atoi(fields[6])
		if errX != nil || errY != nil || errW != nil || errH != nil {
			continue
		}
		app := fields[8]
		// Title is everything past the 9th field; split on first " - "
		// since wmctrl uses that as the class/title separator on most
		// WMs (mutter, kwin). If absent, treat the whole tail as the
		// title and leave app=class.
		var title string
		if len(fields) > 9 {
			rest := strings.Join(fields[9:], " ")
			if i := strings.Index(rest, "- "); i >= 0 {
				title = strings.TrimSpace(rest[i+2:])
			} else {
				title = strings.TrimSpace(rest)
			}
		}
		out = append(out, WindowInfo{
			App:    app,
			Title:  title,
			Bounds: image.Rect(x, y, x+w, y+h),
		})
	}
	return out
}
