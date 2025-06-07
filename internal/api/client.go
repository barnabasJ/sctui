package api

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"soundcloud-tui/internal/config"
)

type Client struct {
	config     *config.Config
	httpClient *http.Client
	oauthFlow  *OAuthFlow
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		config:    cfg,
		oauthFlow: NewOAuthFlow(cfg.ClientID, cfg.ClientSecret),
	}
}

func (c *Client) AuthenticateBrowser(ctx context.Context) (*oauth2.Token, error) {
	return c.oauthFlow.AuthorizeBrowser(ctx)
}

func (c *Client) GetAuthenticatedClient() (*http.Client, error) {
	token, err := c.config.LoadToken()
	if err != nil {
		return nil, fmt.Errorf("no valid token available: %w", err)
	}

	// Create OAuth2 token source for automatic refresh
	tokenSource := c.oauthFlow.config.TokenSource(context.Background(), token)
	
	// Create HTTP client with OAuth2 transport
	return oauth2.NewClient(context.Background(), tokenSource), nil
}