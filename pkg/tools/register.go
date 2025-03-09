package tools

import (
	"context"
	"fmt"

	"github.com/ivanvanderbyl/operable/pkg/auth"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTools registers all tools with the MCP server
func RegisterTools(s *server.MCPServer, authHandler *auth.OAuthHandler) error {
	// Register GCP issues tool
	if err := registerGCPIssuesTools(s, authHandler); err != nil {
		return fmt.Errorf("error registering GCP issues tools: %w", err)
	}

	// Register logging tools
	if err := registerLoggingTools(s, authHandler); err != nil {
		return fmt.Errorf("error registering logging tools: %w", err)
	}

	// Register Kubernetes tools
	if err := registerKubernetesTools(s, authHandler); err != nil {
		return fmt.Errorf("error registering Kubernetes tools: %w", err)
	}

	// Register monitoring tools
	if err := registerMonitoringTools(s, authHandler); err != nil {
		return fmt.Errorf("error registering monitoring tools: %w", err)
	}

	// Register documentation tools
	if err := registerDocumentationTools(s); err != nil {
		return fmt.Errorf("error registering documentation tools: %w", err)
	}

	return nil
}

// AddToolSafe is a wrapper around AddTool that ignores the linting issue
// This is a workaround for the linting issue with s.AddTool
func AddToolSafe(s *server.MCPServer, tool mcp.Tool, handler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	s.AddTool(tool, handler)
}
