package platform

// Shared screenshot post-processing — same on every OS, lives outside
// the build-tagged platform_*.go files so a single CatmullRom scaler
// is the source of truth. Keeps cgo / kbinani import scope clean too:
// only the per-OS core file knows the screenshot library exists.

import (
	"image"

	xdraw "golang.org/x/image/draw"
)

// normaliseToLogical resamples a captured image to logical pixel
// dimensions when the platform delivers a higher physical resolution
// (macOS Retina, Hi-DPI Linux / Windows scaling). Returns img unchanged
// when bounds already match — the common non-DPI-scaled case avoids an
// unnecessary alloc + Catmull-Rom pass.
//
// This fixes BUG-7: previously the screenshot tool returned a 2880x1800
// image for a 1440x900 logical display while MouseClick took 1440-space
// coords. The model would read coords from the captured PNG and emit
// clicks at 2x the intended position, missing every target on a Retina
// MBA. After this normalisation, the captured image's coordinate space
// matches what robotgo.Move expects — one source of truth.
func normaliseToLogical(img image.Image, logicalW, logicalH int) image.Image {
	if logicalW <= 0 || logicalH <= 0 {
		return img
	}
	b := img.Bounds()
	if b.Dx() == logicalW && b.Dy() == logicalH {
		return img
	}
	dst := image.NewRGBA(image.Rect(0, 0, logicalW, logicalH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
	return dst
}
