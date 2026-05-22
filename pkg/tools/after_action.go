package tools

// after_action.go — Tier-1 borrow from Anthropic's reference computer
// -use loop. Two related opt-ins applied at the end of every successful
// action handler:
//
//  1. SETTLE delay (`[mouse] settle_ms`): sleep N ms so animations
//     finish before the next call. Defaults to 0 (off) — Anthropic's
//     2.0s default is too aggressive for snappy UIs but a 250–500ms
//     value dramatically reduces "saw the spinner, clicked the wrong
//     thing" mis-clicks.
//  2. return_screenshot=true (per-call param): grab a fresh screenshot
//     immediately after the action and return it inline so the caller
//     saves a round-trip. Honours the same downsample + format/quality
//     pipeline `screenshot` uses, so callers see consistent bytes.
//
// The two are independent: settle without screenshot is the most
// common "give the UI a moment" pattern; screenshot without settle
// works but usually wants at least a tiny settle to avoid mid-animation
// frames.

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// mouseSettleFor pulls the post-action delay from the active Registry
// (DD-3 pattern). Falls back to 0 when no Registry is bound, which
// matches the package default — the helper is genuinely opt-in.
func mouseSettleFor(ctx context.Context) time.Duration {
	if reg, ok := ctx.Value(registryKey{}).(*Registry); ok && reg != nil {
		if reg.MouseSettleMs > 0 {
			return time.Duration(reg.MouseSettleMs) * time.Millisecond
		}
	}
	return 0
}

// settleAndMaybeShot is the canonical end-of-handler hook. Returns
// (image base64, mime). Both are empty strings when the caller didn't
// request a screenshot AND no encode failure happened — handlers that
// want to attach the result do so unconditionally:
//
//	img, mime := settleAndMaybeShot(ctx, plat, params)
//	return &Result{Text: msg, Image: img, MIMEType: mime}, nil
//
// The screenshot path silently swallows errors (returns empty strings)
// because a settle/return-shot failure shouldn't promote an otherwise-
// successful click to IsError. The model still sees the success Text
// and can call `screenshot` explicitly if it needs the visual.
func settleAndMaybeShot(ctx context.Context, plat platform.Platform, params map[string]any) (string, string) {
	if d := mouseSettleFor(ctx); d > 0 {
		t := time.NewTimer(d)
		select {
		case <-ctx.Done():
			t.Stop()
			return "", ""
		case <-t.C:
		}
	}
	// 2026-05-22: default flipped false → true for state-changing
	// tools. Pre-fix the model had to opt-in by passing
	// `return_screenshot=true` and most models forgot, then made an
	// extra round-trip with a standalone Screenshot call to see what
	// just happened. Auto-attaching the post-action frame mirrors
	// Anthropic's reference cu loop and cuts the model's wall-time
	// roughly in half on multi-step tasks.
	//
	// Cost trade: +1 image per state-changing call (~10K vision
	// tokens), but -1 standalone Screenshot call. Net: model saves
	// 30-40% wall time, token cost goes up ~10% — that's the
	// intentional speed/cost trade. Set `return_screenshot=false`
	// explicitly when batching unattended clicks where intermediate
	// frames aren't useful.
	rs, _ := optionalBool(params, "return_screenshot", true)
	if !rs {
		return "", ""
	}
	img, err := plat.Screenshot()
	if err != nil {
		return "", ""
	}
	maxW, maxH := screenshotLimits(ctx)
	img = downsampleScreenshot(img, maxW, maxH)
	format, quality := screenshotFormat(ctx)
	buf, mime, err := encodeScreenshot(img, format, quality)
	if err != nil {
		return "", ""
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), mime
}

// returnScreenshotSchema is the shared JSON-Schema property fragment
// that handlers add to their `properties` map so additionalProperties
// =false schemas don't reject the field.
func returnScreenshotSchema() map[string]any {
	return map[string]any{
		"type":        "boolean",
		"default":     true,
		"description": "Default true: a post-action screenshot is auto-attached so the model sees the result without a separate Screenshot call. Set to false ONLY when batching unattended actions where intermediate frames waste tokens (e.g. rapid sequential clicks at known coords).",
	}
}
