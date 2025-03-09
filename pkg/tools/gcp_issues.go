package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	errorreporting "cloud.google.com/go/errorreporting/apiv1beta1"
	"cloud.google.com/go/errorreporting/apiv1beta1/errorreportingpb"
	"github.com/ivanvanderbyl/operable/pkg/auth"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"google.golang.org/api/iterator"
)

// registerGCPIssuesTools registers all GCP issues related tools
func registerGCPIssuesTools(s *server.MCPServer, authHandler *auth.OAuthHandler) error {
	// Register list active issues tool
	listActiveIssues := mcp.NewTool("list_active_issues",
		mcp.WithDescription("Lists active issues from GCP Error Reporting"),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.Description("The Google Cloud project ID"),
		),
		mcp.WithNumber("time_range_hours",
			mcp.Description("Time range for issues in hours (default: 24)"),
		),
		mcp.WithNumber("max_results",
			mcp.Description("Maximum number of results to return (default: 10)"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListActiveIssues(ctx, request, authHandler)
	}

	// Register the tool using the safe wrapper
	AddToolSafe(s, listActiveIssues, handler)

	// Register get issue details tool
	getIssueDetails := mcp.NewTool("get_issue_details",
		mcp.WithDescription("Gets detailed information about a specific error group"),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.Description("The Google Cloud project ID"),
		),
		mcp.WithString("error_group_id",
			mcp.Required(),
			mcp.Description("The ID of the error group"),
		),
	)

	detailsHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetIssueDetails(ctx, request, authHandler)
	}

	// Register the tool using the safe wrapper
	AddToolSafe(s, getIssueDetails, detailsHandler)

	return nil
}

// handleListActiveIssues handles the list_active_issues tool request
func handleListActiveIssues(ctx context.Context, request mcp.CallToolRequest, authHandler *auth.OAuthHandler) (*mcp.CallToolResult, error) {
	// Extract parameters
	projectID, ok := request.Params.Arguments["project_id"].(string)
	if !ok || projectID == "" {
		return mcp.NewToolResultError("project_id must be a non-empty string"), nil
	}

	// Unused but kept for future use
	// timeRangeHours := 24.0
	// if val, ok := request.Params.Arguments["time_range_hours"].(float64); ok && val > 0 {
	// 	timeRangeHours = val
	// }

	maxResults := int32(10)
	if val, ok := request.Params.Arguments["max_results"].(float64); ok && val > 0 {
		maxResults = int32(val)
	}

	// Get client options
	opts, err := authHandler.GetClientOptions(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting client options: %v", err)), nil
	}

	// Create error reporting client
	client, err := errorreporting.NewErrorStatsClient(ctx, opts...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating Error Reporting client: %v", err)), nil
	}
	defer client.Close()

	// Create list group stats request
	req := &errorreportingpb.ListGroupStatsRequest{
		ProjectName: fmt.Sprintf("projects/%s", projectID),
		TimeRange: &errorreportingpb.QueryTimeRange{
			Period: errorreportingpb.QueryTimeRange_PERIOD_1_DAY,
		},
		PageSize: maxResults,
		// The GCP SDK uses different enum names than the raw API
		// Here we're ordering by count (most frequent first)
		Alignment: errorreportingpb.TimedCountAlignment_ALIGNMENT_EQUAL_ROUNDED,
	}

	// Call the API
	groupStatsIterator := client.ListGroupStats(ctx, req)

	// Process results
	var errorGroupStats []*errorreportingpb.ErrorGroupStats
	for {
		stat, err := groupStatsIterator.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error iterating through error groups: %v", err)), nil
		}
		errorGroupStats = append(errorGroupStats, stat)
	}

	// Format the results
	var result string
	if len(errorGroupStats) == 0 {
		result = "No active issues found in the specified time range."
	} else {
		result = fmt.Sprintf("Found %d active issues in project %s:\n\n", len(errorGroupStats), projectID)

		for i, stat := range errorGroupStats {
			// Extract the group ID from the name (e.g., "projects/my-project/groups/some-group-id")
			groupIDParts := strings.Split(stat.Group.Name, "/")
			groupID := groupIDParts[len(groupIDParts)-1]

			result += fmt.Sprintf("%d. Error Group: %s\n", i+1, groupID)
			result += fmt.Sprintf("   Count: %d occurrences\n", stat.Count)

			if stat.FirstSeenTime != nil {
				firstSeen := stat.FirstSeenTime.AsTime()
				result += fmt.Sprintf("   First seen: %s\n", firstSeen.Format(time.RFC3339))
			}

			if stat.LastSeenTime != nil {
				lastSeen := stat.LastSeenTime.AsTime()
				result += fmt.Sprintf("   Last seen: %s\n", lastSeen.Format(time.RFC3339))
			}

			if len(stat.AffectedServices) > 0 {
				result += "   Affected services:\n"
				for _, svc := range stat.AffectedServices {
					result += fmt.Sprintf("     - %s (version: %s)\n", svc.Service, svc.Version)
				}
			}

			result += "\n"
		}

		result += "To get more details about a specific error group, use the get_issue_details tool."
	}

	return mcp.NewToolResultText(result), nil
}

// handleGetIssueDetails handles the get_issue_details tool request
func handleGetIssueDetails(ctx context.Context, request mcp.CallToolRequest, authHandler *auth.OAuthHandler) (*mcp.CallToolResult, error) {
	// Extract parameters
	projectID, ok := request.Params.Arguments["project_id"].(string)
	if !ok || projectID == "" {
		return mcp.NewToolResultError("project_id must be a non-empty string"), nil
	}

	errorGroupID, ok := request.Params.Arguments["error_group_id"].(string)
	if !ok || errorGroupID == "" {
		return mcp.NewToolResultError("error_group_id must be a non-empty string"), nil
	}

	// Get client options
	opts, err := authHandler.GetClientOptions(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting client options: %v", err)), nil
	}

	// Create error reporting client
	errClient, err := errorreporting.NewErrorStatsClient(ctx, opts...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating Error Reporting client: %v", err)), nil
	}
	defer errClient.Close()

	// Get errors in the group
	errorsRequest := &errorreportingpb.ListEventsRequest{
		ProjectName: fmt.Sprintf("projects/%s", projectID),
		GroupId:     errorGroupID,
		PageSize:    10,
	}

	eventsIterator := errClient.ListEvents(ctx, errorsRequest)

	// Process results
	var errorEvents []*errorreportingpb.ErrorEvent
	for {
		event, err := eventsIterator.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error iterating through error events: %v", err)), nil
		}
		errorEvents = append(errorEvents, event)
	}

	// Format the results
	result := fmt.Sprintf("## Error Group: %s\n\n", errorGroupID)

	// We don't have group information, so we'll skip tracking issues

	result += "### Recent Error Events\n\n"

	if len(errorEvents) == 0 {
		result += "No recent error events found.\n"
	} else {
		for i, event := range errorEvents {
			result += fmt.Sprintf("#### Event %d\n", i+1)

			if event.EventTime != nil {
				eventTime := event.EventTime.AsTime()
				result += fmt.Sprintf("- Time: %s\n", eventTime.Format(time.RFC3339))
			}

			if event.ServiceContext != nil {
				result += fmt.Sprintf("- Service: %s (version: %s)\n",
					event.ServiceContext.Service, event.ServiceContext.Version)
			}

			if event.Context != nil && event.Context.ReportLocation != nil {
				result += fmt.Sprintf("- Location: %s:%d in %s\n",
					event.Context.ReportLocation.FilePath,
					event.Context.ReportLocation.LineNumber,
					event.Context.ReportLocation.FunctionName)
			}

			if event.Context != nil && event.Context.HttpRequest != nil {
				result += fmt.Sprintf("- Request: %s %s\n",
					event.Context.HttpRequest.Method,
					event.Context.HttpRequest.Url)

				if event.Context.HttpRequest.RemoteIp != "" {
					result += fmt.Sprintf("  - IP: %s\n", event.Context.HttpRequest.RemoteIp)
				}

				if event.Context.HttpRequest.UserAgent != "" {
					result += fmt.Sprintf("  - User-Agent: %s\n", event.Context.HttpRequest.UserAgent)
				}

				if event.Context.HttpRequest.Referrer != "" {
					result += fmt.Sprintf("  - Referrer: %s\n", event.Context.HttpRequest.Referrer)
				}
			}

			if event.Message != "" {
				result += "- Error Message:\n```\n" + event.Message + "\n```\n"
			}

			result += "\n"
		}
	}

	// Add suggested actions
	result += "### Potential Causes and Solutions\n\n"
	result += "1. Check the error messages and stack traces for clues about the root cause.\n"
	result += "2. Look for patterns in the affected services and versions.\n"
	result += "3. Check recent deployments or changes to affected services.\n"
	result += "4. Examine logs around the time of the errors for related issues.\n"
	result += "5. Consider temporary mitigations like rolling back to a previous version if errors persist.\n"

	return mcp.NewToolResultText(result), nil
}
