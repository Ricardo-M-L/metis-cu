package tools

// hint.go — Tier-1 borrow from open-interpreter's `recipient_utils`
// pattern (`format_to_recipient(msg, "assistant")`). Lets tool results
// embed corrective guidance addressed to the model that an MCP client
// MAY hide from the user-visible chat. The convention is a wrapper
// pair the model treats as a side-channel:
//
//	<assistant_hint>
//	You used raw (x,y). Prefer click_text("Send") when a label is
//	visible — accuracy is much higher.
//	</assistant_hint>
//
// MCP clients that don't recognize the tag render it inline (a
// minor cosmetic blemish, not a failure mode). Clients that opt into
// honouring the convention strip the section before display so the
// user only sees actionable text.
//
// The metis-cu side just provides the wrapper helper so individual
// handlers can append hints in a consistent way.

import "strings"

// HintTagOpen / HintTagClose are the wrapper sentinels. Lowercase +
// underscore to stay friendly with terminals that render markup as
// literal text (the worst case is the user sees the tags themselves,
// which is still better than the hint being silently dropped).
const (
	HintTagOpen  = "<assistant_hint>"
	HintTagClose = "</assistant_hint>"
)

// AppendHint returns text with `hint` wrapped in the assistant-only
// tags and joined onto the end. Empty hint or empty text are no-ops
// (return the other verbatim) so handlers can call it unconditionally.
// Two newlines between the action result and the hint so the rendered
// output stays readable in clients that don't strip the tags.
func AppendHint(text, hint string) string {
	hint = strings.TrimSpace(hint)
	if hint == "" {
		return text
	}
	wrapped := HintTagOpen + "\n" + hint + "\n" + HintTagClose
	if text == "" {
		return wrapped
	}
	return text + "\n\n" + wrapped
}

// StripHints removes every <assistant_hint>...</assistant_hint>
// section from text. Useful for clients embedded as a library that
// want to surface a clean user-visible result while still capturing
// the hint content via ExtractHints.
func StripHints(text string) string {
	for {
		o := strings.Index(text, HintTagOpen)
		if o < 0 {
			return text
		}
		c := strings.Index(text[o:], HintTagClose)
		if c < 0 {
			// Unterminated — leave the rest alone rather than swallow
			// the trailing content. This is defensive: well-formed
			// hints always close, but a future tool that emits a stray
			// open tag shouldn't accidentally hide unrelated text.
			return text
		}
		end := o + c + len(HintTagClose)
		text = strings.TrimSpace(text[:o]) + " " + strings.TrimSpace(text[end:])
		text = strings.TrimSpace(text)
	}
}

// ExtractHints returns each assistant-only hint body in document
// order. Empty slice when the text has none. Used by clients that
// log hints for debugging while hiding them from the rendered chat.
func ExtractHints(text string) []string {
	var out []string
	for {
		o := strings.Index(text, HintTagOpen)
		if o < 0 {
			return out
		}
		body := text[o+len(HintTagOpen):]
		c := strings.Index(body, HintTagClose)
		if c < 0 {
			return out
		}
		out = append(out, strings.TrimSpace(body[:c]))
		text = body[c+len(HintTagClose):]
	}
}
