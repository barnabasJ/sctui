package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"github.com/pkg/browser"
)

const (
	AuthURL     = "https://soundcloud.com/connect"
	TokenURL    = "https://api.soundcloud.com/oauth2/token"
	RedirectURL = "http://127.0.0.1:8888/callback"
)

type OAuthFlow struct {
	config   *oauth2.Config
	verifier string
	state    string
}

func NewOAuthFlow(clientID, clientSecret string) *OAuthFlow {
	return &OAuthFlow{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  AuthURL,
				TokenURL: TokenURL,
			},
			RedirectURL: RedirectURL,
			Scopes:      []string{"non-expiring"},
		},
	}
}

func (f *OAuthFlow) AuthorizeBrowser(ctx context.Context) (*oauth2.Token, error) {
	// Generate PKCE verifier and challenge
	if err := f.generatePKCE(); err != nil {
		return nil, fmt.Errorf("failed to generate PKCE: %w", err)
	}

	// Generate state for CSRF protection
	f.state = generateRandomString(32)

	// Build authorization URL
	authURL := f.config.AuthCodeURL(f.state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(f.verifier))

	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("If the browser doesn't open, visit: %s\n", authURL)

	// Open browser
	if err := browser.OpenURL(authURL); err != nil {
		fmt.Printf("Failed to open browser automatically: %v\n", err)
		fmt.Printf("Please manually open the URL above\n")
	}

	// Start callback server and wait for authorization code
	code, err := f.waitForCallback(ctx)
	if err != nil {
		return nil, fmt.Errorf("callback failed: %w", err)
	}

	// Exchange authorization code for token
	token, err := f.config.Exchange(ctx, code, oauth2.VerifierOption(f.verifier))
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	return token, nil
}

func (f *OAuthFlow) generatePKCE() error {
	// Generate code verifier (43-128 characters)
	verifierBytes := make([]byte, 96)
	if _, err := rand.Read(verifierBytes); err != nil {
		return err
	}
	f.verifier = base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(verifierBytes)
	return nil
}

func (f *OAuthFlow) waitForCallback(ctx context.Context) (string, error) {
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	server := &http.Server{
		Addr: ":8888",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}

			// Extract authorization code and state
			query := r.URL.Query()
			code := query.Get("code")
			state := query.Get("state")
			errorParam := query.Get("error")

			if errorParam != "" {
				errChan <- fmt.Errorf("OAuth error: %s", errorParam)
				return
			}

			if state != f.state {
				errChan <- fmt.Errorf("invalid state parameter")
				return
			}

			if code == "" {
				errChan <- fmt.Errorf("no authorization code received")
				return
			}

			// Send success response
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>SoundCloud TUI - Authentication Success</title>
    <style>
        body { font-family: Arial, sans-serif; text-align: center; margin-top: 50px; }
        .success { color: green; font-size: 24px; }
        .message { color: #666; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="success">✓ Authentication Successful!</div>
    <div class="message">You can now close this window and return to your terminal.</div>
</body>
</html>`)

			codeChan <- code
		}),
	}

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	// Wait for callback or timeout
	select {
	case code := <-codeChan:
		server.Shutdown(ctx)
		return code, nil
	case err := <-errChan:
		server.Shutdown(ctx)
		return "", err
	case <-time.After(5 * time.Minute):
		server.Shutdown(ctx)
		return "", fmt.Errorf("authentication timeout after 5 minutes")
	case <-ctx.Done():
		server.Shutdown(ctx)
		return "", ctx.Err()
	}
}

func generateRandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)[:length]
}