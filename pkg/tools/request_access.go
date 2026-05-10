package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "request_access",
			Description: "Ask the human user to grant one or more applications access at a " +
				"given tier. Each not-yet-granted app triggers a native OS confirmation " +
				"dialog the user must click through; previously granted apps are skipped " +
				"silently. Approved grants persist to $HOME/.metis-cu/granted.json so the " +
				"prompt is one-time per app. tier defaults to \"full\"; choose \"click\" " +
				"to allow only pointer events (good for terminals/IDEs) or \"read\" to " +
				"allow only screenshot / clipboard read.",
			Schema: map[string]any{
				"type":     "object",
				"required": []string{"apps"},
				"properties": map[string]any{
					"apps": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Application display names to request access for. Names must match the frontmost-app probe (macOS: `osascript ... frontmost`; Linux: `xdotool getwindowclassname`; Windows: lowercased exe basename).",
					},
					"tier": map[string]any{
						"type":        "string",
						"enum":        []string{"read", "click", "full"},
						"description": "Access tier to grant: read (vision only), click (pointer events), full (typing + everything). Default: full.",
					},
				},
				"additionalProperties": false,
			},
			Handler: handleRequestAccess,
		})
	})
}

func handleRequestAccess(_ context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	raw, ok := params["apps"]
	if !ok {
		return &Result{Text: "missing required field: apps", IsError: true}, nil
	}
	apps, err := asStringSlice(raw)
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid apps: %v", err), IsError: true}, nil
	}
	if len(apps) == 0 {
		return &Result{Text: "request_access: empty apps list (no-op)"}, nil
	}

	// BUG-20: optional per-call tier — defaults to TierFull to preserve
	// the historical contract. Validate against the schema enum so an
	// odd value gets a clear error rather than silently downgrading to
	// Full via normaliseTier in the platform layer.
	tier := platform.TierFull
	if rawTier, present := params["tier"]; present {
		ts, ok := rawTier.(string)
		if !ok {
			return &Result{Text: fmt.Sprintf("tier: expected string, got %T", rawTier), IsError: true}, nil
		}
		switch platform.AccessTier(ts) {
		case platform.TierRead, platform.TierClick, platform.TierFull:
			tier = platform.AccessTier(ts)
		case "":
			// empty string → default
		default:
			return &Result{Text: fmt.Sprintf("tier: %q not in {read, click, full}", ts), IsError: true}, nil
		}
	}

	// Already-granted apps skip the prompt — granted.json is durable
	// per the platform layer, so a second request_access call after a
	// restart still finds prior approvals without re-prompting.
	existing, _ := plat.GrantedApplications()
	alreadyGranted := make(map[string]struct{}, len(existing))
	for _, a := range existing {
		alreadyGranted[a] = struct{}{}
	}

	var (
		approved []string
		skipped  []string
		denied   []string
	)
	for _, app := range apps {
		app = strings.TrimSpace(app)
		if app == "" {
			continue
		}
		if _, ok := alreadyGranted[app]; ok {
			skipped = append(skipped, app)
			continue
		}
		msg := fmt.Sprintf(
			"Allow metis-cu (computer-use MCP server) to control %q at %q access?\n\n"+
				"This grant is persisted to ~/.metis-cu/granted.json — you won't be asked again.",
			app, tier,
		)
		ok, err := plat.Confirm(msg)
		if err != nil {
			return &Result{
				Text:    fmt.Sprintf("request_access(%q): confirm prompt failed: %v", app, err),
				IsError: true,
			}, nil
		}
		if ok {
			approved = append(approved, app)
		} else {
			denied = append(denied, app)
		}
	}

	// Persist all approved apps in one shot — the platform layer takes
	// care of fsync/rename, no need to call once per app.
	if len(approved) > 0 {
		if _, err := plat.RequestAccess(approved, tier); err != nil {
			return &Result{
				Text:    fmt.Sprintf("persist granted: %v", err),
				IsError: true,
			}, nil
		}
	}

	sort.Strings(approved)
	sort.Strings(skipped)
	sort.Strings(denied)

	var b strings.Builder
	if len(approved) > 0 {
		fmt.Fprintf(&b, "Granted (%s): %s\n", tier, strings.Join(approved, ", "))
	}
	if len(skipped) > 0 {
		fmt.Fprintf(&b, "Already granted (skipped): %s\n", strings.Join(skipped, ", "))
	}
	if len(denied) > 0 {
		fmt.Fprintf(&b, "Denied by user: %s\n", strings.Join(denied, ", "))
	}
	if b.Len() == 0 {
		// All entries were empty strings after trim — surface the no-op.
		return &Result{Text: "request_access: no app names provided after trimming"}, nil
	}
	return &Result{Text: strings.TrimRight(b.String(), "\n")}, nil
}
