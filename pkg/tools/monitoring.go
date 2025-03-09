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

// GCP Monitoring API base URL
const gcpMonitoringBaseURL = "https://monitoring.googleapis.com/v3"

// registerMonitoringTools registers all monitoring related tools
func registerMonitoringTools(s *server.MCPServer, authHandler *auth.OAuthHandler) error {
	// Register query metrics tool
	queryMetrics := mcp.NewTool("query_metrics",
		mcp.WithDescription("Queries metrics from GCP Cloud Monitoring"),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.Description("The Google Cloud project ID"),
		),
		mcp.WithString("metric_type",
			mcp.Required(),
			mcp.Description("The metric type to query (e.g., kubernetes.io/container/cpu/utilization)"),
		),
		mcp.WithString("filter",
			mcp.Description("Additional filter for the metrics query"),
		),
		mcp.WithNumber("time_range_hours",
			mcp.Description("Time range for metrics in hours (default: 1)"),
		),
		mcp.WithNumber("alignment_period_seconds",
			mcp.Description("Alignment period in seconds (default: 300)"),
		),
	)

	queryMetricsHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleQueryMetrics(ctx, request, authHandler)
	}

	AddToolSafe(s, queryMetrics, queryMetricsHandler)

	// Register list active alerts tool
	listAlerts := mcp.NewTool("list_alerts",
		mcp.WithDescription("Lists active alerts from GCP Cloud Monitoring"),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.Description("The Google Cloud project ID"),
		),
		mcp.WithString("filter",
			mcp.Description("Additional filter for the alerts query"),
		),
	)

	listAlertsHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListAlerts(ctx, request, authHandler)
	}

	AddToolSafe(s, listAlerts, listAlertsHandler)

	return nil
}

// handleQueryMetrics handles the query_metrics tool request
func handleQueryMetrics(ctx context.Context, request mcp.CallToolRequest, authHandler *auth.OAuthHandler) (*mcp.CallToolResult, error) {
	// Extract parameters
	projectID, ok := request.Params.Arguments["project_id"].(string)
	if !ok || projectID == "" {
		return mcp.NewToolResultError("project_id must be a non-empty string"), nil
	}

	metricType, ok := request.Params.Arguments["metric_type"].(string)
	if !ok || metricType == "" {
		return mcp.NewToolResultError("metric_type must be a non-empty string"), nil
	}

	// Get optional parameters
	filter, _ := request.Params.Arguments["filter"].(string)

	// Get optional parameters with defaults
	timeRangeHours := 1.0
	if val, ok := request.Params.Arguments["time_range_hours"].(float64); ok && val > 0 {
		timeRangeHours = val
	}

	alignmentPeriodSeconds := 300.0
	if val, ok := request.Params.Arguments["alignment_period_seconds"].(float64); ok && val > 0 {
		alignmentPeriodSeconds = val
	}

	// Get HTTP client with authentication
	client, err := authHandler.GetClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting authenticated client: %v", err)), nil
	}

	// Calculate time range
	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(timeRangeHours) * time.Hour)

	// Construct the request body
	requestBody := map[string]interface{}{
		"metricDescriptor": map[string]string{
			"type": metricType,
		},
		"aggregation": map[string]interface{}{
			"alignmentPeriod":    fmt.Sprintf("%.0fs", alignmentPeriodSeconds),
			"perSeriesAligner":   "ALIGN_MEAN",
			"crossSeriesReducer": "REDUCE_MEAN",
		},
		"interval": map[string]string{
			"startTime": startTime.Format(time.RFC3339),
			"endTime":   endTime.Format(time.RFC3339),
		},
	}

	// Add filter if provided
	if filter != "" {
		requestBody["filter"] = filter
	}

	requestBodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error marshaling request body: %v", err)), nil
	}

	// Construct URL for the Monitoring API
	apiURL := fmt.Sprintf("%s/projects/%s/timeSeries:query", gcpMonitoringBaseURL, projectID)

	// Make the API request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(requestBodyJSON)))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating request: %v", err)), nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error making request to Monitoring API: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("Error from Monitoring API: %s", resp.Status)), nil
	}

	// Parse the response
	var response struct {
		TimeSeriesData []struct {
			LabelValues []struct {
				StringValue string `json:"stringValue"`
				BoolValue   bool   `json:"boolValue"`
				Int64Value  string `json:"int64Value"`
			} `json:"labelValues"`
			PointData []struct {
				Values []struct {
					DoubleValue float64 `json:"doubleValue"`
					Int64Value  string  `json:"int64Value"`
					BoolValue   bool    `json:"boolValue"`
					StringValue string  `json:"stringValue"`
				} `json:"values"`
				TimeInterval struct {
					StartTime string `json:"startTime"`
					EndTime   string `json:"endTime"`
				} `json:"timeInterval"`
			} `json:"pointData"`
		} `json:"timeSeriesData"`
		TimeSeriesDescriptor struct {
			LabelDescriptors []struct {
				Key         string `json:"key"`
				ValueType   string `json:"valueType"`
				Description string `json:"description"`
			} `json:"labelDescriptors"`
			PointDescriptors []struct {
				Key         string `json:"key"`
				ValueType   string `json:"valueType"`
				MetricKind  string `json:"metricKind"`
				Unit        string `json:"unit"`
				Description string `json:"description"`
			} `json:"pointDescriptors"`
		} `json:"timeSeriesDescriptor"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error parsing response: %v", err)), nil
	}

	// Format the results
	var result string
	if len(response.TimeSeriesData) == 0 {
		result = fmt.Sprintf("No metrics data found for metric type %s in the specified time range.", metricType)
	} else {
		result = fmt.Sprintf("# Metrics Data for %s\n\n", metricType)

		// Get label keys
		labelKeys := make([]string, len(response.TimeSeriesDescriptor.LabelDescriptors))
		for i, ld := range response.TimeSeriesDescriptor.LabelDescriptors {
			labelKeys[i] = ld.Key
		}

		// Get point keys
		pointKeys := make([]string, len(response.TimeSeriesDescriptor.PointDescriptors))
		for i, pd := range response.TimeSeriesDescriptor.PointDescriptors {
			pointKeys[i] = pd.Key
		}

		// Format each time series
		for i, ts := range response.TimeSeriesData {
			result += fmt.Sprintf("## Time Series %d\n\n", i+1)

			// Format labels
			result += "### Labels\n\n"
			for j, lv := range ts.LabelValues {
				if j < len(labelKeys) {
					var value string
					if lv.StringValue != "" {
						value = lv.StringValue
					} else if lv.Int64Value != "" {
						value = lv.Int64Value
					} else {
						value = fmt.Sprintf("%t", lv.BoolValue)
					}
					result += fmt.Sprintf("- **%s**: %s\n", labelKeys[j], value)
				}
			}

			// Format points
			result += "\n### Data Points\n\n"
			if len(ts.PointData) == 0 {
				result += "No data points available.\n"
			} else {
				result += "| Time | Value |\n"
				result += "| ---- | ----- |\n"

				for _, pd := range ts.PointData {
					// Format time
					endTime, err := time.Parse(time.RFC3339, pd.TimeInterval.EndTime)
					timeStr := pd.TimeInterval.EndTime
					if err == nil {
						timeStr = endTime.Format("2006-01-02 15:04:05")
					}

					// Format value
					var valueStr string
					if len(pd.Values) > 0 {
						if pd.Values[0].DoubleValue != 0 {
							valueStr = fmt.Sprintf("%.6f", pd.Values[0].DoubleValue)
						} else if pd.Values[0].Int64Value != "" {
							valueStr = pd.Values[0].Int64Value
						} else if pd.Values[0].StringValue != "" {
							valueStr = pd.Values[0].StringValue
						} else {
							valueStr = fmt.Sprintf("%t", pd.Values[0].BoolValue)
						}
					} else {
						valueStr = "N/A"
					}

					result += fmt.Sprintf("| %s | %s |\n", timeStr, valueStr)
				}
			}

			result += "\n"
		}
	}

	return mcp.NewToolResultText(result), nil
}

// handleListAlerts handles the list_alerts tool request
func handleListAlerts(ctx context.Context, request mcp.CallToolRequest, authHandler *auth.OAuthHandler) (*mcp.CallToolResult, error) {
	// Extract parameters
	projectID, ok := request.Params.Arguments["project_id"].(string)
	if !ok || projectID == "" {
		return mcp.NewToolResultError("project_id must be a non-empty string"), nil
	}

	// Get optional parameters
	filter, _ := request.Params.Arguments["filter"].(string)

	// Get HTTP client with authentication
	client, err := authHandler.GetClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting authenticated client: %v", err)), nil
	}

	// Construct URL for the Monitoring API
	apiURL := fmt.Sprintf("%s/projects/%s/alertPolicies", gcpMonitoringBaseURL, projectID)
	if filter != "" {
		apiURL += fmt.Sprintf("?filter=%s", filter)
	}

	// Make the API request for alert policies
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating request: %v", err)), nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error making request to Monitoring API: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("Error from Monitoring API: %s", resp.Status)), nil
	}

	// Parse the response
	var policiesResponse struct {
		AlertPolicies []struct {
			Name          string `json:"name"`
			DisplayName   string `json:"displayName"`
			Documentation struct {
				Content string `json:"content"`
			} `json:"documentation"`
			Conditions []struct {
				Name        string `json:"name"`
				DisplayName string `json:"displayName"`
				Severity    string `json:"severity"`
			} `json:"conditions"`
			Enabled bool `json:"enabled"`
		} `json:"alertPolicies"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&policiesResponse); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error parsing alert policies response: %v", err)), nil
	}

	// Get active incidents
	incidentsURL := fmt.Sprintf("%s/projects/%s/incidents", gcpMonitoringBaseURL, projectID)

	incidentsReq, err := http.NewRequestWithContext(ctx, "GET", incidentsURL, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating incidents request: %v", err)), nil
	}

	incidentsResp, err := client.Do(incidentsReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error making request to Monitoring API for incidents: %v", err)), nil
	}
	defer incidentsResp.Body.Close()

	if incidentsResp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("Error from Monitoring API for incidents: %s", incidentsResp.Status)), nil
	}

	// Parse the incidents response
	var incidentsResponse struct {
		Incidents []struct {
			Name                string `json:"name"`
			ResourceName        string `json:"resourceName"`
			PolicyName          string `json:"policyName"`
			ConditionName       string `json:"conditionName"`
			StartTime           string `json:"startTime"`
			EndTime             string `json:"endTime"`
			State               string `json:"state"`
			Summary             string `json:"summary"`
			Documentation       string `json:"documentation"`
			Severity            string `json:"severity"`
			ResourceDisplayName string `json:"resourceDisplayName"`
		} `json:"incidents"`
	}

	if err := json.NewDecoder(incidentsResp.Body).Decode(&incidentsResponse); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error parsing incidents response: %v", err)), nil
	}

	// Create a map of policy names to policies for quick lookup
	policyMap := make(map[string]struct {
		DisplayName   string
		Documentation string
		Conditions    map[string]string
	})

	for _, policy := range policiesResponse.AlertPolicies {
		policyInfo := struct {
			DisplayName   string
			Documentation string
			Conditions    map[string]string
		}{
			DisplayName:   policy.DisplayName,
			Documentation: policy.Documentation.Content,
			Conditions:    make(map[string]string),
		}

		for _, condition := range policy.Conditions {
			policyInfo.Conditions[condition.Name] = condition.DisplayName
		}

		policyMap[policy.Name] = policyInfo
	}

	// Format the results
	var result string
	activeIncidents := 0
	for _, incident := range incidentsResponse.Incidents {
		if incident.State == "OPEN" {
			activeIncidents++
		}
	}

	if activeIncidents == 0 {
		result = "No active alerts found."
	} else {
		result = fmt.Sprintf("# Active Alerts in Project %s\n\n", projectID)
		result += fmt.Sprintf("Found %d active alerts:\n\n", activeIncidents)

		for i, incident := range incidentsResponse.Incidents {
			if incident.State != "OPEN" {
				continue
			}

			result += fmt.Sprintf("## %d. Alert: %s\n\n", i+1, incident.ResourceDisplayName)

			// Get policy and condition info
			policyInfo, hasPolicyInfo := policyMap[incident.PolicyName]
			policyDisplayName := "Unknown Policy"
			conditionDisplayName := "Unknown Condition"
			documentation := ""

			if hasPolicyInfo {
				policyDisplayName = policyInfo.DisplayName
				documentation = policyInfo.Documentation

				if condName, hasCondition := policyInfo.Conditions[incident.ConditionName]; hasCondition {
					conditionDisplayName = condName
				}
			}

			// Format incident details
			result += fmt.Sprintf("- **Policy**: %s\n", policyDisplayName)
			result += fmt.Sprintf("- **Condition**: %s\n", conditionDisplayName)
			result += fmt.Sprintf("- **Severity**: %s\n", incident.Severity)
			result += fmt.Sprintf("- **Started**: %s\n", formatTime(incident.StartTime))

			if incident.Summary != "" {
				result += fmt.Sprintf("- **Summary**: %s\n", incident.Summary)
			}

			if documentation != "" {
				result += "\n### Documentation\n\n"
				result += documentation + "\n"
			}

			result += "\n"
		}

		result += "## Recommended Actions\n\n"
		result += "1. Check the affected resources for any recent changes or deployments\n"
		result += "2. Review logs around the time the alert was triggered\n"
		result += "3. Check for related alerts that might indicate a broader issue\n"
		result += "4. Verify resource utilization and performance metrics\n"
		result += "5. Consider scaling resources if the alert is related to resource constraints\n"
	}

	return mcp.NewToolResultText(result), nil
}

// formatTime formats a RFC3339 time string to a more readable format
func formatTime(timeStr string) string {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return timeStr
	}
	return t.Format("2006-01-02 15:04:05")
}
