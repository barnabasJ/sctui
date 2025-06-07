package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"soundcloud-tui/internal/soundcloud"
)

func main() {
	var (
		searchFlag = flag.String("search", "", "Search for tracks")
		trackFlag  = flag.String("track", "", "Get info for a specific track URL")
		helpFlag   = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *helpFlag {
		showHelp()
		return
	}

	// Show disclaimer on first run
	showDisclaimer()

	client, err := soundcloud.NewClient()
	if err != nil {
		log.Fatalf("Failed to create SoundCloud client: %v", err)
	}

	if *searchFlag != "" {
		if err := searchTracks(client, *searchFlag); err != nil {
			log.Fatalf("Search failed: %v", err)
		}
		return
	}

	if *trackFlag != "" {
		if err := getTrackInfo(client, *trackFlag); err != nil {
			log.Fatalf("Failed to get track info: %v", err)
		}
		return
	}

	// TODO: Start TUI application
	fmt.Println("SoundCloud TUI - Interactive mode coming soon!")
	fmt.Println("Try: ./sctui -search \"your query\" or ./sctui -help")
}

func searchTracks(client *soundcloud.Client, query string) error {
	fmt.Printf("🔍 Searching for: %s\n\n", query)
	
	tracks, err := client.Search(query)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(tracks) == 0 {
		fmt.Println("No tracks found.")
		return nil
	}

	fmt.Printf("Found %d tracks:\n\n", len(tracks))
	for i, track := range tracks[:min(10, len(tracks))] {
		duration := formatDuration(track.Duration)
		fmt.Printf("%2d. %s\n", i+1, track.Title)
		fmt.Printf("    by %s\n", track.User.FullName())
		fmt.Printf("    Duration: %s | URL: %s\n\n", duration, track.PermalinkURL)
	}

	return nil
}

func getTrackInfo(client *soundcloud.Client, url string) error {
	fmt.Printf("🎵 Getting track info for: %s\n\n", url)
	
	track, err := client.GetTrackInfo(url)
	if err != nil {
		return fmt.Errorf("failed to get track info: %w", err)
	}

	duration := formatDuration(track.Duration)
	fmt.Printf("Title: %s\n", track.Title)
	fmt.Printf("Artist: %s\n", track.User.FullName())
	fmt.Printf("Duration: %s\n", duration)
	if track.Description != "" {
		fmt.Printf("Description: %s\n", track.Description)
	}
	fmt.Printf("URL: %s\n", track.PermalinkURL)

	return nil
}

func formatDuration(ms int64) string {
	seconds := ms / 1000
	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func showDisclaimer() {
	fmt.Println("⚠️  IMPORTANT DISCLAIMER ⚠️")
	fmt.Println()
	fmt.Println("This application uses SoundCloud's undocumented internal API")
	fmt.Println("through a reverse-engineered Go library. This may violate")
	fmt.Println("SoundCloud's Terms of Service.")
	fmt.Println()
	fmt.Println("By using this software, you acknowledge:")
	fmt.Println("• This is for educational/personal use only")
	fmt.Println("• You assume full responsibility for ToS compliance")
	fmt.Println("• The functionality may break if SoundCloud changes their API")
	fmt.Println("• Consider supporting artists through official channels")
	fmt.Println()
	fmt.Println("Use at your own discretion and risk.")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()
}

func showHelp() {
	fmt.Printf(`SoundCloud TUI - Terminal User Interface for SoundCloud

Usage:
  %s [flags]

Flags:
  -search "query"    Search for tracks by keyword
  -track "url"       Get information for a specific track URL
  -help              Show this help message

Examples:
  %s -search "lofi hip hop"
  %s -track "https://soundcloud.com/artist/track"
  %s                 # Start interactive TUI (coming soon)

Note: This application uses SoundCloud's undocumented API.
See disclaimer above for important legal considerations.
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}