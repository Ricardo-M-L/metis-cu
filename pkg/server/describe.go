package server

import (
	"context"
	"encoding/json"
	"runtime"
	"sync/atomic"

	"github.com/Ricardo-M-L/metis-cu/pkg/platform"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const ManagedProtocolVersion = 1
const statusResourceURI = "metis-cu://status"

// Descriptor is the protocol-1 contract shared by --describe and the MCP
// status resource. Describe does not initialize a desktop backend or request
// permissions.
type Descriptor struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	ProtocolVersion int               `json:"protocolVersion"`
	Platform        string            `json:"platform"`
	Arch            string            `json:"arch"`
	Capabilities    []string          `json:"capabilities"`
	Permissions     map[string]string `json:"permissions"`
}

func Describe() Descriptor {
	return Descriptor{
		Name:            "metis-cu",
		Version:         Version,
		ProtocolVersion: ManagedProtocolVersion,
		Platform:        runtime.GOOS,
		Arch:            runtime.GOARCH,
		Capabilities: []string{
			"status",
			"stop",
			"end-turn",
			"serialized-input",
			"input-ownership",
		},
		Permissions: platform.PermissionStatus(),
	}
}

// toolActivity counts registered MCP tool handlers that have started and
// have not returned. Idle does not imply OS input ownership is free, nor does
// running describe a host turn: this process cannot observe either reliably.
type toolActivity struct {
	active atomic.Int64
}

func (a *toolActivity) state() string {
	if a.active.Load() > 0 {
		return "running"
	}
	return "idle"
}

type statusDocument struct {
	Descriptor
	Lifecycle struct {
		State string `json:"state"`
	} `json:"lifecycle"`
}

func registerStatusResource(srv *mcpserver.MCPServer, activity *toolActivity) {
	resource := mcp.NewResource(statusResourceURI, "Computer use status",
		mcp.WithMIMEType("application/json"),
		mcp.WithResourceDescription("Current helper descriptor and MCP tool activity. lifecycle.state is running only while a registered tool handler executes; idle means none is executing. It does not report host turn state or OS input ownership."),
	)
	srv.AddResource(resource, func(context.Context, mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		status := statusDocument{Descriptor: Describe()}
		status.Lifecycle.State = activity.state()
		data, err := json.Marshal(status)
		if err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{mcp.TextResourceContents{
			URI: statusResourceURI, MIMEType: "application/json", Text: string(data),
		}}, nil
	})
}
