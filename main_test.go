package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Ricardo-M-L/metis-cu/pkg/server"
)

func TestPermissionRequestProducesDescriptorWithoutStartingServer(t *testing.T) {
	for _, tc := range []struct {
		args []string
		kind string
	}{
		{[]string{"--request-permission", "accessibility", "--json"}, "accessibility"},
		{[]string{"--json", "--request-permission", "screen-recording"}, "screen-recording"},
		{[]string{"--request-permission=accessibility", "--json"}, "accessibility"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			requests := 0
			code := runCLI(tc.args, &stdout, &stderr,
				func(server.Options) error { t.Fatal("started MCP server"); return nil },
				func(kind string) error {
					requests++
					if kind != tc.kind {
						t.Errorf("requested %q, want %q", kind, tc.kind)
					}
					return nil
				})
			if code != 0 || requests != 1 || stderr.Len() != 0 {
				t.Fatalf("exit=%d requests=%d stderr=%q", code, requests, &stderr)
			}
			var got description
			if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
				t.Fatalf("invalid descriptor: %v, output=%q", err, &stdout)
			}
			if got.Name != "metis-cu" || got.ProtocolVersion != managedProtocolVersion {
				t.Fatalf("wrong protocol descriptor: %+v", got)
			}
			for _, name := range []string{"screenRecording", "accessibility"} {
				if got.Permissions[name] == "" || got.Permissions[name] == "runtime" {
					t.Errorf("%s status is not actionable: %q", name, got.Permissions[name])
				}
			}
		})
	}
}

func TestDescribeDoesNotRequestPermissionOrStartServer(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"--describe", "--json"}, &stdout, &stderr,
		func(server.Options) error { t.Fatal("started MCP server"); return nil },
		func(string) error { t.Fatal("requested OS permission"); return nil })
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("exit=%d stderr=%q", code, &stderr)
	}
	var got description
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("invalid descriptor: %v", err)
	}
	if got.Permissions["accessibility"] == "runtime" || got.Permissions["screenRecording"] == "runtime" {
		t.Fatalf("descriptor still reports runtime: %+v", got.Permissions)
	}
}

func TestInvalidPermissionRequestNeverPrompts(t *testing.T) {
	for _, args := range [][]string{
		{"--request-permission", ""},
		{"--request-permission", "camera"},
		{"--request-permission", "accessibility", "extra"},
		{"--request-permission", "screen-recording", "--describe"},
		{"--debug", "--request-permission", "accessibility"},
	} {
		var stdout, stderr bytes.Buffer
		code := runCLI(args, &stdout, &stderr,
			func(server.Options) error { t.Fatal("started MCP server"); return nil },
			func(string) error { t.Fatal("requested OS permission"); return nil })
		if code != 2 || stdout.Len() != 0 || stderr.Len() == 0 {
			t.Errorf("args=%q exit=%d stdout=%q stderr=%q", args, code, &stdout, &stderr)
		}
	}
}

func TestPermissionRequestPropagatesError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"--request-permission", "accessibility", "--json"}, &stdout, &stderr,
		func(server.Options) error { t.Fatal("started MCP server"); return nil },
		func(string) error { return errors.New("unavailable") })
	if code != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "unavailable") {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, &stdout, &stderr)
	}
}

func TestPermissionRequestWithoutJSONPrintsCurrentStatus(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"--request-permission", "accessibility"}, &stdout, &stderr,
		func(server.Options) error { t.Fatal("started MCP server"); return nil },
		func(string) error { return nil })
	if code != 0 || stderr.Len() != 0 || !strings.HasPrefix(stdout.String(), "accessibility: ") {
		t.Fatalf("exit=%d stdout=%q stderr=%q", code, &stdout, &stderr)
	}
}
