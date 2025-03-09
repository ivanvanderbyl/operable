package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// Scopes defines the OAuth scopes required for each permission level
var (
	// ReadOnlyScopes are the scopes required for diagnostic operations
	ReadOnlyScopes = []string{
		"https://www.googleapis.com/auth/cloud-platform.read-only",
		"https://www.googleapis.com/auth/logging.read",
		"https://www.googleapis.com/auth/monitoring.read",
		"https://www.googleapis.com/auth/compute.readonly",
		"https://www.googleapis.com/auth/container.readonly",
	}

	// ReadWriteScopes are the scopes required for remediation operations
	ReadWriteScopes = []string{
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/logging.admin",
		"https://www.googleapis.com/auth/monitoring",
		"https://www.googleapis.com/auth/compute",
		"https://www.googleapis.com/auth/container",
	}
)

// OAuthHandler handles the OAuth authentication flow for GCP
type OAuthHandler struct {
	clientID        string
	clientSecret    string
	currentScopes   []string
	credentialsFile string
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler() (*OAuthHandler, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	credentialsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	// We need either OAuth client credentials or a service account credentials file
	if (clientID == "" || clientSecret == "") && credentialsFile == "" {
		return nil, fmt.Errorf("either GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET or GOOGLE_APPLICATION_CREDENTIALS environment variables must be set")
	}

	return &OAuthHandler{
		clientID:        clientID,
		clientSecret:    clientSecret,
		credentialsFile: credentialsFile,
		currentScopes:   ReadOnlyScopes,
	}, nil
}

// GetClient returns an HTTP client with OAuth credentials
func (h *OAuthHandler) GetClient(ctx context.Context) (*http.Client, error) {
	// If credentials file is provided, use it
	if h.credentialsFile != "" {
		creds, err := google.FindDefaultCredentials(ctx, h.currentScopes...)
		if err != nil {
			return nil, fmt.Errorf("error finding default credentials: %w", err)
		}
		return oauth2.NewClient(ctx, creds.TokenSource), nil
	}

	// Otherwise use the OAuth flow with client ID and secret
	config := &oauth2.Config{
		ClientID:     h.clientID,
		ClientSecret: h.clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       h.currentScopes,
		RedirectURL:  "http://localhost:8085/oauth/callback",
	}

	// For simplicity, since this is just a demo, we'll use the client without token persistence
	// In a real application, you would handle the OAuth flow and token storage
	ts := config.TokenSource(ctx, nil)
	return oauth2.NewClient(ctx, ts), nil
}

// UpgradePermissions upgrades the permissions to read-write
func (h *OAuthHandler) UpgradePermissions(ctx context.Context) error {
	// Only upgrade if we're not already at read-write
	if len(h.currentScopes) == len(ReadWriteScopes) {
		return nil
	}

	// Update scopes to read-write
	h.currentScopes = ReadWriteScopes

	return nil
}

// GetClientOptions returns the client options for the GCP SDK
func (h *OAuthHandler) GetClientOptions(ctx context.Context) ([]option.ClientOption, error) {
	// Create authentication options
	var opts []option.ClientOption

	// If credentials file is provided, use it
	if h.credentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(h.credentialsFile))
	} else {
		// Get client and convert to options
		client, err := h.GetClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting OAuth client: %w", err)
		}
		opts = append(opts, option.WithHTTPClient(client))
	}

	return opts, nil
}
