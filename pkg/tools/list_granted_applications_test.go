package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

type fakeGrantedAppsPlat struct {
	stubPlat
	appsErr   error
	apps      []string
	frontErr  error
	frontApp  string
	frontTier platform.AccessTier
	tiers     map[string]platform.AccessTier
}

func (p *fakeGrantedAppsPlat) GrantedApplications() ([]string, error) {
	return p.apps, p.appsErr
}

func (p *fakeGrantedAppsPlat) FrontmostApp() (string, platform.AccessTier, error) {
	return p.frontApp, p.frontTier, p.frontErr
}

func (p *fakeGrantedAppsPlat) Tier(name string) platform.AccessTier {
	if t, ok := p.tiers[name]; ok {
		return t
	}
	return platform.TierFull
}

func TestListGrantedApplications_OK(t *testing.T) {
	p := &fakeGrantedAppsPlat{
		apps:      []string{"Safari", "Terminal", "VSCode"},
		frontApp:  "Safari",
		frontTier: platform.TierRead,
	}
	res, err := handleListGrantedApplications(context.Background(), p, nil)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %s", res.Text)
	}
}

func TestListGrantedApplications_Empty(t *testing.T) {
	p := &fakeGrantedAppsPlat{
		apps:      []string{},
		frontApp:  "Finder",
		frontTier: platform.TierFull,
	}
	res, err := handleListGrantedApplications(context.Background(), p, nil)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if res.IsError {
		t.Fatalf("empty apps list should not be an error: %s", res.Text)
	}
}

func TestListGrantedApplications_GrantedAppsError(t *testing.T) {
	p := &fakeGrantedAppsPlat{appsErr: errors.New("failed to get granted apps")}
	res, _ := handleListGrantedApplications(context.Background(), p, nil)
	if !res.IsError {
		t.Fatal("expected IsError when GrantedApplications fails")
	}
}

func TestListGrantedApplications_FrontmostError(t *testing.T) {
	p := &fakeGrantedAppsPlat{
		apps:     []string{"Safari"},
		frontErr: errors.New("no frontmost app"),
	}
	res, err := handleListGrantedApplications(context.Background(), p, nil)
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	// frontmost lookup failure is surfaced in JSON but doesn't make IsError
	if res.IsError {
		t.Fatalf("frontmost error should not make IsError: %s", res.Text)
	}
}

// TestListGrantedApplications_StructuredJSON (DD-6): Result text is
// machine-parseable JSON with frontmost + per-app tier + source.
// Replaces the previous bullet list which forced the model to do
// fuzzy string matching.
func TestListGrantedApplications_StructuredJSON(t *testing.T) {
	p := &fakeGrantedAppsPlat{
		apps:      []string{"Safari", "Terminal"},
		frontApp:  "Safari",
		frontTier: platform.TierRead,
		tiers: map[string]platform.AccessTier{
			"Safari":   platform.TierRead,
			"Terminal": platform.TierClick,
		},
	}
	res, err := handleListGrantedApplications(context.Background(), p, nil)
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	var report listGrantedReport
	if err := json.Unmarshal([]byte(res.Text), &report); err != nil {
		t.Fatalf("Result.Text not valid JSON: %v\n--- raw:\n%s", err, res.Text)
	}
	if !report.Frontmost.Available || report.Frontmost.Name != "Safari" || report.Frontmost.Tier != "read" {
		t.Errorf("frontmost: %+v", report.Frontmost)
	}
	if len(report.Apps) != 2 {
		t.Fatalf("apps len: got %d, want 2; raw: %s", len(report.Apps), res.Text)
	}
	for _, a := range report.Apps {
		switch a.Name {
		case "Safari":
			if a.Tier != "read" {
				t.Errorf("Safari tier: got %s, want read", a.Tier)
			}
		case "Terminal":
			if a.Tier != "click" {
				t.Errorf("Terminal tier: got %s, want click", a.Tier)
			}
		default:
			t.Errorf("unexpected app: %+v", a)
		}
	}
}
