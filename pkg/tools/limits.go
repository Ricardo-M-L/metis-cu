package tools

// Per-handler accessor for the runtime-configurable limits stored on
// the Registry (DD-3). All helpers follow the same pattern:
//   1. Pull the active *Registry off ctx via registryKey
//   2. If present and the relevant field is positive, return it
//   3. Otherwise fall back to the baked-in package default
//
// Helpers live here (one file, all together) rather than co-located
// with each handler so a future "list every tunable" audit only has
// to scan one place.

import "context"

// Defaults — exposed so direct unit tests can reference the canonical
// value when the registry isn't on the ctx.
const (
	DefaultHoldKeyMaxMs        = 10000
	DefaultClipboardMaxBytes   = 64 * 1024
	DefaultBatchMaxSteps       = 32
	DefaultZoomMaxFactor       = 16.0
	DefaultZoomMaxOutputPixels = 16_000_000
)

func limitsFromCtx(ctx context.Context) *Registry {
	if reg, ok := ctx.Value(registryKey{}).(*Registry); ok {
		return reg
	}
	return nil
}

func holdKeyMaxMsFor(ctx context.Context) int {
	if r := limitsFromCtx(ctx); r != nil && r.HoldKeyMaxMs > 0 {
		return r.HoldKeyMaxMs
	}
	return DefaultHoldKeyMaxMs
}

func clipboardMaxBytesFor(ctx context.Context) int {
	if r := limitsFromCtx(ctx); r != nil && r.ClipboardMaxBytes > 0 {
		return r.ClipboardMaxBytes
	}
	return DefaultClipboardMaxBytes
}

func batchMaxStepsFor(ctx context.Context) int {
	if r := limitsFromCtx(ctx); r != nil && r.BatchMaxSteps > 0 {
		return r.BatchMaxSteps
	}
	return DefaultBatchMaxSteps
}

func zoomMaxFactorFor(ctx context.Context) float64 {
	if r := limitsFromCtx(ctx); r != nil && r.ZoomMaxFactor > 0 {
		return r.ZoomMaxFactor
	}
	return DefaultZoomMaxFactor
}

func zoomMaxOutputPixelsFor(ctx context.Context) int {
	if r := limitsFromCtx(ctx); r != nil && r.ZoomMaxOutputPixels > 0 {
		return r.ZoomMaxOutputPixels
	}
	return DefaultZoomMaxOutputPixels
}
