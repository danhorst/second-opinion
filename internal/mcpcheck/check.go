// Package mcpcheck performs non-review MCP discovery. It never calls a tool.
package mcpcheck

import (
	"context"
	"errors"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Result struct {
	ServerName    string   `json:"server_name"`
	ServerVersion string   `json:"server_version"`
	Tools         []string `json:"tools"`
}

// Discover initializes a real MCP session and checks the first tools page.
// Second Opinion exposes exactly two tools without pagination.
func Discover(ctx context.Context, transport mcp.Transport) (Result, error) {
	var result Result
	if err := ctx.Err(); err != nil {
		return result, err
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "second-opinion-doctor", Version: "1"}, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return result, errors.New("MCP initialization failed; check executable, service, and token settings")
	}
	defer session.Close()
	info := session.InitializeResult().ServerInfo
	if info == nil || info.Name != "second-opinion" {
		return result, errors.New("endpoint is not a Second Opinion MCP server")
	}
	result.ServerName, result.ServerVersion = info.Name, info.Version
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		return result, errors.New("MCP tool discovery failed")
	}
	found := map[string]bool{}
	for _, tool := range tools.Tools {
		if tool != nil {
			found[tool.Name] = true
			result.Tools = append(result.Tools, tool.Name)
		}
	}
	sort.Strings(result.Tools)
	if !found["review"] || !found["list_providers"] {
		return result, errors.New("MCP server is missing required tools: review and list_providers")
	}
	return result, nil
}
