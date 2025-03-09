package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerDocumentationTools registers all documentation related tools
func registerDocumentationTools(s *server.MCPServer) error {
	// Register search GCP documentation tool
	searchGCPDocs := mcp.NewTool("search_gcp_docs",
		mcp.WithDescription("Searches Google Cloud documentation"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The search query"),
		),
		mcp.WithNumber("max_results",
			mcp.Description("Maximum number of results to return (default: 5)"),
		),
	)

	searchGCPDocsHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSearchGCPDocs(ctx, request)
	}

	AddToolSafe(s, searchGCPDocs, searchGCPDocsHandler)

	// Register search Kubernetes documentation tool
	searchK8sDocs := mcp.NewTool("search_k8s_docs",
		mcp.WithDescription("Searches Kubernetes documentation"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The search query"),
		),
		mcp.WithNumber("max_results",
			mcp.Description("Maximum number of results to return (default: 5)"),
		),
	)

	searchK8sDocsHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSearchK8sDocs(ctx, request)
	}

	AddToolSafe(s, searchK8sDocs, searchK8sDocsHandler)

	// Register get error documentation tool
	getErrorDocs := mcp.NewTool("get_error_docs",
		mcp.WithDescription("Gets documentation for a specific error code or message"),
		mcp.WithString("error_code",
			mcp.Description("The error code to look up"),
		),
		mcp.WithString("error_message",
			mcp.Description("The error message to look up"),
		),
	)

	getErrorDocsHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetErrorDocs(ctx, request)
	}

	AddToolSafe(s, getErrorDocs, getErrorDocsHandler)

	return nil
}

// handleSearchGCPDocs handles the search_gcp_docs tool request
func handleSearchGCPDocs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	query, ok := request.Params.Arguments["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query must be a non-empty string"), nil
	}

	// Get optional parameters with defaults
	maxResults := 5.0
	if val, ok := request.Params.Arguments["max_results"].(float64); ok && val > 0 {
		maxResults = val
	}

	// In a real implementation, you would use the Google Custom Search API
	// with a custom search engine configured for GCP documentation
	// For demonstration purposes, we'll simulate the search results

	simulatedResults := []struct {
		Title       string
		Link        string
		Snippet     string
		DisplayLink string
	}{
		{
			Title:       "Error Reporting | Google Cloud",
			Link:        "https://cloud.google.com/error-reporting",
			Snippet:     "Error Reporting counts, analyzes, and aggregates the crashes in your running cloud services.",
			DisplayLink: "cloud.google.com",
		},
		{
			Title:       "Monitoring | Google Cloud",
			Link:        "https://cloud.google.com/monitoring",
			Snippet:     "Gain visibility into the performance, availability, and health of your applications and infrastructure.",
			DisplayLink: "cloud.google.com",
		},
		{
			Title:       "Logging | Google Cloud",
			Link:        "https://cloud.google.com/logging",
			Snippet:     "Logging allows you to store, search, analyze, monitor, and alert on log data and events from Google Cloud and Amazon Web Services.",
			DisplayLink: "cloud.google.com",
		},
		{
			Title:       "Kubernetes Engine | Google Cloud",
			Link:        "https://cloud.google.com/kubernetes-engine",
			Snippet:     "Google Kubernetes Engine (GKE) is a managed, production-ready environment for running containerized applications.",
			DisplayLink: "cloud.google.com",
		},
		{
			Title:       "Troubleshooting GKE | Google Cloud",
			Link:        "https://cloud.google.com/kubernetes-engine/docs/troubleshooting",
			Snippet:     "This page provides troubleshooting information for common issues that you might encounter when using Google Kubernetes Engine.",
			DisplayLink: "cloud.google.com",
		},
	}

	// Filter results based on the query
	var filteredResults []struct {
		Title       string
		Link        string
		Snippet     string
		DisplayLink string
	}

	queryLower := strings.ToLower(query)
	for _, result := range simulatedResults {
		if strings.Contains(strings.ToLower(result.Title), queryLower) ||
			strings.Contains(strings.ToLower(result.Snippet), queryLower) {
			filteredResults = append(filteredResults, result)
		}
	}

	// Format the results
	var result string
	if len(filteredResults) == 0 {
		result = fmt.Sprintf("No documentation found for query: %s", query)
	} else {
		result = fmt.Sprintf("# Google Cloud Documentation Search Results for \"%s\"\n\n", query)

		for i, searchResult := range filteredResults {
			if i >= int(maxResults) {
				break
			}

			result += fmt.Sprintf("## %d. %s\n", i+1, searchResult.Title)
			result += fmt.Sprintf("**URL**: [%s](%s)\n\n", searchResult.Link, searchResult.Link)
			result += fmt.Sprintf("%s\n\n", searchResult.Snippet)
		}

		result += "For more results, visit the [Google Cloud documentation](https://cloud.google.com/docs)."
	}

	return mcp.NewToolResultText(result), nil
}

// handleSearchK8sDocs handles the search_k8s_docs tool request
func handleSearchK8sDocs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	query, ok := request.Params.Arguments["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query must be a non-empty string"), nil
	}

	// Get optional parameters with defaults
	maxResults := 5.0
	if val, ok := request.Params.Arguments["max_results"].(float64); ok && val > 0 {
		maxResults = val
	}

	// In a real implementation, you would use the Kubernetes documentation search API
	// or a custom search engine configured for Kubernetes documentation

	// For demonstration purposes, we'll simulate the search results
	simulatedResults := []struct {
		Title       string
		Link        string
		Snippet     string
		DisplayLink string
	}{
		{
			Title:       "Troubleshooting Clusters | Kubernetes",
			Link:        "https://kubernetes.io/docs/tasks/debug/debug-cluster/",
			Snippet:     "This guide is to help users debug applications that are deployed into Kubernetes and not behaving correctly.",
			DisplayLink: "kubernetes.io",
		},
		{
			Title:       "Troubleshooting Applications | Kubernetes",
			Link:        "https://kubernetes.io/docs/tasks/debug/debug-application/",
			Snippet:     "This guide is to help users debug applications that are deployed into Kubernetes and not behaving correctly.",
			DisplayLink: "kubernetes.io",
		},
		{
			Title:       "Debugging Pods | Kubernetes",
			Link:        "https://kubernetes.io/docs/tasks/debug/debug-application/debug-pods/",
			Snippet:     "This guide is to help users debug applications that are deployed into Kubernetes and not behaving correctly.",
			DisplayLink: "kubernetes.io",
		},
		{
			Title:       "Debugging Services | Kubernetes",
			Link:        "https://kubernetes.io/docs/tasks/debug/debug-application/debug-service/",
			Snippet:     "This guide is to help users debug applications that are deployed into Kubernetes and not behaving correctly.",
			DisplayLink: "kubernetes.io",
		},
		{
			Title:       "Debugging Init Containers | Kubernetes",
			Link:        "https://kubernetes.io/docs/tasks/debug/debug-application/debug-init-containers/",
			Snippet:     "This guide is to help users debug applications that are deployed into Kubernetes and not behaving correctly.",
			DisplayLink: "kubernetes.io",
		},
	}

	// Filter results based on the query
	var filteredResults []struct {
		Title       string
		Link        string
		Snippet     string
		DisplayLink string
	}

	queryLower := strings.ToLower(query)
	for _, result := range simulatedResults {
		if strings.Contains(strings.ToLower(result.Title), queryLower) ||
			strings.Contains(strings.ToLower(result.Snippet), queryLower) {
			filteredResults = append(filteredResults, result)
		}
	}

	// Format the results
	var result string
	if len(filteredResults) == 0 {
		result = fmt.Sprintf("No documentation found for query: %s", query)
	} else {
		result = fmt.Sprintf("# Kubernetes Documentation Search Results for \"%s\"\n\n", query)

		for i, searchResult := range filteredResults {
			if i >= int(maxResults) {
				break
			}

			result += fmt.Sprintf("## %d. %s\n", i+1, searchResult.Title)
			result += fmt.Sprintf("**URL**: [%s](%s)\n\n", searchResult.Link, searchResult.Link)
			result += fmt.Sprintf("%s\n\n", searchResult.Snippet)
		}

		result += "For more results, visit the [Kubernetes documentation](https://kubernetes.io/docs/)."
	}

	return mcp.NewToolResultText(result), nil
}

// handleGetErrorDocs handles the get_error_docs tool request
func handleGetErrorDocs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	errorCode, hasErrorCode := request.Params.Arguments["error_code"].(string)
	errorMessage, hasErrorMessage := request.Params.Arguments["error_message"].(string)

	if !hasErrorCode && !hasErrorMessage {
		return mcp.NewToolResultError("either error_code or error_message must be provided"), nil
	}

	// In a real implementation, you would query a database of known errors
	// or use an API to look up documentation for the error

	// For demonstration purposes, we'll simulate the error documentation
	errorDocs := map[string]struct {
		Title       string
		Description string
		Solution    string
		References  []string
	}{
		"RESOURCE_EXHAUSTED": {
			Title:       "Resource Exhausted Error",
			Description: "This error occurs when a resource quota has been exceeded. It typically happens when you've reached the limit for a particular resource in your Google Cloud project.",
			Solution:    "1. Check your current quota usage in the Google Cloud Console.\n2. Request a quota increase if needed.\n3. Optimize your resource usage to stay within limits.",
			References: []string{
				"https://cloud.google.com/docs/quota",
				"https://cloud.google.com/compute/docs/resource-quotas",
			},
		},
		"PERMISSION_DENIED": {
			Title:       "Permission Denied Error",
			Description: "This error occurs when the authenticated user does not have sufficient permissions to perform the requested operation.",
			Solution:    "1. Check the IAM permissions for the user or service account.\n2. Grant the necessary roles or permissions.\n3. Verify that the service account has the required scopes.",
			References: []string{
				"https://cloud.google.com/iam/docs/overview",
				"https://cloud.google.com/iam/docs/troubleshooting-access",
			},
		},
		"NOT_FOUND": {
			Title:       "Resource Not Found Error",
			Description: "This error occurs when the requested resource does not exist or is not accessible.",
			Solution:    "1. Verify that the resource name or ID is correct.\n2. Check if the resource exists in the specified project and region.\n3. Ensure that the resource hasn't been deleted.",
			References: []string{
				"https://cloud.google.com/apis/design/errors",
			},
		},
		"FAILED_PRECONDITION": {
			Title:       "Failed Precondition Error",
			Description: "This error occurs when the system is not in a state required for the operation's execution.",
			Solution:    "1. Check the current state of the resource.\n2. Ensure all prerequisites for the operation are met.\n3. Retry the operation after resolving any conflicts.",
			References: []string{
				"https://cloud.google.com/apis/design/errors",
			},
		},
		"DEADLINE_EXCEEDED": {
			Title:       "Deadline Exceeded Error",
			Description: "This error occurs when the operation took longer than the deadline specified by the client or the system.",
			Solution:    "1. Increase the timeout for the operation if possible.\n2. Break down large operations into smaller ones.\n3. Check for performance issues in your application.",
			References: []string{
				"https://cloud.google.com/apis/design/errors",
			},
		},
	}

	// Look up the error documentation
	var errorInfo struct {
		Title       string
		Description string
		Solution    string
		References  []string
	}

	var found bool

	if hasErrorCode {
		errorInfo, found = errorDocs[errorCode]
	}

	if !found && hasErrorMessage {
		// Search for the error message in the descriptions
		errorMessageLower := strings.ToLower(errorMessage)
		for _, doc := range errorDocs {
			if strings.Contains(strings.ToLower(doc.Description), errorMessageLower) {
				errorInfo = doc
				found = true
				break
			}
		}
	}

	// Format the results
	var result string
	if !found {
		result = "No documentation found for the specified error."

		if hasErrorCode {
			result += fmt.Sprintf(" Error code: %s", errorCode)
		}

		if hasErrorMessage {
			result += fmt.Sprintf(" Error message: %s", errorMessage)
		}

		result += "\n\nTry searching the Google Cloud documentation or Kubernetes documentation for more information."
	} else {
		result = fmt.Sprintf("# %s\n\n", errorInfo.Title)
		result += fmt.Sprintf("## Description\n\n%s\n\n", errorInfo.Description)
		result += fmt.Sprintf("## Solution\n\n%s\n\n", errorInfo.Solution)

		if len(errorInfo.References) > 0 {
			result += "## References\n\n"
			for _, ref := range errorInfo.References {
				result += fmt.Sprintf("- [%s](%s)\n", ref, ref)
			}
		}
	}

	return mcp.NewToolResultText(result), nil
}
