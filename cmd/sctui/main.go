package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"soundcloud-tui/internal/api"
	"soundcloud-tui/internal/config"
)

func main() {
	var (
		authFlag = flag.Bool("auth", false, "Authenticate with SoundCloud")
		helpFlag = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *helpFlag {
		showHelp()
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *authFlag {
		if err := authenticate(cfg); err != nil {
			log.Fatalf("Authentication failed: %v", err)
		}
		fmt.Println("Authentication successful!")
		return
	}

	// TODO: Start TUI application
	fmt.Println("SoundCloud TUI - Coming soon!")
}

func authenticate(cfg *config.Config) error {
	client := api.NewClient(cfg)
	ctx := context.Background()
	
	token, err := client.AuthenticateBrowser(ctx)
	if err != nil {
		return fmt.Errorf("OAuth flow failed: %w", err)
	}

	if err := cfg.StoreToken(token); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	return nil
}

func showHelp() {
	fmt.Printf(`SoundCloud TUI - Terminal User Interface for SoundCloud

Usage:
  %s [flags]

Flags:
  -auth    Authenticate with SoundCloud OAuth
  -help    Show this help message

Before first use, you need to authenticate:
  %s -auth

Note: This application requires you to provide your own SoundCloud
API credentials due to SoundCloud's closed API registration.
Set SOUNDCLOUD_CLIENT_ID and SOUNDCLOUD_CLIENT_SECRET environment variables.
`, os.Args[0], os.Args[0])
}