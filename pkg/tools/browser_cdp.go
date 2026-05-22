package tools

// browser_cdp.go — Chrome DevTools Protocol bridge. When the user
// launches Chrome with `--remote-debugging-port=9222`, these tools
// connect to the live page and operate it via CSS selectors instead
// of pixel-based OCR + click. Per browser-use's 70k-star validation,
// DOM-based browser automation is ~5x faster and ~70% fewer tokens
// than vision+coords on browser tasks.
//
// Trade: only works for browsers, and only when the browser was
// started with the debug port. metis-cu's pixel-based tools remain
// the universal fallback for the desktop / non-browser case.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// cdpDefaultPort is the conventional Chrome debug port. Users override
// per-call with `port` if they ran Chrome on a different one.
const cdpDefaultPort = 9222

// cdpAllocator caches the chromedp remote allocator so successive
// browser_* tool calls reuse the same WebSocket connection instead of
// re-handshaking each round-trip. Per process; one cache key per
// (port). Closed on process exit (deliberately — no explicit close
// needed since the parent metis-cu binary holds the connection).
var (
	cdpAllocatorMu    sync.Mutex
	cdpAllocatorCache = map[int]context.Context{}
	cdpCancelCache    = map[int]context.CancelFunc{}
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name:        "browser_dom_outline",
			Description: "Read the active Chrome tab's DOM as a compact text outline. Requires Chrome started with --remote-debugging-port=9222. Returns interactive elements (links, buttons, inputs) with their CSS selector and visible text — use the selector directly in browser_click / browser_type instead of guessing pixel coords. 5x faster + 70% fewer tokens than screenshot+OCR for browser tasks. Falls back to a clear error when the debug port is unreachable, so the model knows to use the pixel-based tools instead.",
			Schema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]any{
					"port": map[string]any{
						"type":        "integer",
						"description": "Chrome --remote-debugging-port to connect to. Default 9222.",
						"default":     cdpDefaultPort,
					},
					"max_elements": map[string]any{
						"type":        "integer",
						"description": "Cap on interactive elements returned. Default 60 — keeps the outline under ~3KB for the LLM.",
						"default":     60,
						"minimum":     1,
					},
				},
			},
			Handler: handleBrowserDOMOutline,
		})

		r.register(Spec{
			Name:        "browser_click",
			Description: "Click an element in the active Chrome tab by CSS selector. Use after browser_dom_outline — paste a selector from its output. Equivalent to clicking the centre of the element after scrolling it into view. MUCH more reliable than left_click+coords for buttons/links that shift on resize. Requires Chrome's --remote-debugging-port.",
			Schema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"selector"},
				"properties": map[string]any{
					"selector": map[string]any{
						"type":        "string",
						"description": "CSS selector (e.g. `button[type=submit]`, `a.signup-link`, `#search-button`). Same syntax as document.querySelector.",
					},
					"port": map[string]any{
						"type":        "integer",
						"description": "Chrome --remote-debugging-port. Default 9222.",
						"default":     cdpDefaultPort,
					},
				},
			},
			Handler: handleBrowserClick,
		})

		r.register(Spec{
			Name:        "browser_type",
			Description: "Focus an input element by CSS selector and type text into it. Auto-clears any existing value before typing. For search boxes / forms in Chrome where you'd otherwise click-then-type at guessed coords. Requires --remote-debugging-port.",
			Schema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"selector", "text"},
				"properties": map[string]any{
					"selector": map[string]any{
						"type":        "string",
						"description": "CSS selector of the input/textarea to focus.",
					},
					"text": map[string]any{
						"type":        "string",
						"description": "Text to type. Existing value is cleared first.",
					},
					"port": map[string]any{
						"type":        "integer",
						"description": "Chrome --remote-debugging-port. Default 9222.",
						"default":     cdpDefaultPort,
					},
				},
			},
			Handler: handleBrowserType,
		})
	})
}

// cdpContext returns the chromedp context for the given debug port,
// allocating + caching on first use. Returns an error wrapping a
// "did you start Chrome with --remote-debugging-port?" hint when the
// allocator can't reach the port — the most common failure mode.
//
// Target selection: enumerates Chrome's /json target list and picks
// the first `page`-type target whose URL is NOT about:blank /
// chrome://* — that's nearly always the user's actual tab. Without
// this filter chromedp attaches to whatever target is first (often
// about:blank), and Evaluate returns nothing useful (the bug we hit
// 2026-05-22 testing baidu.com).
func cdpContext(port int) (context.Context, error) {
	cdpAllocatorMu.Lock()
	defer cdpAllocatorMu.Unlock()
	if ctx, ok := cdpAllocatorCache[port]; ok {
		return ctx, nil
	}

	wsURL, err := chromedpResolveWS(port)
	if err != nil {
		return nil, fmt.Errorf("connect Chrome on :%d failed: %w (start Chrome with --remote-debugging-port=%d to enable browser_* tools)", port, err, port)
	}

	// 2026-05-22: correct chromedp attach pattern is
	// NewRemoteAllocator(BROWSER_ws) + NewContext(WithTargetID(page_id)).
	// We hit this twice: first using page-level wsURL (chromedp
	// silently created a fresh tab and never bound to the real
	// page), now switched to browser ws + explicit target id.
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), wsURL)

	targetID, err := pickPageTargetID(port)
	if err != nil {
		allocCancel()
		return nil, fmt.Errorf("pick Chrome target on :%d failed: %w", port, err)
	}

	ctx, ctxCancel := chromedp.NewContext(allocCtx, chromedp.WithTargetID(target.ID(targetID)))

	cdpAllocatorCache[port] = ctx
	cdpCancelCache[port] = func() {
		ctxCancel()
		allocCancel()
	}
	return ctx, nil
}

// pickPageTargetID returns the target ID of the first non-blank
// page target. Returns the last page-type target as fallback if
// every page is about:blank. chromedp needs the bare ID (UUID-like
// string from /json), not the ws URL — the ID is passed to
// WithTargetID and chromedp manages the actual ws routing internally.
func pickPageTargetID(port int) (string, error) {
	client := newCDPHTTPClient()
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/json", port))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var targets []struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		URL     string `json:"url"`
		WSDebug string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return "", fmt.Errorf("decode /json: %w", err)
	}
	var fallback string
	for _, t := range targets {
		if t.Type != "page" || t.ID == "" {
			continue
		}
		fallback = t.ID
		if t.URL == "" || t.URL == "about:blank" || hasAnyPrefix(t.URL, "chrome://", "devtools://", "edge://") {
			continue
		}
		return t.ID, nil
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", fmt.Errorf("no page-type Chrome targets — open a real tab first")
}

func hasAnyPrefix(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if len(s) >= len(p) && s[:len(p)] == p {
			return true
		}
	}
	return false
}

// newCDPHTTPClient is a tiny helper — Chrome's debug-port server
// is local, so the timeout is generous (10s covers a busy browser)
// but bounded so a closed port fails fast instead of hanging.
func newCDPHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

// paramIntOr / paramStringOr — tiny, local helpers so this file
// doesn't depend on whatever convention the other tool files use.
func paramIntOr(p map[string]any, key string, def int) int {
	v, ok := p[key]
	if !ok {
		return def
	}
	n, err := asInt(v)
	if err != nil {
		return def
	}
	return n
}

func paramStringOr(p map[string]any, key string, def string) string {
	v, ok := p[key]
	if !ok {
		return def
	}
	s, ok := v.(string)
	if !ok {
		return def
	}
	return s
}

// chromedpResolveWS hits Chrome's /json/version to get the
// webSocketDebuggerUrl. Done with stdlib http to avoid adding a
// dependency just for this one call.
func chromedpResolveWS(port int) (string, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/json/version", port)
	client := newCDPHTTPClient()
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var meta struct {
		WS string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return "", fmt.Errorf("parse Chrome /json/version: %w", err)
	}
	if meta.WS == "" {
		return "", fmt.Errorf("Chrome /json/version had no webSocketDebuggerUrl")
	}
	return meta.WS, nil
}

func handleBrowserDOMOutline(callCtx context.Context, _ platform.Platform, params map[string]any) (*Result, error) {
	port := paramIntOr(params, "port", cdpDefaultPort)
	maxElements := paramIntOr(params, "max_elements", 60)

	ctx, err := cdpContext(port)
	if err != nil {
		return &Result{Text: err.Error(), IsError: true}, nil
	}

	// JS: walk DOM and collect interactive elements with a short
	// "stable enough" selector. Use a per-element data attribute as
	// a fallback when no id is present.
	js := fmt.Sprintf(`
		(function() {
			const limit = %d;
			const out = [];
			const tags = ['a','button','input','textarea','select','[role=button]','[role=link]'];
			const seen = new Set();
			tags.forEach(t => {
				document.querySelectorAll(t).forEach(el => {
					if (out.length >= limit) return;
					if (seen.has(el)) return;
					seen.add(el);
					let sel;
					if (el.id) sel = '#' + el.id;
					else if (el.getAttribute('data-testid')) sel = '[data-testid="' + el.getAttribute('data-testid') + '"]';
					else if (el.name) sel = el.tagName.toLowerCase() + '[name="' + el.name + '"]';
					else sel = el.tagName.toLowerCase() + ':nth-of-type(' + ([...el.parentNode.children].filter(s => s.tagName === el.tagName).indexOf(el) + 1) + ')';
					out.push({
						tag: el.tagName.toLowerCase(),
						type: el.type || '',
						selector: sel,
						text: (el.innerText || el.value || el.placeholder || '').trim().slice(0, 80),
						href: el.href || '',
					});
				});
			});
			return JSON.stringify({total: out.length, elements: out});
		})()
	`, maxElements)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var result string
	if err := chromedp.Run(ctx, chromedp.Evaluate(js, &result)); err != nil {
		return &Result{Text: fmt.Sprintf("browser_dom_outline: %v", err), IsError: true}, nil
	}
	return &Result{Text: result}, nil
}

func handleBrowserClick(callCtx context.Context, _ platform.Platform, params map[string]any) (*Result, error) {
	selector := paramStringOr(params, "selector", "")
	if selector == "" {
		return &Result{Text: "browser_click: `selector` is required (e.g. `button[type=submit]`)", IsError: true}, nil
	}
	port := paramIntOr(params, "port", cdpDefaultPort)
	ctx, err := cdpContext(port)
	if err != nil {
		return &Result{Text: err.Error(), IsError: true}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 2026-05-22: chromedp.Click/WaitVisible kept opening new
	// about:blank tabs even with WithTargetID, so the click never
	// hit the user's actual page. Runtime.Evaluate is the lowest-
	// level CDP call available — bind it through JS that calls
	// .click() directly on the matched element. Same semantics for
	// 99% of click-handlers (no synthetic event dispatch needed).
	js := fmt.Sprintf(`
		(function() {
			const el = document.querySelector(%q);
			if (!el) return "ERR:not_found";
			el.scrollIntoView({block:"center"});
			el.click();
			return "ok";
		})()
	`, selector)
	var result string
	if err := chromedp.Run(ctx, chromedp.Evaluate(js, &result)); err != nil {
		return &Result{Text: fmt.Sprintf("browser_click(%q): %v", selector, err), IsError: true}, nil
	}
	if result == "ERR:not_found" {
		return &Result{Text: fmt.Sprintf("browser_click(%q): element not found — verify selector with browser_dom_outline", selector), IsError: true}, nil
	}
	return &Result{Text: fmt.Sprintf("clicked %s", selector)}, nil
}

func handleBrowserType(callCtx context.Context, _ platform.Platform, params map[string]any) (*Result, error) {
	selector := paramStringOr(params, "selector", "")
	if selector == "" {
		return &Result{Text: "browser_type: `selector` is required", IsError: true}, nil
	}
	text := paramStringOr(params, "text", "")
	port := paramIntOr(params, "port", cdpDefaultPort)
	ctx, err := cdpContext(port)
	if err != nil {
		return &Result{Text: err.Error(), IsError: true}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 2026-05-22: same Runtime.Evaluate workaround as browser_click.
	// Uses the React-compatible native setter pattern + dispatches
	// `input` and `change` events so frameworks (React, Vue) see
	// the change as a real user edit. Most search forms also fire
	// on `input` (autocomplete etc), so this matches what a real
	// keypress would do without simulating individual key codes.
	jsonText, _ := json.Marshal(text)
	js := fmt.Sprintf(`
		(function() {
			const el = document.querySelector(%q);
			if (!el) return "ERR:not_found";
			el.focus();
			const proto = el.tagName === "TEXTAREA" ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
			const setter = Object.getOwnPropertyDescriptor(proto, "value").set;
			setter.call(el, %s);
			el.dispatchEvent(new Event("input", {bubbles: true}));
			el.dispatchEvent(new Event("change", {bubbles: true}));
			return "ok";
		})()
	`, selector, string(jsonText))
	var result string
	if err := chromedp.Run(ctx, chromedp.Evaluate(js, &result)); err != nil {
		return &Result{Text: fmt.Sprintf("browser_type(%q): %v", selector, err), IsError: true}, nil
	}
	if result == "ERR:not_found" {
		return &Result{Text: fmt.Sprintf("browser_type(%q): element not found — verify selector with browser_dom_outline", selector), IsError: true}, nil
	}
	return &Result{Text: fmt.Sprintf("typed %d chars into %s", len(text), selector)}, nil
}
