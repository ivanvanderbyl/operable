package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ivanvanderbyl/arnold/pkg/auth"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// GCP Container API base URL
const gcpContainerBaseURL = "https://container.googleapis.com/v1"

// registerKubernetesTools registers all Kubernetes related tools
func registerKubernetesTools(s *server.MCPServer, authHandler *auth.OAuthHandler) error {
	// Register list clusters tool
	listClusters := mcp.NewTool("list_clusters",
		mcp.WithDescription("Lists GKE clusters in a project"),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.Description("The Google Cloud project ID"),
		),
		mcp.WithString("location",
			mcp.Description("The location to list clusters from (optional, if not provided, all locations will be queried)"),
		),
	)

	listClustersHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListClusters(ctx, request, authHandler)
	}

	AddToolSafe(s, listClusters, listClustersHandler)

	// Register get cluster info tool
	getClusterInfo := mcp.NewTool("get_cluster_info",
		mcp.WithDescription("Gets detailed information about a GKE cluster"),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.Description("The Google Cloud project ID"),
		),
		mcp.WithString("location",
			mcp.Required(),
			mcp.Description("The location of the cluster"),
		),
		mcp.WithString("cluster_name",
			mcp.Required(),
			mcp.Description("The name of the cluster"),
		),
	)

	getClusterInfoHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetClusterInfo(ctx, request, authHandler)
	}

	AddToolSafe(s, getClusterInfo, getClusterInfoHandler)

	// Register list node pools tool
	listNodePools := mcp.NewTool("list_node_pools",
		mcp.WithDescription("Lists node pools in a GKE cluster"),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.Description("The Google Cloud project ID"),
		),
		mcp.WithString("location",
			mcp.Required(),
			mcp.Description("The location of the cluster"),
		),
		mcp.WithString("cluster_name",
			mcp.Required(),
			mcp.Description("The name of the cluster"),
		),
	)

	listNodePoolsHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListNodePools(ctx, request, authHandler)
	}

	AddToolSafe(s, listNodePools, listNodePoolsHandler)

	return nil
}

// handleListClusters handles the list_clusters tool request
func handleListClusters(ctx context.Context, request mcp.CallToolRequest, authHandler *auth.OAuthHandler) (*mcp.CallToolResult, error) {
	// Extract parameters
	projectID, ok := request.Params.Arguments["project_id"].(string)
	if !ok || projectID == "" {
		return mcp.NewToolResultError("project_id must be a non-empty string"), nil
	}

	location, _ := request.Params.Arguments["location"].(string)

	// Get HTTP client with authentication
	client, err := authHandler.GetClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting authenticated client: %v", err)), nil
	}

	// Construct URL for the Container API
	var apiURL string
	if location == "" {
		apiURL = fmt.Sprintf("%s/projects/%s/locations/-/clusters", gcpContainerBaseURL, projectID)
	} else {
		apiURL = fmt.Sprintf("%s/projects/%s/locations/%s/clusters", gcpContainerBaseURL, projectID, location)
	}

	// Make the API request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating request: %v", err)), nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error making request to Container API: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("Error from Container API: %s", resp.Status)), nil
	}

	// Parse the response
	var response struct {
		Clusters []struct {
			Name             string `json:"name"`
			Description      string `json:"description"`
			Location         string `json:"location"`
			Status           string `json:"status"`
			NodeCount        int    `json:"currentNodeCount"`
			MasterVersion    string `json:"currentMasterVersion"`
			NodeVersion      string `json:"currentNodeVersion"`
			Network          string `json:"network"`
			Subnetwork       string `json:"subnetwork"`
			ClusterIpv4Cidr  string `json:"clusterIpv4Cidr"`
			ServicesIpv4Cidr string `json:"servicesIpv4Cidr"`
			Endpoint         string `json:"endpoint"`
			CreateTime       string `json:"createTime"`
		} `json:"clusters"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error parsing response: %v", err)), nil
	}

	// Format the results
	var result string
	if len(response.Clusters) == 0 {
		result = fmt.Sprintf("No GKE clusters found in project %s", projectID)
		if location != "" {
			result += fmt.Sprintf(" in location %s", location)
		}
		result += "."
	} else {
		result = fmt.Sprintf("Found %d GKE clusters in project %s", len(response.Clusters), projectID)
		if location != "" {
			result += fmt.Sprintf(" in location %s", location)
		}
		result += ":\n\n"

		for i, cluster := range response.Clusters {
			result += fmt.Sprintf("### %d. Cluster: %s\n", i+1, cluster.Name)
			result += fmt.Sprintf("- **Location**: %s\n", cluster.Location)
			result += fmt.Sprintf("- **Status**: %s\n", cluster.Status)
			result += fmt.Sprintf("- **Node Count**: %d\n", cluster.NodeCount)
			result += fmt.Sprintf("- **Kubernetes Version**: %s (master) / %s (nodes)\n",
				cluster.MasterVersion, cluster.NodeVersion)
			result += fmt.Sprintf("- **Endpoint**: %s\n", cluster.Endpoint)
			result += fmt.Sprintf("- **Network**: %s\n", cluster.Network)
			result += fmt.Sprintf("- **Subnetwork**: %s\n", cluster.Subnetwork)
			result += fmt.Sprintf("- **Pod CIDR**: %s\n", cluster.ClusterIpv4Cidr)
			result += fmt.Sprintf("- **Service CIDR**: %s\n", cluster.ServicesIpv4Cidr)
			result += fmt.Sprintf("- **Created**: %s\n", cluster.CreateTime)

			if cluster.Description != "" {
				result += fmt.Sprintf("- **Description**: %s\n", cluster.Description)
			}

			result += "\n"
		}
	}

	return mcp.NewToolResultText(result), nil
}

// handleGetClusterInfo handles the get_cluster_info tool request
func handleGetClusterInfo(ctx context.Context, request mcp.CallToolRequest, authHandler *auth.OAuthHandler) (*mcp.CallToolResult, error) {
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

	// Get HTTP client with authentication
	client, err := authHandler.GetClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting authenticated client: %v", err)), nil
	}

	// Construct URL for the Container API
	apiURL := fmt.Sprintf("%s/projects/%s/locations/%s/clusters/%s",
		gcpContainerBaseURL, projectID, location, clusterName)

	// Make the API request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating request: %v", err)), nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error making request to Container API: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("Error from Container API: %s", resp.Status)), nil
	}

	// Parse the response
	var cluster struct {
		Name              string `json:"name"`
		Description       string `json:"description"`
		Location          string `json:"location"`
		Status            string `json:"status"`
		NodeCount         int    `json:"currentNodeCount"`
		MasterVersion     string `json:"currentMasterVersion"`
		NodeVersion       string `json:"currentNodeVersion"`
		Network           string `json:"network"`
		Subnetwork        string `json:"subnetwork"`
		ClusterIpv4Cidr   string `json:"clusterIpv4Cidr"`
		ServicesIpv4Cidr  string `json:"servicesIpv4Cidr"`
		Endpoint          string `json:"endpoint"`
		CreateTime        string `json:"createTime"`
		MaintenancePolicy struct {
			Window struct {
				DailyMaintenanceWindow struct {
					StartTime string `json:"startTime"`
					Duration  string `json:"duration"`
				} `json:"dailyMaintenanceWindow"`
			} `json:"maintenanceWindow"`
		} `json:"maintenancePolicy"`
		NetworkConfig struct {
			Network    string `json:"network"`
			Subnetwork string `json:"subnetwork"`
		} `json:"networkConfig"`
		AddonsConfig struct {
			HttpLoadBalancing struct {
				Disabled bool `json:"disabled"`
			} `json:"httpLoadBalancing"`
			HorizontalPodAutoscaling struct {
				Disabled bool `json:"disabled"`
			} `json:"horizontalPodAutoscaling"`
			KubernetesDashboard struct {
				Disabled bool `json:"disabled"`
			} `json:"kubernetesDashboard"`
			NetworkPolicyConfig struct {
				Disabled bool `json:"disabled"`
			} `json:"networkPolicyConfig"`
		} `json:"addonsConfig"`
		Locations      []string          `json:"locations"`
		ResourceLabels map[string]string `json:"resourceLabels"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&cluster); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error parsing response: %v", err)), nil
	}

	// Format the results
	result := fmt.Sprintf("# GKE Cluster: %s\n\n", cluster.Name)

	result += "## Basic Information\n\n"
	result += fmt.Sprintf("- **Location**: %s\n", cluster.Location)
	result += fmt.Sprintf("- **Status**: %s\n", cluster.Status)
	result += fmt.Sprintf("- **Node Count**: %d\n", cluster.NodeCount)
	result += fmt.Sprintf("- **Kubernetes Version**: %s (master) / %s (nodes)\n",
		cluster.MasterVersion, cluster.NodeVersion)
	result += fmt.Sprintf("- **Endpoint**: %s\n", cluster.Endpoint)
	result += fmt.Sprintf("- **Created**: %s\n", cluster.CreateTime)

	if cluster.Description != "" {
		result += fmt.Sprintf("- **Description**: %s\n", cluster.Description)
	}

	result += "\n## Network Configuration\n\n"
	result += fmt.Sprintf("- **Network**: %s\n", cluster.Network)
	result += fmt.Sprintf("- **Subnetwork**: %s\n", cluster.Subnetwork)
	result += fmt.Sprintf("- **Pod CIDR**: %s\n", cluster.ClusterIpv4Cidr)
	result += fmt.Sprintf("- **Service CIDR**: %s\n", cluster.ServicesIpv4Cidr)

	result += "\n## Add-ons Configuration\n\n"
	result += fmt.Sprintf("- **HTTP Load Balancing**: %s\n",
		boolToEnabledString(!cluster.AddonsConfig.HttpLoadBalancing.Disabled))
	result += fmt.Sprintf("- **Horizontal Pod Autoscaling**: %s\n",
		boolToEnabledString(!cluster.AddonsConfig.HorizontalPodAutoscaling.Disabled))
	result += fmt.Sprintf("- **Kubernetes Dashboard**: %s\n",
		boolToEnabledString(!cluster.AddonsConfig.KubernetesDashboard.Disabled))
	result += fmt.Sprintf("- **Network Policy**: %s\n",
		boolToEnabledString(!cluster.AddonsConfig.NetworkPolicyConfig.Disabled))

	if len(cluster.Locations) > 0 {
		result += "\n## Node Locations\n\n"
		for _, loc := range cluster.Locations {
			result += fmt.Sprintf("- %s\n", loc)
		}
	}

	if len(cluster.ResourceLabels) > 0 {
		result += "\n## Resource Labels\n\n"
		for k, v := range cluster.ResourceLabels {
			result += fmt.Sprintf("- **%s**: %s\n", k, v)
		}
	}

	if cluster.MaintenancePolicy.Window.DailyMaintenanceWindow.StartTime != "" {
		result += "\n## Maintenance Window\n\n"
		result += fmt.Sprintf("- **Start Time**: %s\n",
			cluster.MaintenancePolicy.Window.DailyMaintenanceWindow.StartTime)
		result += fmt.Sprintf("- **Duration**: %s\n",
			cluster.MaintenancePolicy.Window.DailyMaintenanceWindow.Duration)
	}

	return mcp.NewToolResultText(result), nil
}

// handleListNodePools handles the list_node_pools tool request
func handleListNodePools(ctx context.Context, request mcp.CallToolRequest, authHandler *auth.OAuthHandler) (*mcp.CallToolResult, error) {
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

	// Get HTTP client with authentication
	client, err := authHandler.GetClient(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error getting authenticated client: %v", err)), nil
	}

	// Construct URL for the Container API
	apiURL := fmt.Sprintf("%s/projects/%s/locations/%s/clusters/%s/nodePools",
		gcpContainerBaseURL, projectID, location, clusterName)

	// Make the API request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating request: %v", err)), nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error making request to Container API: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("Error from Container API: %s", resp.Status)), nil
	}

	// Parse the response
	var response struct {
		NodePools []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Config struct {
				MachineType    string            `json:"machineType"`
				DiskSizeGb     int               `json:"diskSizeGb"`
				OauthScopes    []string          `json:"oauthScopes"`
				ServiceAccount string            `json:"serviceAccount"`
				Preemptible    bool              `json:"preemptible"`
				Labels         map[string]string `json:"labels"`
			} `json:"config"`
			InitialNodeCount  int      `json:"initialNodeCount"`
			Locations         []string `json:"locations"`
			InstanceGroupUrls []string `json:"instanceGroupUrls"`
			Version           string   `json:"version"`
			Autoscaling       struct {
				Enabled      bool `json:"enabled"`
				MinNodeCount int  `json:"minNodeCount"`
				MaxNodeCount int  `json:"maxNodeCount"`
			} `json:"autoscaling"`
			Management struct {
				AutoUpgrade bool `json:"autoUpgrade"`
				AutoRepair  bool `json:"autoRepair"`
			} `json:"management"`
		} `json:"nodePools"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error parsing response: %v", err)), nil
	}

	// Format the results
	var result string
	if len(response.NodePools) == 0 {
		result = fmt.Sprintf("No node pools found in cluster %s in location %s.", clusterName, location)
	} else {
		result = fmt.Sprintf("# Node Pools in Cluster %s\n\n", clusterName)

		for i, pool := range response.NodePools {
			result += fmt.Sprintf("## %d. Node Pool: %s\n\n", i+1, pool.Name)
			result += fmt.Sprintf("- **Status**: %s\n", pool.Status)
			result += fmt.Sprintf("- **Version**: %s\n", pool.Version)
			result += fmt.Sprintf("- **Initial Node Count**: %d\n", pool.InitialNodeCount)

			result += "\n### Machine Configuration\n\n"
			result += fmt.Sprintf("- **Machine Type**: %s\n", pool.Config.MachineType)
			result += fmt.Sprintf("- **Disk Size**: %d GB\n", pool.Config.DiskSizeGb)
			result += fmt.Sprintf("- **Preemptible**: %t\n", pool.Config.Preemptible)
			result += fmt.Sprintf("- **Service Account**: %s\n", pool.Config.ServiceAccount)

			if len(pool.Config.OauthScopes) > 0 {
				result += "- **OAuth Scopes**:\n"
				for _, scope := range pool.Config.OauthScopes {
					result += fmt.Sprintf("  - %s\n", scope)
				}
			}

			if len(pool.Config.Labels) > 0 {
				result += "- **Labels**:\n"
				for k, v := range pool.Config.Labels {
					result += fmt.Sprintf("  - %s: %s\n", k, v)
				}
			}

			result += "\n### Autoscaling\n\n"
			if pool.Autoscaling.Enabled {
				result += fmt.Sprintf("- **Enabled**: Yes\n")
				result += fmt.Sprintf("- **Min Nodes**: %d\n", pool.Autoscaling.MinNodeCount)
				result += fmt.Sprintf("- **Max Nodes**: %d\n", pool.Autoscaling.MaxNodeCount)
			} else {
				result += "- **Enabled**: No\n"
			}

			result += "\n### Management\n\n"
			result += fmt.Sprintf("- **Auto Upgrade**: %t\n", pool.Management.AutoUpgrade)
			result += fmt.Sprintf("- **Auto Repair**: %t\n", pool.Management.AutoRepair)

			if len(pool.Locations) > 0 {
				result += "\n### Locations\n\n"
				for _, loc := range pool.Locations {
					result += fmt.Sprintf("- %s\n", loc)
				}
			}

			result += "\n"
		}
	}

	return mcp.NewToolResultText(result), nil
}

// boolToEnabledString converts a boolean to "Enabled" or "Disabled"
func boolToEnabledString(b bool) string {
	if b {
		return "Enabled"
	}
	return "Disabled"
}
