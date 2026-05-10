package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "list_granted_applications",
			Description: "List the applications metis-cu currently classifies plus the " +
				"frontmost-app context. Returns a JSON object with `frontmost` (the " +
				"currently focused app + its tier) and `apps` (every classified app + " +
				"its tier and source). Apps without explicit user grants fall back to " +
				"the hard-coded default classification (browsers→read, terminals/IDEs→" +
				"click, everything else→full). Use `request_access` to add or override.",
			Schema:  noArgsSchema(),
			Handler: handleListGrantedApplications,
		})
	})
}

// listGrantedReport is the JSON-shaped Result text — DD-6 fix to make
// "is Safari read or full?" parseable without string-scanning the
// previous bullet-list. Preserves the previous columns + adds an
// explicit `source` per app so the model can tell user-granted from
// default-classified entries.
type listGrantedReport struct {
	Frontmost frontmostInfo `json:"frontmost"`
	Apps      []appInfo     `json:"apps"`
}

type frontmostInfo struct {
	Name      string `json:"name,omitempty"`
	Tier      string `json:"tier,omitempty"`
	Available bool   `json:"available"`
	Error     string `json:"error,omitempty"`
}

type appInfo struct {
	Name string `json:"name"`
	Tier string `json:"tier"`
}

func handleListGrantedApplications(_ context.Context, plat platform.Platform, _ map[string]any) (*Result, error) {
	apps, err := plat.GrantedApplications()
	if err != nil {
		return &Result{Text: fmt.Sprintf("list_granted_applications: %v", err), IsError: true}, nil
	}
	report := listGrantedReport{
		Apps: make([]appInfo, 0, len(apps)),
	}
	for _, app := range apps {
		report.Apps = append(report.Apps, appInfo{
			Name: app,
			Tier: string(plat.Tier(app)),
		})
	}
	if name, tier, ferr := plat.FrontmostApp(); ferr != nil {
		report.Frontmost = frontmostInfo{Available: false, Error: ferr.Error()}
	} else {
		report.Frontmost = frontmostInfo{Available: true, Name: name, Tier: string(tier)}
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return &Result{Text: fmt.Sprintf("list_granted_applications: marshal: %v", err), IsError: true}, nil
	}
	return &Result{Text: string(body)}, nil
}
