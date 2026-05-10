//go:build linux

package platform

// Linux OCR backend — shells out to the `tesseract` binary if it's on
// PATH. Cheap, no cgo, no model download (Tesseract ships English by
// default on apt installs). When tesseract isn't installed the OCR
// method returns ErrNotImplemented so the caller surfaces a clean
// "OCR backend not available" message instead of a bare exec error.
//
// Output format: tesseract's `tsv` writer gives one line per word
// with `level conf left top width height text`. We aggregate at level
// 5 (word) — block / paragraph / line levels could be added later if
// callers want coarser regions.

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const ocrShellTimeout = 5 * time.Second

func (p *linuxPlatform) OCR(img image.Image) ([]OCRResult, error) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		return nil, fmt.Errorf("OCR: tesseract not on PATH (apt install tesseract-ocr): %w", ErrNotImplemented)
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return nil, fmt.Errorf("OCR: png encode: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), ocrShellTimeout)
	defer cancel()
	// `tesseract stdin stdout tsv` — read PNG from stdin, emit TSV
	// to stdout. `-c tessedit_create_tsv=1` is the default for the
	// tsv config but spelling it out keeps the call hermetic against
	// distro-packaged config edits.
	cmd := exec.CommandContext(ctx, "tesseract", "stdin", "stdout", "tsv")
	cmd.Stdin = &pngBuf
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("OCR: tesseract timed out after %s", ocrShellTimeout)
		}
		return nil, fmt.Errorf("OCR: tesseract: %w (%s)", err, strings.TrimSpace(errBuf.String()))
	}
	return parseTesseractTSV(out.String()), nil
}

// parseTesseractTSV extracts (text, bbox, conf) tuples from the
// word-level lines of tesseract's TSV output. Header row + non-word
// lines (level<5) + empty-text lines are skipped.
//
// Format columns (tab-separated):
//
//	level page block para line word left top width height conf text
func parseTesseractTSV(s string) []OCRResult {
	var out []OCRResult
	for i, line := range strings.Split(s, "\n") {
		if i == 0 || line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 12 {
			continue
		}
		level, err := strconv.Atoi(f[0])
		if err != nil || level != 5 {
			continue
		}
		text := strings.TrimSpace(f[11])
		if text == "" {
			continue
		}
		left, _ := strconv.Atoi(f[6])
		top, _ := strconv.Atoi(f[7])
		w, _ := strconv.Atoi(f[8])
		h, _ := strconv.Atoi(f[9])
		conf, _ := strconv.ParseFloat(f[10], 64)
		// Tesseract reports confidence 0..100 as int-ish strings.
		// Normalise to 0..1 for the OCRResult contract; -1 (no conf)
		// becomes 0.
		if conf < 0 {
			conf = 0
		} else {
			conf /= 100.0
		}
		out = append(out, OCRResult{
			Text:       text,
			Bounds:     image.Rect(left, top, left+w, top+h),
			Confidence: conf,
		})
	}
	return out
}
