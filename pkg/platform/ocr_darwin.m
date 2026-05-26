// ocr_darwin.m — Objective-C side of the macOS Vision-framework OCR
// path used by click_text / find_text_on_screen / screenshot_annotated
// / highlight_text_span. Built via cgo from ocr_darwin.go.
//
// Why a separate .m. cgo can call C and Objective-C, but mixing them
// inside a single Go-resident comment block makes the build flags
// (`-x objective-c`, framework links) brittle. A standalone .m gives
// Xcode-grade ARC + framework-link semantics with no surprises.
//
// API contract — VisionOCR_RecognizeText(pngBytes, pngLen):
//   * Input: PNG-encoded image bytes (the Go side already has these
//     via image/png.Encode — a one-time encode is much cheaper than
//     marshalling a Go image.Image into a CGImage cell-by-cell).
//   * Output: malloc'd C string containing one JSON-encoded
//     {"results":[…], "error":"…"} document. Caller frees with
//     VisionOCR_Free.
//   * Each result has: text, confidence (0..1), x, y, w, h in PIXELS
//     of the source image, with origin at TOP-LEFT (we flip Vision's
//     normalised bottom-left coordinate space for the caller).
//
// Recognition options are set for accuracy + automatic language
// (English + Chinese cover the metis user base; revision 3 is the
// 2023+ multilingual model and is the only one that handles CJK
// reliably).

#import <Foundation/Foundation.h>
#import <Vision/Vision.h>
#import <CoreImage/CoreImage.h>
#import <ImageIO/ImageIO.h>
#import <CoreGraphics/CoreGraphics.h>

#include <stdlib.h>
#include <string.h>

// Helper that builds a "{\"error\":\"...\"}" string we can hand back
// to Go when something fails before the OCR loop runs. Always returns
// a freshly malloc'd C string the caller must free.
static char *visionocr_make_error(NSString *reason) {
    NSDictionary *envelope = @{@"error": reason ?: @"unknown"};
    NSError *jsonErr = nil;
    NSData *data = [NSJSONSerialization dataWithJSONObject:envelope
                                                   options:0
                                                     error:&jsonErr];
    if (!data) {
        const char *fallback = "{\"error\":\"json_marshal_failed\"}";
        char *buf = (char *)malloc(strlen(fallback) + 1);
        strcpy(buf, fallback);
        return buf;
    }
    char *buf = (char *)malloc(data.length + 1);
    memcpy(buf, data.bytes, data.length);
    buf[data.length] = 0;
    return buf;
}

// VisionOCR_RecognizeText runs synchronous text recognition over the
// supplied PNG bytes. See the header docstring for the JSON shape.
// The function captures every error path (decode, request creation,
// recognition itself) into the same JSON envelope so the Go side
// only has one error-handling branch.
char *VisionOCR_RecognizeText(const unsigned char *pngBytes, int pngLen) {
    @autoreleasepool {
        if (pngBytes == NULL || pngLen <= 0) {
            return visionocr_make_error(@"empty input");
        }

        NSData *imgData = [NSData dataWithBytes:pngBytes length:(NSUInteger)pngLen];
        CGImageSourceRef src = CGImageSourceCreateWithData((__bridge CFDataRef)imgData, NULL);
        if (src == NULL) {
            return visionocr_make_error(@"image_source_create_failed");
        }
        CGImageRef cgImage = CGImageSourceCreateImageAtIndex(src, 0, NULL);
        CFRelease(src);
        if (cgImage == NULL) {
            return visionocr_make_error(@"cgimage_decode_failed");
        }

        // Capture dimensions before handing the image off — we need
        // them to denormalise Vision's [0,1] bounding boxes back to
        // source pixel coordinates.
        CGFloat imgW = (CGFloat)CGImageGetWidth(cgImage);
        CGFloat imgH = (CGFloat)CGImageGetHeight(cgImage);

        VNImageRequestHandler *handler =
            [[VNImageRequestHandler alloc] initWithCGImage:cgImage
                                                   options:@{}];

        VNRecognizeTextRequest *req = [[VNRecognizeTextRequest alloc] init];
        req.recognitionLevel = VNRequestTextRecognitionLevelAccurate;
        req.usesLanguageCorrection = YES;
        // Revision 3 (macOS 13+) is the first one that ships a real
        // multilingual model — required for Chinese / Japanese /
        // Korean text on metis-cu's target user base. Older revisions
        // silently strip CJK from results.
        if ([VNRecognizeTextRequest respondsToSelector:@selector(supportedRecognitionLanguagesForTextRecognitionLevel:revision:error:)]) {
            req.revision = VNRecognizeTextRequestRevision3;
        }
        // Set explicit language hints so the engine biases toward
        // mixed-script UI text. NB: setRecognitionLanguages: validates
        // each tag — passing an unsupported one throws on macOS < 13.
        // The supported set is intersected with availability below.
        NSMutableArray<NSString *> *wanted = [NSMutableArray array];
        if (@available(macOS 11.0, *)) {
            [wanted addObject:@"en-US"];
        }
        if (@available(macOS 13.0, *)) {
            // Chinese (Simplified) tag varies across releases; the
            // Vision docs settled on "zh-Hans" for revision 3.
            [wanted addObject:@"zh-Hans"];
            [wanted addObject:@"zh-Hant"];
            [wanted addObject:@"ja"];
            [wanted addObject:@"ko"];
        }
        if (wanted.count > 0) {
            req.recognitionLanguages = wanted;
        }

        NSError *err = nil;
        BOOL ok = [handler performRequests:@[req] error:&err];
        CGImageRelease(cgImage);
        if (!ok || err) {
            NSString *reason = err.localizedDescription ?: @"perform_requests_failed";
            return visionocr_make_error(reason);
        }

        NSMutableArray *out = [NSMutableArray array];
        for (VNRecognizedTextObservation *obs in req.results) {
            VNRecognizedText *best = [[obs topCandidates:1] firstObject];
            if (best == nil) continue;
            NSString *text = best.string ?: @"";
            CGFloat confidence = (CGFloat)best.confidence;

            // Vision returns boundingBox in normalised [0,1] with
            // origin at BOTTOM-LEFT. Convert to pixel coords with
            // origin at TOP-LEFT (Go image.Rectangle convention,
            // matches every other OCRResult producer in metis-cu).
            CGRect bb = obs.boundingBox;
            CGFloat px = bb.origin.x * imgW;
            CGFloat pw = bb.size.width * imgW;
            CGFloat py = (1.0 - bb.origin.y - bb.size.height) * imgH;
            CGFloat ph = bb.size.height * imgH;

            [out addObject:@{
                @"text": text,
                @"confidence": @(confidence),
                @"x": @((int)round(px)),
                @"y": @((int)round(py)),
                @"w": @((int)round(pw)),
                @"h": @((int)round(ph)),
            }];
        }

        NSDictionary *envelope = @{@"results": out};
        NSError *jsonErr = nil;
        NSData *jsonData = [NSJSONSerialization dataWithJSONObject:envelope
                                                           options:0
                                                             error:&jsonErr];
        if (!jsonData) {
            return visionocr_make_error(@"json_marshal_results_failed");
        }
        char *buf = (char *)malloc(jsonData.length + 1);
        memcpy(buf, jsonData.bytes, jsonData.length);
        buf[jsonData.length] = 0;
        return buf;
    }
}

// VisionOCR_Free releases a buffer returned by VisionOCR_RecognizeText.
// Separate function so the Go side doesn't have to import <stdlib.h>
// just for free(); keeps the cgo preamble minimal.
void VisionOCR_Free(char *buf) {
    if (buf != NULL) {
        free(buf);
    }
}
