package tools

import (
	"context"
	"fmt"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
)

func init() {
	addRegistration(func(r *Registry) {
		r.register(Spec{
			Name: "open_application",
			Description: "Launch a macOS application by display name (e.g. \"Safari\", " +
				"\"Visual Studio Code\"). Equivalent to `open -a <name>`. Brings the app " +
				"to the foreground if it is already running.",
			Schema: map[string]any{
				"type":     "object",
				"required": []string{"name"},
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "App name (e.g. \"Safari\")",
					},
				},
				"additionalProperties": false,
			},
			Handler: handleOpenApplication,
		})
	})
}

func handleOpenApplication(_ context.Context, plat platform.Platform, params map[string]any) (*Result, error) {
	name, err := requireString(params, "name")
	if err != nil {
		return &Result{Text: fmt.Sprintf("invalid params: %v", err), IsError: true}, nil
	}
	if err := plat.OpenApplication(name); err != nil {
		return &Result{Text: fmt.Sprintf("open_application(%q): %v", name, err), IsError: true}, nil
	}
	return &Result{Text: fmt.Sprintf("opened %s", name)}, nil
}
