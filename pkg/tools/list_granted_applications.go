package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "list_granted_applications",
			Description: "List the applications metis-cu currently classifies plus the " +
				"frontmost-app context. Apps without explicit user grants fall back to " +
				"the hard-coded default classification (browsers→read, terminals/IDEs→" +
				"click, everything else→full). Use `request_access` to add or override.",
			Schema: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
			Handler: handleListGrantedApplications,
		})
	})
}

func handleListGrantedApplications(_ context.Context, plat platform.Platform, _ map[string]any) (*Result, error) {
	apps, err := plat.GrantedApplications()
	if err != nil {
		return &Result{Text: fmt.Sprintf("list_granted_applications: %v", err), IsError: true}, nil
	}
	var b strings.Builder
	b.WriteString("Granted applications:\n")
	if len(apps) == 0 {
		b.WriteString("(none)\n")
	}
	for _, app := range apps {
		// TODO(phase-3): expose Tier(app) on platform.Platform so we can
		// annotate each entry with its assigned tier. For now we list
		// names only — the LLM can call `request_access` to inspect or
		// override. Keeping this iteration scoped to file-ownership.
		fmt.Fprintf(&b, "- %s\n", app)
	}
	name, tier, ferr := plat.FrontmostApp()
	if ferr != nil {
		fmt.Fprintf(&b, "\nFrontmost lookup unavailable: %v\n", ferr)
	} else {
		fmt.Fprintf(&b, "\nCurrently frontmost: %s (%s)\n", name, tier)
	}
	return &Result{Text: b.String()}, nil
}
