package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ivanvanderbyl/operable/pkg/auth"
	"github.com/ivanvanderbyl/operable/pkg/tools"
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "GCP/K8s Incident Response"
	serverVersion = "0.1.0"
)

func main() {
	// Parse command-line flags
	mode := flag.String("mode", "stdio", "Server mode: 'stdio' or 'sse'")
	addr := flag.String("addr", ":8080", "Address to listen on in SSE mode")
	baseURL := flag.String("base-url", "http://localhost:8080", "Base URL for SSE mode")
	flag.Parse()

	// Create a new MCP server
	s := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithLogging(),
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

	// Start the server in the specified mode
	fmt.Printf("Starting %s v%s MCP server in %s mode...\n", serverName, serverVersion, *mode)

	switch *mode {
	case "stdio":
		// Start the stdio server
		if err := server.ServeStdio(s); err != nil {
			fmt.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	case "sse":
		// Create and start the SSE server
		sseServer := server.NewSSEServer(s, *baseURL)

		// Start the server in a goroutine
		go func() {
			if err := sseServer.Start(*addr); err != nil {
				fmt.Printf("SSE server error: %v\n", err)
				cancel() // Cancel the context to trigger shutdown
			}
		}()

		fmt.Printf("SSE server listening on %s\n", *addr)
		fmt.Printf("Base URL: %s\n", *baseURL)
		fmt.Println("Press Ctrl+C to stop the server")

		// Wait for context cancellation (e.g., SIGINT or SIGTERM)
		<-ctx.Done()

		// Graceful shutdown
		fmt.Println("Shutting down server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := sseServer.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("Error during server shutdown: %v\n", err)
		}
	default:
		fmt.Printf("Unknown mode: %s. Supported modes are 'stdio' and 'sse'.\n", *mode)
		os.Exit(1)
	}
}
