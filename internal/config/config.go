package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"github.com/99designs/keyring"
)

const (
	AppName = "soundcloud-tui"
	KeyringService = "soundcloud-tui"
	TokenKey = "oauth_token"
)

type Config struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	keyring      keyring.Keyring
}

func Load() (*Config, error) {
	cfg := &Config{
		ClientID:     os.Getenv("SOUNDCLOUD_CLIENT_ID"),
		ClientSecret: os.Getenv("SOUNDCLOUD_CLIENT_SECRET"),
	}

	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("SOUNDCLOUD_CLIENT_ID and SOUNDCLOUD_CLIENT_SECRET environment variables must be set")
	}

	// Initialize keyring
	ring, err := keyring.Open(keyring.Config{
		ServiceName: KeyringService,
		
		// Keyring backends to try in order
		AllowedBackends: []keyring.BackendType{
			keyring.SecretServiceBackend,
			keyring.KeychainBackend,
			keyring.WinCredBackend,
			keyring.FileBackend,
		},
		
		// FileBackend config for fallback
		FileDir: getConfigDir(),
		FilePasswordFunc: keyring.FixedStringPrompt("Please enter a password to encrypt your tokens"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}

	cfg.keyring = ring
	return cfg, nil
}

func (c *Config) StoreToken(token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	err = c.keyring.Set(keyring.Item{
		Key:   TokenKey,
		Data:  data,
		Label: "SoundCloud OAuth Token",
	})
	if err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	return nil
}

func (c *Config) LoadToken() (*oauth2.Token, error) {
	item, err := c.keyring.Get(TokenKey)
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return nil, fmt.Errorf("no token found, please authenticate first")
		}
		return nil, fmt.Errorf("failed to retrieve token: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(item.Data, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

func (c *Config) ClearToken() error {
	err := c.keyring.Remove(TokenKey)
	if err != nil && err != keyring.ErrKeyNotFound {
		return fmt.Errorf("failed to clear token: %w", err)
	}
	return nil
}

func getConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return os.TempDir()
	}
	return filepath.Join(homeDir, ".config", AppName)
}