# Operable - GCP/Kubernetes Incident Response MCP Server

Operable is an MCP (Model Context Protocol) server that helps non-operations developers and time-constrained operations personnel quickly respond to incidents in Google Cloud Platform (GCP) and Kubernetes environments. It provides diagnostic capabilities and remediation guidance through any AI chat interface that supports MCP, such as Cursor.

## Features

### Phase 1 (Current)
- **OAuth Authentication**: Secure authentication with GCP using OAuth
- **GCP Issues Tools**: Query and analyze active issues from GCP Error Reporting
- **Logging Tools**: Query logs from GCP Cloud Logging and Kubernetes pods
- **Kubernetes Tools**: Inspect GKE clusters, node pools, and resources
- **Monitoring Tools**: Query metrics and alerts from GCP Cloud Monitoring
- **Documentation Tools**: Search GCP and Kubernetes documentation for help

### Phase 2 (Planned)
- **Remediation Actions**: Execute approved remediation steps
- **Verification**: Confirm successful resolution
- **Rollback**: Safely revert changes if needed

## Prerequisites

- Go 1.24 or later
- Google Cloud Platform account with appropriate permissions
- OAuth client credentials for GCP

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/ivanvanderbyl/operable.git
   cd operable
   ```

2. Install dependencies:
   ```
   go mod download
   ```

3. Set up OAuth credentials:
   - Create OAuth credentials in the Google Cloud Console
   - Set the environment variables:
     ```
     export GOOGLE_CLIENT_ID=your_client_id
     export GOOGLE_CLIENT_SECRET=your_client_secret
     ```

## Usage

1. Build and run the server:
   ```
   go run cmd/main.go
   ```

2. Connect to the server using an MCP-compatible client like Cursor.

3. Authenticate with your Google Cloud account when prompted.

4. Use the available tools to diagnose and respond to incidents.

## Available Tools

### GCP Issues Tools

- `list_active_issues`: Lists active issues from GCP Error Reporting
- `get_issue_details`: Gets detailed information about a specific error group

### Logging Tools
- `query_logs`: Queries logs from GCP Cloud Logging
- `get_pod_logs`: Gets logs for a specific Kubernetes pod

### Kubernetes Tools

- `list_clusters`: Lists GKE clusters in a project
- `get_cluster_info`: Gets detailed information about a GKE cluster
- `list_node_pools`: Lists node pools in a GKE cluster

### Monitoring Tools

- `query_metrics`: Queries metrics from GCP Cloud Monitoring
- `list_alerts`: Lists active alerts from GCP Cloud Monitoring

### Documentation Tools
- `search_gcp_docs`: Searches Google Cloud documentation
- `search_k8s_docs`: Searches Kubernetes documentation
- `get_error_docs`: Gets documentation for a specific error code or message

## Architecture

The system is implemented as an MCP server using the mark3labs/mcp-go library, consisting of:

- **Core MCP Server**: Implements the MCP protocol for AI chat interface integration
- **Authentication Layer**: Handles OAuth authentication with GCP
- **Resource Access Layer**: Provides access to GCP and Kubernetes resources
- **Tools Layer**: Implements diagnostic and remediation capabilities
- **Documentation Integration**: Accesses relevant documentation and best practices

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
