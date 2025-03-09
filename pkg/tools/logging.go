package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ivanvanderbyl/operable/pkg/auth"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GCP Logging API base URL
const gcpLoggingBaseURL = "https://logging.googleapis.com/v2"

// registerLoggingTools registers all logging related tools
func registerLoggingTools(s *server.MCPServer, authHandler *auth.OAuthHandler) error {
	// Register query logs tool
	queryLogs := mcp.NewTool("query_logs",
		mcp.WithDescription("Queries logs from GCP Cloud Logging"),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.Description("The Google Cloud project ID"),
		),
		mcp.WithString("filter",
			mcp.Required(),
			mcp.Description("The filter expression for the logs query"),
		),
		mcp.WithNumber("time_range_hours",
			mcp.Description("Time range for logs in hours (default: 1)"),
		),
		mcp.WithNumber("max_results",
			mcp.Description("Maximum number of results to return (default: 50)"),
		),
	)

	queryHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleQueryLogs(ctx, request, authHandler)
	}

	AddToolSafe(s, queryLogs, queryHandler)

	// Register get kubernetes pod logs tool
	getPodLogs := mcp.NewTool("get_pod_logs",
		mcp.WithDescription("Gets logs for a specific Kubernetes pod"),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.Description("The Google Cloud project ID"),
		),
		mcp.WithString("location",
			mcp.Required(),
			mcp.Description("The GKE cluster location"),
		),
		mcp.WithString("cluster_name",
			mcp.Required(),
			mcp.Description("The GKE cluster name"),
		),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("The Kubernetes namespace"),
		),
		mcp.WithString("pod_name",
			mcp.Required(),
			mcp.Description("The name of the pod"),
		),
		mcp.WithString("container_name",
			mcp.Description("The name of the container (if not provided, logs from all containers will be returned)"),
		),
		mcp.WithNumber("time_range_hours",
			mcp.Description("Time range for logs in hours (default: 1)"),
		),
		mcp.WithNumber("max_results",
			mcp.Description("Maximum number of results to return (default: 100)"),
		),
	)

	podLogsHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetPodLogs(ctx, request, authHandler)
	}

	AddToolSafe(s, getPodLogs, podLogsHandler)

	return nil
}

// handleQueryLogs handles the query_logs tool request
func handleQueryLogs(ctx context.Context, request mcp.CallToolRequest, authHandler *auth.OAuthHandler) (*mcp.CallToolResult, error) {
	// Extract parameters
	projectID, ok := request.Params.Arguments["project_id"].(string)
	if !ok || projectID == "" {
		return mcp.NewToolResultError("project_id must be a non-empty string"), nil
	}

	filter, ok := request.Params.Arguments["filter"].(string)
	if !ok || filter == "" {
		return mcp.NewToolResultError("filter must be a non-empty string"), nil
	}

	// Get optional parameters with defaults
	timeRangeHours := 1.0
	if val, ok := request.Params.Arguments["time_range_hours"].(float64); ok && val > 0 {
		timeRangeHours = val
	}

	maxResults := 50.0
	if val, ok := request.Params.Arguments["max_results"].(float64); ok && val > 0 {
		maxResults = val
	}

	// Get HTTP client with authentication
	client, err := authHandler.GetClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting authenticated client: %v", err)), nil
	}

	// Calculate time range
	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(timeRangeHours) * time.Hour)

	// Add time range to filter if not already present
	if !strings.Contains(filter, "timestamp") {
		filter = fmt.Sprintf(`%s AND timestamp >= "%s" AND timestamp <= "%s"`,
			filter,
			startTime.Format(time.RFC3339),
			endTime.Format(time.RFC3339))
	}

	// Construct the request body
	requestBody := map[string]interface{}{
		"resourceNames": []string{fmt.Sprintf("projects/%s", projectID)},
		"filter":        filter,
		"orderBy":       "timestamp desc",
		"pageSize":      int(maxResults),
	}

	requestBodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error marshaling request body: %v", err)), nil
	}

	// Construct URL for the Logging API
	apiURL := fmt.Sprintf("%s/entries:list", gcpLoggingBaseURL)

	// Make the API request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(requestBodyJSON)))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating request: %v", err)), nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error making request to Logging API: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("Error from Logging API: %s", resp.Status)), nil
	}

	// Parse the response
	var response struct {
		Entries []struct {
			LogName  string `json:"logName"`
			Resource struct {
				Type   string            `json:"type"`
				Labels map[string]string `json:"labels"`
			} `json:"resource"`
			Timestamp   string                 `json:"timestamp"`
			Severity    string                 `json:"severity"`
			TextPayload string                 `json:"textPayload"`
			JsonPayload map[string]interface{} `json:"jsonPayload"`
			Labels      map[string]string      `json:"labels"`
		} `json:"entries"`
		NextPageToken string `json:"nextPageToken"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error parsing response: %v", err)), nil
	}

	// Format the results
	var result string
	if len(response.Entries) == 0 {
		result = "No logs found matching the filter criteria."
	} else {
		result = fmt.Sprintf("Found %d log entries matching the filter criteria:\n\n", len(response.Entries))

		for i, entry := range response.Entries {
			result += fmt.Sprintf("### Log Entry %d\n", i+1)
			result += fmt.Sprintf("- **Timestamp**: %s\n", entry.Timestamp)
			result += fmt.Sprintf("- **Severity**: %s\n", entry.Severity)
			result += fmt.Sprintf("- **Log Name**: %s\n", entry.LogName)
			result += fmt.Sprintf("- **Resource Type**: %s\n", entry.Resource.Type)

			if len(entry.Resource.Labels) > 0 {
				result += "- **Resource Labels**:\n"
				for k, v := range entry.Resource.Labels {
					result += fmt.Sprintf("  - %s: %s\n", k, v)
				}
			}

			if len(entry.Labels) > 0 {
				result += "- **Labels**:\n"
				for k, v := range entry.Labels {
					result += fmt.Sprintf("  - %s: %s\n", k, v)
				}
			}

			result += "- **Payload**:\n"
			if entry.TextPayload != "" {
				result += "```\n" + entry.TextPayload + "\n```\n"
			} else if entry.JsonPayload != nil {
				jsonBytes, err := json.MarshalIndent(entry.JsonPayload, "", "  ")
				if err != nil {
					result += "Error formatting JSON payload\n"
				} else {
					result += "```json\n" + string(jsonBytes) + "\n```\n"
				}
			} else {
				result += "No payload\n"
			}

			result += "\n"
		}

		if response.NextPageToken != "" {
			result += "Note: There are more log entries available. Refine your filter or increase max_results to see more.\n"
		}
	}

	return mcp.NewToolResultText(result), nil
}

// handleGetPodLogs handles the get_pod_logs tool request
func handleGetPodLogs(ctx context.Context, request mcp.CallToolRequest, authHandler *auth.OAuthHandler) (*mcp.CallToolResult, error) {
	// Extract parameters
	projectID, ok := request.Params.Arguments["project_id"].(string)
	if !ok || projectID == "" {
		return mcp.NewToolResultError("project_id must be a non-empty string"), nil
	}

	location, ok := request.Params.Arguments["location"].(string)
	if !ok || location == "" {
		return mcp.NewToolResultError("location must be a non-empty string"), nil
	}

	clusterName, ok := request.Params.Arguments["cluster_name"].(string)
	if !ok || clusterName == "" {
		return mcp.NewToolResultError("cluster_name must be a non-empty string"), nil
	}

	namespace, ok := request.Params.Arguments["namespace"].(string)
	if !ok || namespace == "" {
		return mcp.NewToolResultError("namespace must be a non-empty string"), nil
	}

	podName, ok := request.Params.Arguments["pod_name"].(string)
	if !ok || podName == "" {
		return mcp.NewToolResultError("pod_name must be a non-empty string"), nil
	}

	// Get optional parameters
	containerName, _ := request.Params.Arguments["container_name"].(string)

	// Get optional parameters with defaults
	timeRangeHours := 1.0
	if val, ok := request.Params.Arguments["time_range_hours"].(float64); ok && val > 0 {
		timeRangeHours = val
	}

	maxResults := 100.0
	if val, ok := request.Params.Arguments["max_results"].(float64); ok && val > 0 {
		maxResults = val
	}

	// Get HTTP client with authentication
	client, err := authHandler.GetClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting authenticated client: %v", err)), nil
	}

	// Calculate time range
	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(timeRangeHours) * time.Hour)

	// Construct filter for pod logs
	filter := fmt.Sprintf(`resource.type="k8s_container"
		AND resource.labels.project_id="%s"
		AND resource.labels.location="%s"
		AND resource.labels.cluster_name="%s"
		AND resource.labels.namespace_name="%s"
		AND resource.labels.pod_name="%s"`,
		projectID, location, clusterName, namespace, podName)

	// Add container name to filter if provided
	if containerName != "" {
		filter += fmt.Sprintf(` AND resource.labels.container_name="%s"`, containerName)
	}

	// Add time range to filter
	filter += fmt.Sprintf(` AND timestamp >= "%s" AND timestamp <= "%s"`,
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339))

	// Construct the request body
	requestBody := map[string]interface{}{
		"resourceNames": []string{fmt.Sprintf("projects/%s", projectID)},
		"filter":        filter,
		"orderBy":       "timestamp desc",
		"pageSize":      int(maxResults),
	}

	requestBodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error marshaling request body: %v", err)), nil
	}

	// Construct URL for the Logging API
	apiURL := fmt.Sprintf("%s/entries:list", gcpLoggingBaseURL)

	// Make the API request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(requestBodyJSON)))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating request: %v", err)), nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error making request to Logging API: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("Error from Logging API: %s", resp.Status)), nil
	}

	// Parse the response
	var response struct {
		Entries []struct {
			Timestamp   string                 `json:"timestamp"`
			Severity    string                 `json:"severity"`
			TextPayload string                 `json:"textPayload"`
			JsonPayload map[string]interface{} `json:"jsonPayload"`
			Resource    struct {
				Labels map[string]string `json:"labels"`
			} `json:"resource"`
		} `json:"entries"`
		NextPageToken string `json:"nextPageToken"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error parsing response: %v", err)), nil
	}

	// Format the results
	var result string
	if len(response.Entries) == 0 {
		result = fmt.Sprintf("No logs found for pod %s in namespace %s.", podName, namespace)
	} else {
		// Get container name from the first entry if not provided
		if containerName == "" && len(response.Entries) > 0 {
			containerName = response.Entries[0].Resource.Labels["container_name"]
		}

		result = fmt.Sprintf("## Logs for pod %s", podName)
		if containerName != "" {
			result += fmt.Sprintf(", container %s", containerName)
		}
		result += fmt.Sprintf(" in namespace %s\n\n", namespace)

		result += fmt.Sprintf("Found %d log entries in the last %.1f hours:\n\n", len(response.Entries), timeRangeHours)

		result += "```\n"
		for i := len(response.Entries) - 1; i >= 0; i-- { // Reverse to show oldest first
			entry := response.Entries[i]

			// Format timestamp
			t, err := time.Parse(time.RFC3339, entry.Timestamp)
			timestamp := entry.Timestamp
			if err == nil {
				timestamp = t.Format("2006-01-02 15:04:05")
			}

			// Get container name
			entryContainer := entry.Resource.Labels["container_name"]

			// Format log line
			var logLine string
			if entry.TextPayload != "" {
				logLine = entry.TextPayload
			} else if entry.JsonPayload != nil {
				if msg, ok := entry.JsonPayload["message"]; ok {
					logLine = fmt.Sprintf("%v", msg)
				} else {
					jsonBytes, err := json.Marshal(entry.JsonPayload)
					if err == nil {
						logLine = string(jsonBytes)
					} else {
						logLine = "[complex json payload]"
					}
				}
			}

			// Add container name if multiple containers
			if containerName == "" {
				result += fmt.Sprintf("[%s] [%s] %s\n", timestamp, entryContainer, logLine)
			} else {
				result += fmt.Sprintf("[%s] %s\n", timestamp, logLine)
			}
		}
		result += "```\n\n"

		if response.NextPageToken != "" {
			result += "Note: There are more log entries available. Increase time_range_hours or max_results to see more.\n"
		}
	}

	return mcp.NewToolResultText(result), nil
}
