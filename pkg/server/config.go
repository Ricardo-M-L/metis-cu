package server

// Runtime config loaded from $HOME/.metis-cu/config.toml. Keeping the
// loader in pkg/server (rather than pkg/tools) so the tool layer
// stays free of file-system / TOML dependencies — pkg/server is the
// only caller that knows where the file lives and when to read it
// (once at startup).
//
// Schema is deliberately tiny — only the knobs we've actually been
// asked for. Add new sections here as the project grows; do NOT
// silently expand the TOML to surface every internal default, or the
// config file will rot into bit-rot territory.

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config is the parsed shape of ~/.metis-cu/config.toml. Top-level
// sections map onto subsystems; field tags use snake_case so the TOML
// keys stay terminal-friendly.
type Config struct {
	Screenshot ScreenshotConfig `toml:"screenshot"`
	Keyboard   KeyboardConfig   `toml:"keyboard"`
	Mouse      MouseConfig      `toml:"mouse"`
	Gate       GateConfig       `toml:"gate"`
	Limits     LimitsConfig     `toml:"limits"`
	Failsafe   FailsafeConfig   `toml:"failsafe"`
}

// FailsafeConfig — opt-in corner-exit kill switch (Tier-1 borrow from
// open-interpreter). Disabled by default; the rest of the project
// avoids surprise process-killers. Enable when you want a physical-
// world stop button independent of the model loop.
type FailsafeConfig struct {
	Enabled  bool `toml:"enabled"`
	PollMs   int  `toml:"poll_ms"`
	HoldMs   int  `toml:"hold_ms"`
	CornerPx int  `toml:"corner_px"`
}

// ScreenshotConfig caps the per-call image dimensions before base64
// encoding. Defaults to 1280×800 (Anthropic CU recommendation) when
// the user hasn't set the keys; non-positive values are treated as
// "fall back to default" so a stub-out config can't accidentally
// produce a 0×0 image.
//
// Format chooses the wire encoding:
//
//	"jpeg" (DEFAULT as of 2026-05-26) — 5-10× smaller than PNG at
//	    visually-identical quality on typical UI screenshots; keeps
//	    base64 payload well under the metis context-overflow snipper
//	    threshold (~150 KB ≈ 35-40k tokens). The default-quality
//	    q=85 was empirically picked by Anthropic's computer-use-demo
//	    after testing OCR accuracy on UI text.
//	"png" — lossless. Use only when 1-pixel edge fidelity matters
//	    (rare in cu workloads; tested OCR doesn't gain meaningfully).
//	    A 1280×800 PNG of a typical UI is ~400-800 KB → ~130k+ tokens,
//	    which forces metis's emergency snipper to truncate every
//	    tool_result. Session 87e366f post-mortem (2026-05-26):
//	    100+ screenshot calls all snipped, model never saw a clean
//	    frame, failed to land a click for 18 min.
//
// Anything other than "jpeg" / "png" falls back to "jpeg". Quality
// applies only when format=jpeg and clamps to [1,100].
type ScreenshotConfig struct {
	MaxWidth  int    `toml:"max_width"`
	MaxHeight int    `toml:"max_height"`
	Format    string `toml:"format"`
	Quality   int    `toml:"quality"`
}

// KeyboardConfig governs the `type` tool's switching point between
// per-key synthetic events and clipboard-paste fallback (BUG-22).
// Default is 80 runes — below that the per-key path is reliable
// enough; above that, robotgo.TypeStr starts dropping characters on
// macOS under load. Lower the value to be more aggressive about
// pasting; raise it to keep typing visible per-character (useful for
// recording demos). hold_max_ms caps hold_key duration (BUG-11).
type KeyboardConfig struct {
	TypePasteThreshold int `toml:"type_paste_threshold"`
	HoldMaxMs          int `toml:"hold_max_ms"`
}

// MouseConfig — robotgo.MoveSmooth easing during MouseDrag. low/high
// > 1 = slower, more "human"; < 1 = brisker. Default 1.0/1.0.
//
// SettleMs is the post-action quiescence delay (Tier-1 borrow from
// Anthropic's reference: `_screenshot_delay = 2.0s`). After every
// successful mouse / keyboard action we sleep SettleMs milliseconds
// before returning, so the next screenshot the model takes captures
// the post-animation steady state rather than mid-transition. 0 =
// disabled (default — matches pre-Tier-1 behaviour). Common values:
// 250–500 for snappy UIs, 1000+ for animation-heavy apps.
type MouseConfig struct {
	SmoothLow  float64 `toml:"smooth_low"`
	SmoothHigh float64 `toml:"smooth_high"`
	SettleMs   int     `toml:"settle_ms"`
}

// GateConfig — frontmost-app probe knobs (DD-3 + DD-4). timeout is
// how long we wait for osascript / xdotool before giving up; cache
// TTL is how long we coalesce repeat lookups across a burst of gated
// calls.
//
// HostTerminalTier (added 2026-05-26) opts the host-terminal apps
// (Terminal / iTerm2 / Ghostty / WezTerm / Alacritty / kitty / Hyper /
// Tabby) into a fixed tier regardless of the hard-coded TierClick
// default. Targets the metis case: an MCP client running INSIDE a
// terminal gets its `open_application` calls rejected because the
// frontmost app is iTerm2 (TierClick) and the call needs TierFull.
// Set to "full" to let metis drive `open_application` / `type` etc.
// while the terminal is in front. Empty / unrecognised value keeps
// the historical behaviour.
//
// Can also be set per-run via the METIS_CU_HOST_TERMINAL_TIER env var
// (env wins over config — useful when metis spawns metis-cu and wants
// to opt in without writing to the user's TOML).
type GateConfig struct {
	FrontmostTimeoutMs  int    `toml:"frontmost_timeout_ms"`
	FrontmostCacheTtlMs int    `toml:"frontmost_cache_ttl_ms"`
	HostTerminalTier    string `toml:"host_terminal_tier"`
}

// LimitsConfig — anti-OOM / anti-DoS caps for tools that take
// model-supplied size inputs. All defaults match the bare-const
// values previously hardcoded in pkg/tools.
type LimitsConfig struct {
	ClipboardMaxBytes   int     `toml:"clipboard_max_bytes"`
	BatchMaxSteps       int     `toml:"batch_max_steps"`
	ZoomMaxFactor       float64 `toml:"zoom_max_factor"`
	ZoomMaxOutputPixels int     `toml:"zoom_max_output_pixels"`
}

// DefaultConfig returns the baked-in defaults applied before any TOML
// load. Exported so tests can compare against a known-clean value.
func DefaultConfig() Config {
	return Config{
		Screenshot: ScreenshotConfig{
			MaxWidth:  1280,
			MaxHeight: 800,
			// JPEG default (was PNG until 2026-05-26): PNG payloads
			// blow past the metis context-overflow snipper threshold
			// on every cu call, leaving the model with truncated
			// frames it can't act on. JPEG q=85 keeps the same UI
			// readability for ~5-10× less wire bytes.
			Format:  "jpeg",
			Quality: 85,
		},
		Keyboard: KeyboardConfig{
			TypePasteThreshold: 80,
			HoldMaxMs:          10000,
		},
		Mouse: MouseConfig{
			SmoothLow:  1.0,
			SmoothHigh: 1.0,
		},
		Gate: GateConfig{
			FrontmostTimeoutMs:  1500,
			FrontmostCacheTtlMs: 250,
		},
		Limits: LimitsConfig{
			ClipboardMaxBytes:   64 * 1024,
			BatchMaxSteps:       32,
			ZoomMaxFactor:       16.0,
			ZoomMaxOutputPixels: 16_000_000,
		},
	}
}

// LoadConfig reads $HOME/.metis-cu/config.toml and merges it onto the
// defaults. Missing file → defaults verbatim (no error: the file is
// optional). Malformed TOML or unreadable file → defaults plus the
// underlying error so the caller can decide whether to surface a
// warning or proceed silently. The MCP server currently proceeds
// silently (the alternative — refusing to boot on a typo — is
// hostile).
func LoadConfig() (Config, error) {
	c := DefaultConfig()
	home, err := os.UserHomeDir()
	if err != nil {
		return c, err
	}
	path := filepath.Join(home, ".metis-cu", "config.toml")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return c, err
	}
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return c, err
	}
	// Sanitise non-positive values back to defaults so a partial
	// stanza (only max_width set) doesn't zero out the other axis.
	if c.Screenshot.MaxWidth <= 0 {
		c.Screenshot.MaxWidth = 1280
	}
	if c.Screenshot.MaxHeight <= 0 {
		c.Screenshot.MaxHeight = 800
	}
	switch c.Screenshot.Format {
	case "png", "jpeg":
		// valid
	default:
		// 2026-05-26: unknown format collapses to jpeg (was png).
		// Matches DefaultConfig — jpeg is the only payload size that
		// keeps base64 below the metis context-overflow threshold.
		c.Screenshot.Format = "jpeg"
	}
	if c.Screenshot.Quality < 1 || c.Screenshot.Quality > 100 {
		c.Screenshot.Quality = 85
	}
	if c.Keyboard.TypePasteThreshold <= 0 {
		c.Keyboard.TypePasteThreshold = 80
	}
	if c.Keyboard.HoldMaxMs <= 0 {
		c.Keyboard.HoldMaxMs = 10000
	}
	if c.Mouse.SmoothLow <= 0 {
		c.Mouse.SmoothLow = 1.0
	}
	if c.Mouse.SmoothHigh <= 0 {
		c.Mouse.SmoothHigh = 1.0
	}
	if c.Mouse.SettleMs < 0 {
		c.Mouse.SettleMs = 0
	}
	if c.Gate.FrontmostTimeoutMs <= 0 {
		c.Gate.FrontmostTimeoutMs = 1500
	}
	if c.Gate.FrontmostCacheTtlMs <= 0 {
		c.Gate.FrontmostCacheTtlMs = 250
	}
	if c.Limits.ClipboardMaxBytes <= 0 {
		c.Limits.ClipboardMaxBytes = 64 * 1024
	}
	if c.Limits.BatchMaxSteps <= 0 {
		c.Limits.BatchMaxSteps = 32
	}
	if c.Limits.ZoomMaxFactor <= 0 {
		c.Limits.ZoomMaxFactor = 16.0
	}
	if c.Limits.ZoomMaxOutputPixels <= 0 {
		c.Limits.ZoomMaxOutputPixels = 16_000_000
	}
	return c, nil
}
