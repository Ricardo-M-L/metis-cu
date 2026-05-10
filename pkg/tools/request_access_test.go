package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

// fakeRequestAccessPlat is a Platform fake that records confirm
// invocations and proxies grant persistence. The Confirm hook returns
// the next bool from confirmAnswers (queued FIFO) so a single test
// can simulate a mix of allow/deny clicks across multiple apps.
type fakeRequestAccessPlat struct {
	stubPlat

	// pre-set state
	alreadyGranted []string
	confirmAnswers []bool // FIFO: one bool per Confirm call
	confirmErr     error  // returned by every Confirm call when non-nil
	reqErr         error  // returned by RequestAccess

	// recorded calls
	confirmCalls []string
	requestCalls [][]string
}

func (p *fakeRequestAccessPlat) GrantedApplications() ([]string, error) {
	out := append([]string(nil), p.alreadyGranted...)
	return out, nil
}

func (p *fakeRequestAccessPlat) Confirm(message string) (bool, error) {
	p.confirmCalls = append(p.confirmCalls, message)
	if p.confirmErr != nil {
		return false, p.confirmErr
	}
	if len(p.confirmAnswers) == 0 {
		return false, nil
	}
	ans := p.confirmAnswers[0]
	p.confirmAnswers = p.confirmAnswers[1:]
	return ans, nil
}

func (p *fakeRequestAccessPlat) RequestAccess(apps []string, tier platform.AccessTier) (map[string]platform.AccessTier, error) {
	p.requestCalls = append(p.requestCalls, append([]string(nil), apps...))
	if p.reqErr != nil {
		return nil, p.reqErr
	}
	out := make(map[string]platform.AccessTier, len(apps))
	for _, a := range apps {
		if tier == "" {
			tier = platform.TierFull
		}
		out[a] = tier
	}
	return out, nil
}

// TestRequestAccess_AllAllowed: user clicks Allow for every app →
// every app ends up in the persist list and Result reports Granted.
func TestRequestAccess_AllAllowed(t *testing.T) {
	p := &fakeRequestAccessPlat{confirmAnswers: []bool{true, true}}
	params := map[string]any{"apps": []any{"Safari", "Notes"}}
	res, err := handleRequestAccess(context.Background(), p, params)
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if len(p.confirmCalls) != 2 {
		t.Errorf("expected 2 Confirm calls, got %d", len(p.confirmCalls))
	}
	if len(p.requestCalls) != 1 || len(p.requestCalls[0]) != 2 {
		t.Errorf("expected 1 batched RequestAccess of 2 apps, got %v", p.requestCalls)
	}
	if !strings.Contains(res.Text, "Granted (full): Notes, Safari") {
		t.Errorf("expected sorted Granted list with default tier, got: %s", res.Text)
	}
}

// TestRequestAccess_TierClick: explicit tier=click → Result echoes
// the tier and the platform RequestAccess call carries it through.
func TestRequestAccess_TierClick(t *testing.T) {
	p := &fakeRequestAccessPlat{confirmAnswers: []bool{true}}
	params := map[string]any{"apps": []any{"iTerm2"}, "tier": "click"}
	res, err := handleRequestAccess(context.Background(), p, params)
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if !strings.Contains(res.Text, "Granted (click): iTerm2") {
		t.Errorf("expected tier=click in Result, got: %s", res.Text)
	}
}

// TestRequestAccess_TierInvalid: bogus tier rejected at the tool
// layer with a helpful enum-listing error.
func TestRequestAccess_TierInvalid(t *testing.T) {
	p := &fakeRequestAccessPlat{}
	params := map[string]any{"apps": []any{"Notes"}, "tier": "supreme"}
	res, _ := handleRequestAccess(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError for invalid tier")
	}
	if !strings.Contains(res.Text, "{read, click, full}") {
		t.Errorf("expected enum hint in error, got: %s", res.Text)
	}
}

// TestRequestAccess_AllDenied: user clicks Deny → no persistence call,
// Result reports Denied.
func TestRequestAccess_AllDenied(t *testing.T) {
	p := &fakeRequestAccessPlat{confirmAnswers: []bool{false, false}}
	params := map[string]any{"apps": []any{"Safari", "Notes"}}
	res, err := handleRequestAccess(context.Background(), p, params)
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if len(p.requestCalls) != 0 {
		t.Errorf("expected zero RequestAccess calls when all denied, got %v", p.requestCalls)
	}
	if !strings.Contains(res.Text, "Denied by user: Notes, Safari") {
		t.Errorf("expected sorted Denied list, got: %s", res.Text)
	}
}

// TestRequestAccess_AlreadyGranted_NoPrompt: apps already in
// granted.json skip the Confirm dialog entirely.
func TestRequestAccess_AlreadyGranted_NoPrompt(t *testing.T) {
	p := &fakeRequestAccessPlat{
		alreadyGranted: []string{"Safari"},
		confirmAnswers: []bool{true},
	}
	params := map[string]any{"apps": []any{"Safari", "Notes"}}
	res, err := handleRequestAccess(context.Background(), p, params)
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if len(p.confirmCalls) != 1 {
		t.Fatalf("expected only 1 Confirm call (Notes); got %d (%v)", len(p.confirmCalls), p.confirmCalls)
	}
	if !strings.Contains(p.confirmCalls[0], "Notes") {
		t.Errorf("expected Notes in Confirm message, got %q", p.confirmCalls[0])
	}
	if !strings.Contains(res.Text, "Granted (full): Notes") {
		t.Errorf("expected Notes granted at full tier, got: %s", res.Text)
	}
	if !strings.Contains(res.Text, "Already granted (skipped): Safari") {
		t.Errorf("expected Safari skipped, got: %s", res.Text)
	}
}

// TestRequestAccess_MixedAllowDeny: one Allow + one Deny — both lists
// in Result, only allowed app persisted.
func TestRequestAccess_MixedAllowDeny(t *testing.T) {
	p := &fakeRequestAccessPlat{confirmAnswers: []bool{true, false}}
	params := map[string]any{"apps": []any{"Safari", "Notes"}}
	res, err := handleRequestAccess(context.Background(), p, params)
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
	if len(p.requestCalls) != 1 || len(p.requestCalls[0]) != 1 || p.requestCalls[0][0] != "Safari" {
		t.Errorf("expected only Safari persisted, got %v", p.requestCalls)
	}
	if !strings.Contains(res.Text, "Granted (full): Safari") {
		t.Errorf("expected Granted: Safari in result, got: %s", res.Text)
	}
	if !strings.Contains(res.Text, "Denied by user: Notes") {
		t.Errorf("expected Denied Notes in result, got: %s", res.Text)
	}
}

// TestRequestAccess_ConfirmError: Confirm returning error (e.g. on
// linux where the dialog backend isn't implemented) surfaces as an
// IsError result so the LLM doesn't think the grant succeeded.
func TestRequestAccess_ConfirmError(t *testing.T) {
	p := &fakeRequestAccessPlat{confirmErr: platform.ErrNotImplemented}
	params := map[string]any{"apps": []any{"Safari"}}
	res, _ := handleRequestAccess(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when Confirm fails")
	}
	if !strings.Contains(res.Text, "confirm prompt failed") {
		t.Errorf("expected error text, got: %s", res.Text)
	}
	if len(p.requestCalls) != 0 {
		t.Errorf("expected no RequestAccess on confirm failure, got %v", p.requestCalls)
	}
}

func TestRequestAccess_EmptyApps(t *testing.T) {
	p := &fakeRequestAccessPlat{}
	res, _ := handleRequestAccess(context.Background(), p, map[string]any{"apps": []any{}})
	if res.IsError {
		t.Fatalf("empty apps should be a no-op, not error: %s", res.Text)
	}
	if len(p.confirmCalls) != 0 {
		t.Errorf("expected 0 Confirm calls for empty apps, got %d", len(p.confirmCalls))
	}
}

func TestRequestAccess_MissingApps(t *testing.T) {
	p := &fakeRequestAccessPlat{}
	res, _ := handleRequestAccess(context.Background(), p, map[string]any{})
	if !res.IsError {
		t.Fatal("expected IsError when apps is missing")
	}
}

func TestRequestAccess_InvalidAppsType(t *testing.T) {
	p := &fakeRequestAccessPlat{}
	res, _ := handleRequestAccess(context.Background(), p, map[string]any{"apps": "not-an-array"})
	if !res.IsError {
		t.Fatal("expected IsError when apps is not an array")
	}
}

// TestRequestAccess_PersistError: Confirm succeeds but the underlying
// RequestAccess (granted.json write) fails — surface the error so the
// LLM doesn't think the grant landed.
func TestRequestAccess_PersistError(t *testing.T) {
	p := &fakeRequestAccessPlat{
		confirmAnswers: []bool{true},
		reqErr:         errors.New("disk full"),
	}
	params := map[string]any{"apps": []any{"Safari"}}
	res, _ := handleRequestAccess(context.Background(), p, params)
	if !res.IsError {
		t.Fatal("expected IsError when persistence fails")
	}
	if !strings.Contains(res.Text, "disk full") {
		t.Errorf("expected disk full in result, got: %s", res.Text)
	}
}
