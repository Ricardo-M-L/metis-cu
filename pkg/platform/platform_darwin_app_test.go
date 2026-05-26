//go:build darwin

package platform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestFrontmostApp_NonEmpty verifies the osascript invocation works
// against a real macOS GUI session. Skips when osascript can't reach
// System Events (typical on CI runners with no logged-in window
// server) — we only assert basic shape, not a specific app name, since
// the active app at test time depends on what the user is doing.
func TestFrontmostApp_NonEmpty(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping FrontmostApp test in CI (no GUI session)")
	}
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	name, tier, err := p.FrontmostApp()
	if err != nil {
		t.Skipf("frontmost lookup unavailable: %v", err)
	}
	if name == "" {
		t.Fatal("FrontmostApp returned empty name with no error")
	}
	switch tier {
	case TierRead, TierClick, TierFull:
		// ok
	default:
		t.Fatalf("FrontmostApp returned unknown tier %q for app %q", tier, name)
	}
}

// TestGrantedApplications_ContainsDefaults runs without requiring a
// macOS GUI — GrantedApplications just unions the in-process maps.
// Asserts the hard-coded default list survives the call (Safari is
// stable for the duration of the project).
func TestGrantedApplications_ContainsDefaults(t *testing.T) {
	// Use a clean HOME so we don't get cross-test contamination.
	t.Setenv("HOME", t.TempDir())
	resetGrantedForTest()
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	apps, err := p.GrantedApplications()
	if err != nil {
		t.Fatalf("GrantedApplications: %v", err)
	}
	if !contains(apps, "Safari") {
		t.Fatalf("GrantedApplications missing default %q; got %v", "Safari", apps)
	}
	if !contains(apps, "Terminal") {
		t.Fatalf("GrantedApplications missing default %q; got %v", "Terminal", apps)
	}
	// Sorted alphabetically.
	for i := 1; i < len(apps); i++ {
		if apps[i-1] > apps[i] {
			t.Fatalf("GrantedApplications not sorted: %v", apps)
		}
	}
}

// TestTier_HonoursHostTerminalOverride — 2026-05-26 regression for
// session 41040bea: with the host-terminal override unset, iTerm2
// resolves to TierClick (historical default) which means an MCP call
// from inside iTerm2 gets `open_application` rejected. With the
// override set, iTerm2 resolves to the configured tier so metis can
// drive the app it just launched.
//
// Pins the lookup order: user grants > host-terminal override >
// hard-coded defaults > TierFull. The middle layer is what's new in
// this change and the regression risk lives in there — granted-app
// callers must still win even when the override is configured.
func TestTier_HonoursHostTerminalOverride(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resetGrantedForTest()
	SetHostTerminalOverride("")
	t.Cleanup(func() { SetHostTerminalOverride("") })

	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Baseline: iTerm2 is TierClick per defaultTiers; that's what
	// makes the metis case break.
	if got := p.Tier("iTerm2"); got != TierClick {
		t.Fatalf("baseline Tier(iTerm2) = %q, want click (defaultTiers); did the table change?", got)
	}

	// Promote the host terminals — iTerm2 should now report full.
	SetHostTerminalOverride(TierFull)
	if got := p.Tier("iTerm2"); got != TierFull {
		t.Fatalf("after override=full, Tier(iTerm2) = %q, want full", got)
	}
	// Non-terminal default is untouched (the override only widens the
	// allow-listed host terminals).
	if got := p.Tier("Safari"); got != TierRead {
		t.Errorf("override leaked to non-terminal app: Tier(Safari) = %q, want read", got)
	}

	// User grant must still win over the host-terminal override —
	// otherwise a deliberately-restricted "iTerm2 = read" grant
	// would be silently widened back to full by the override.
	if _, err := p.RequestAccess([]string{"iTerm2"}, TierRead); err != nil {
		t.Fatalf("RequestAccess: %v", err)
	}
	if got := p.Tier("iTerm2"); got != TierRead {
		t.Errorf("user grant should win over override; got %q want read", got)
	}
}

// TestRequestAccess_AddsAndPersists verifies the in-memory record plus
// the JSON persistence side-effect. Uses t.Setenv to point HOME at a
// temp dir so we don't pollute the real user's $HOME/.metis-cu.
func TestRequestAccess_AddsAndPersists(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	resetGrantedForTest()
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got, err := p.RequestAccess([]string{"TestApp1", "TestApp2"}, TierFull)
	if err != nil {
		t.Fatalf("RequestAccess: %v", err)
	}
	if got["TestApp1"] != TierFull || got["TestApp2"] != TierFull {
		t.Fatalf("RequestAccess returned unexpected tiers: %v", got)
	}
	apps, err := p.GrantedApplications()
	if err != nil {
		t.Fatalf("GrantedApplications: %v", err)
	}
	if !contains(apps, "TestApp1") {
		t.Fatalf("GrantedApplications missing TestApp1 after RequestAccess; got %v", apps)
	}
	if !contains(apps, "TestApp2") {
		t.Fatalf("GrantedApplications missing TestApp2 after RequestAccess; got %v", apps)
	}
	// Verify persistence file shape.
	path := filepath.Join(tempHome, ".metis-cu", "granted.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected granted.json at %s: %v", path, err)
	}
	var wire map[string]string
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatalf("granted.json malformed: %v\n%s", err, string(data))
	}
	if wire["TestApp1"] != string(TierFull) {
		t.Fatalf("granted.json missing TestApp1=full; got %v", wire)
	}
	if wire["TestApp2"] != string(TierFull) {
		t.Fatalf("granted.json missing TestApp2=full; got %v", wire)
	}
}

// TestConfirmScript_Escapes asserts the AppleScript builder escapes
// backslashes and embedded quotes. Driven through this helper rather
// than calling Confirm directly so we don't pop a real osascript dialog
// during go test (which would either block forever or fail in CI).
func TestConfirmScript_Escapes(t *testing.T) {
	cases := []struct {
		name, in string
		mustHave []string
	}{
		{
			name: "plain",
			in:   "Allow this?",
			mustHave: []string{
				`display dialog "Allow this?"`,
				`buttons {"Deny", "Allow"}`,
				`default button "Deny"`,
				`giving up after 60`,
			},
		},
		{
			name:     "escapes-double-quote",
			in:       `Allow access to "Notes"?`,
			mustHave: []string{`Allow access to \"Notes\"?`},
		},
		{
			name:     "escapes-backslash",
			in:       `path C:\Users`,
			mustHave: []string{`path C:\\Users`},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := confirmScript(tc.in)
			for _, want := range tc.mustHave {
				if !contains([]string{got}, want) && !containsSubstring(got, want) {
					t.Errorf("confirmScript(%q) missing %q\nfull script: %s", tc.in, want, got)
				}
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
