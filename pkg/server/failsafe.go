package server

// failsafe.go — Tier-1 borrow from open-interpreter's corner-exit
// kill switch (`computer_use/loop.py:544-563`). A long autonomous
// computer-use session can wedge an app, click somewhere destructive,
// or just need an immediate stop with no MCP-protocol ceremony. This
// goroutine polls the cursor and, if the user shoves it into ANY
// screen corner and holds it there for `holdMs`, terminates the MCP
// server process.
//
// Independent of the model loop and the osascript Confirm dialog —
// this is a physical-world kill switch the user can trigger by
// flicking the mouse, no keyboard / focus / app-state required.
//
// Disabled by default ([failsafe] enabled = false). Opt-in via
// config.toml — the rest of the project leans toward "don't surprise
// the user" defaults, and a process-killing watchdog crosses that
// threshold.

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// failsafeConfig captures only what the watchdog needs from Config —
// keeps the goroutine ignorant of the wider config shape so a future
// section rename / refactor doesn't ripple here.
type failsafeConfig struct {
	enabled  bool
	pollMs   int
	holdMs   int
	cornerPx int
}

// failsafePollDefault / failsafeHoldDefault / failsafeCornerDefault
// are the baked-in fallbacks when the user enables the watchdog
// without specifying tuning. 100ms poll is slow enough to be
// negligible CPU but fast enough to feel responsive. 250ms hold
// avoids accidental triggers during normal pointer flicks. 5px
// corner zone is small enough that real cursor "rest at corner"
// during normal use never trips it.
const (
	failsafePollDefault   = 100
	failsafeHoldDefault   = 250
	failsafeCornerDefault = 5
)

// startFailsafeWatchdog kicks off the polling goroutine. Returns a
// cancel func; callers should defer it during graceful shutdown so
// the goroutine doesn't outlive the parent context. Does nothing
// (returns a no-op cancel) when the config has it disabled.
func startFailsafeWatchdog(plat platform.Platform, cfg failsafeConfig) func() {
	if !cfg.enabled {
		return func() {}
	}
	pollMs := cfg.pollMs
	if pollMs <= 0 {
		pollMs = failsafePollDefault
	}
	holdMs := cfg.holdMs
	if holdMs <= 0 {
		holdMs = failsafeHoldDefault
	}
	cornerPx := cfg.cornerPx
	if cornerPx <= 0 {
		cornerPx = failsafeCornerDefault
	}
	ctx, cancel := context.WithCancel(context.Background())
	go runFailsafe(ctx, plat, time.Duration(pollMs)*time.Millisecond, time.Duration(holdMs)*time.Millisecond, cornerPx)
	return cancel
}

// runFailsafe is the watchdog body. Exits cleanly on ctx cancel; on
// trigger logs to stderr and calls os.Exit(2) so the supervising
// process can distinguish a failsafe abort from a normal shutdown.
// Exit code 2 deliberately matches Unix "misuse of shell builtins"
// territory so monitoring tools that already split 0 / 1 / >1 land
// it in the "operator action" bucket.
func runFailsafe(ctx context.Context, plat platform.Platform, pollEvery, holdFor time.Duration, cornerPx int) {
	// Gather corners for EVERY display, not just the primary — otherwise
	// slamming the cursor into a corner of a secondary monitor (the user's
	// instinctive abort gesture) silently does nothing.
	type cornerBox struct{ minX, minY, maxX, maxY int }
	var boxes []cornerBox
	n, _ := plat.DisplayCount()
	if n < 1 {
		n = 1
	}
	for i := 0; i < n; i++ {
		if b, derr := plat.DisplayBounds(i); derr == nil {
			boxes = append(boxes, cornerBox{b.Min.X, b.Min.Y, b.Max.X - 1, b.Max.Y - 1})
		}
	}
	if len(boxes) == 0 {
		log.Printf("failsafe: cannot read any display bounds; watchdog disabled")
		return
	}

	t := time.NewTicker(pollEvery)
	defer t.Stop()

	var inCornerSince time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		pt, err := plat.CursorPosition()
		if err != nil {
			// Don't kill the server because the cursor probe glitched
			// — log once-per-second-ish (caller's tick rate) and
			// reset the hold timer so a false positive can't queue up.
			inCornerSince = time.Time{}
			continue
		}
		inCorner := false
		for _, b := range boxes {
			if isInAnyCorner(pt, b.minX, b.minY, b.maxX, b.maxY, cornerPx) {
				inCorner = true
				break
			}
		}
		if inCorner {
			if inCornerSince.IsZero() {
				inCornerSince = time.Now()
				continue
			}
			if time.Since(inCornerSince) >= holdFor {
				log.Printf("failsafe: cursor held in screen corner for %v — terminating metis-cu (exit 2)", holdFor)
				// Release platform handles before the hard exit — os.Exit
				// skips every deferred cleanup, which could otherwise strand
				// the model's just-typed text (a secret) on the clipboard or
				// leak OS handles / the CDP allocator.
				_ = plat.Close()
				os.Exit(2)
			}
			continue
		}
		inCornerSince = time.Time{}
	}
}

// isInAnyCorner returns true when (pt) lies within cornerPx of any of
// the four corners of the rectangle defined by (minX,minY)-(maxX,maxY).
// Pure func so unit tests don't need a Platform.
func isInAnyCorner(pt platform.Point, minX, minY, maxX, maxY, cornerPx int) bool {
	near := func(a, b int) bool {
		d := a - b
		if d < 0 {
			d = -d
		}
		return d <= cornerPx
	}
	corners := [4][2]int{
		{minX, minY},
		{maxX, minY},
		{minX, maxY},
		{maxX, maxY},
	}
	for _, c := range corners {
		if near(pt.X, c[0]) && near(pt.Y, c[1]) {
			return true
		}
	}
	return false
}
