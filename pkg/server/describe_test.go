package server

import (
	"context"
	"encoding/json"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func readStatus(t *testing.T, c *client.Client, ctx context.Context) statusDocument {
	t.Helper()
	req := mcp.ReadResourceRequest{}
	req.Params.URI = statusResourceURI
	resp, err := c.ReadResource(ctx, req)
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	if len(resp.Contents) != 1 {
		t.Fatalf("status content count = %d, want 1", len(resp.Contents))
	}
	content, ok := resp.Contents[0].(mcp.TextResourceContents)
	if !ok || content.URI != statusResourceURI || content.MIMEType != "application/json" {
		t.Fatalf("status content = %#v", resp.Contents[0])
	}
	var status statusDocument
	decoder := json.NewDecoder(strings.NewReader(content.Text))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&status); err != nil {
		t.Fatalf("decode status: %v; body=%s", err, content.Text)
	}
	return status
}

func TestStatusResourceMatchesProtocolOneDescriptor(t *testing.T) {
	c, ctx := startInProcessClient(t)
	listed, err := c.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	found := false
	for _, resource := range listed.Resources {
		if resource.URI != statusResourceURI {
			continue
		}
		found = true
		if resource.MIMEType != "application/json" ||
			!strings.Contains(resource.Description, "lifecycle.state") ||
			!strings.Contains(resource.Description, "input ownership") {
			t.Fatalf("incomplete status resource metadata: %+v", resource)
		}
	}
	if !found {
		t.Fatalf("status resource not listed: %+v", listed.Resources)
	}

	status := readStatus(t, c, ctx)
	if status.Name != "metis-cu" || status.Version != Version ||
		status.ProtocolVersion != ManagedProtocolVersion ||
		status.Platform != runtime.GOOS || status.Arch != runtime.GOARCH {
		t.Fatalf("status descriptor differs from --describe contract: %+v", status.Descriptor)
	}
	if !reflect.DeepEqual(status.Capabilities, Describe().Capabilities) {
		t.Fatalf("status capabilities = %q, want %q", status.Capabilities, Describe().Capabilities)
	}
	for _, key := range []string{"accessibility", "screenRecording"} {
		if status.Permissions[key] == "" {
			t.Errorf("missing permission status: %s", key)
		}
	}
	if status.Lifecycle.State != "idle" {
		t.Fatalf("initial lifecycle.state = %q, want idle", status.Lifecycle.State)
	}
}

func TestStatusResourceReportsActiveToolHandler(t *testing.T) {
	c, ctx := startInProcessClient(t)
	done := make(chan error, 1)
	go func() {
		req := mcp.CallToolRequest{}
		req.Params.Name = "wait"
		req.Params.Arguments = map[string]any{"ms": 1200}
		_, err := c.CallTool(ctx, req)
		done <- err
	}()

	deadline := time.Now().Add(900 * time.Millisecond)
	for {
		if status := readStatus(t, c, ctx); status.Lifecycle.State == "running" {
			break
		}
		select {
		case err := <-done:
			t.Fatalf("wait tool finished before running status was observed: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("lifecycle.state did not become running during wait tool")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := <-done; err != nil {
		t.Fatalf("wait tool: %v", err)
	}
	if state := readStatus(t, c, ctx).Lifecycle.State; state != "idle" {
		t.Fatalf("lifecycle.state after wait = %q, want idle", state)
	}
}
