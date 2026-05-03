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
			Description: "Record one or more applications as user-approved at the `full` " +
				"tier. The grant is persisted to $HOME/.metis-cu/granted.json so it " +
				"survives MCP server restarts. Pass an array of display names exactly " +
				"as macOS reports them in `osascript ... frontmost` output.",
			Schema: map[string]any{
				"type":     "object",
				"required": []string{"apps"},
				"properties": map[string]any{
					"apps": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Application display names to grant full access to.",
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
	tiers, err := plat.RequestAccess(apps)
	if err != nil {
		return &Result{Text: fmt.Sprintf("request_access: %v", err), IsError: true}, nil
	}
	// Stable output ordering — easier for traces / eval datasets to diff.
	names := make([]string, 0, len(tiers))
	for app := range tiers {
		names = append(names, app)
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString("Granted access:\n")
	for _, app := range names {
		fmt.Fprintf(&b, "- %s (%s)\n", app, tiers[app])
	}
	return &Result{Text: b.String()}, nil
}
