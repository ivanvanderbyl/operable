package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ivanvanderbyl/arnold/pkg/auth"
	"github.com/ivanvanderbyl/arnold/pkg/tools"
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "GCP/K8s Incident Response"
	serverVersion = "0.1.0"
)

func main() {
	// Create a new MCP server
	s := server.NewMCPServer(
		serverName,
		serverVersion,
	)

	// Set up auth handler
	authHandler, err := auth.NewOAuthHandler()
	if err != nil {
		fmt.Printf("Error setting up auth handler: %v\n", err)
		os.Exit(1)
	}

	// Register all tools
	if err := tools.RegisterTools(s, authHandler); err != nil {
		fmt.Printf("Error registering tools: %v\n", err)
		os.Exit(1)
	}

	// Set up context with cancellation for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Log startup message
	fmt.Printf("Starting %s v%s MCP server...\n", serverName, serverVersion)
	fmt.Println("Press Ctrl+C to exit")

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
		os.Exit(1)
	}

	// Wait for context cancellation (this won't be reached in the current implementation
	// as ServeStdio blocks until completion, but included for future enhancements)
	<-ctx.Done()
	fmt.Println("Shutting down server...")
}
