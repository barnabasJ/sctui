package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"soundcloud-tui/internal/audio"
	"soundcloud-tui/internal/soundcloud"
	"soundcloud-tui/internal/ui/app"
	"soundcloud-tui/internal/ui/components/player"
)

func main() {
	var (
		searchFlag = flag.String("search", "", "Search for tracks")
		trackFlag  = flag.String("track", "", "Get info for a specific track URL")
		playFlag   = flag.String("play", "", "Play a specific track URL directly")
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

	if *playFlag != "" {
		if err := playTrackFromURL(client, *playFlag); err != nil {
			log.Fatalf("Failed to play track: %v", err)
		}
		return
	}

	// Start TUI application
	application := app.NewApp()
	program := tea.NewProgram(application, tea.WithAltScreen())
	
	if _, err := program.Run(); err != nil {
		log.Fatalf("Failed to start TUI: %v", err)
	}
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

// validateSoundCloudURL validates and normalizes a SoundCloud URL
func validateSoundCloudURL(url string) error {
	// Remove whitespace and common prefixes
	url = strings.TrimSpace(url)
	
	// Ensure it's a valid SoundCloud URL
	soundcloudPattern := regexp.MustCompile(`^https?://(www\.|m\.)?soundcloud\.com/[^/]+/[^/?]+`)
	if !soundcloudPattern.MatchString(url) {
		return fmt.Errorf("invalid SoundCloud URL format. Expected: https://soundcloud.com/artist/track")
	}
	
	return nil
}

// playTrackFromURL plays a track directly from a SoundCloud URL
func playTrackFromURL(client *soundcloud.Client, url string) error {
	fmt.Printf("🎵 Loading track from: %s\n\n", url)
	
	// Validate URL format
	if err := validateSoundCloudURL(url); err != nil {
		return err
	}
	
	// Get track information
	track, err := client.GetTrackInfo(url)
	if err != nil {
		return fmt.Errorf("failed to get track info: %w", err)
	}
	
	fmt.Printf("Now playing: %s by %s\n", track.Title, track.User.FullName())
	fmt.Printf("Duration: %s\n\n", formatDuration(track.Duration))
	
	// Create audio components
	audioPlayer := audio.NewBeepPlayer()
	defer audioPlayer.Close()
	
	streamExtractor := audio.NewRealSoundCloudStreamExtractor(client)
	
	// Create player-only TUI
	playerComponent := player.NewPlayerComponent(audioPlayer, streamExtractor)
	
	// Create simple app that only shows the player
	playApp := &DirectPlayApp{
		player: playerComponent,
		track:  track,
	}
	
	// Start the player TUI
	program := tea.NewProgram(playApp, tea.WithAltScreen())
	_, err = program.Run()
	
	return err
}

// DirectPlayApp is a minimal TUI app for direct URL playback
type DirectPlayApp struct {
	player *player.PlayerComponent
	track  *soundcloud.Track
	width  int
	height int
}

func (a *DirectPlayApp) Init() tea.Cmd {
	// Start playing the track immediately
	return tea.Batch(
		a.player.Init(),
		func() tea.Msg {
			return player.PlayTrackMsg{Track: a.track}
		},
	)
}

func (a *DirectPlayApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle quit keys
		if msg.Type == tea.KeyCtrlC || msg.String() == "q" {
			return a, tea.Quit
		}
		
		// Pass all other keys to player
		updatedPlayer, cmd := a.player.Update(msg)
		a.player = updatedPlayer.(*player.PlayerComponent)
		return a, cmd
		
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.player.SetSize(msg.Width, msg.Height-2) // Reserve space for title
		return a, nil
		
	default:
		// Pass all other messages to player
		updatedPlayer, cmd := a.player.Update(msg)
		a.player = updatedPlayer.(*player.PlayerComponent)
		return a, cmd
	}
}

func (a *DirectPlayApp) View() string {
	// Simple header
	header := fmt.Sprintf("SoundCloud TUI - Direct Play Mode (Press 'q' or Ctrl+C to quit)")
	
	// Player view
	playerView := a.player.View()
	
	return fmt.Sprintf("%s\n%s", header, playerView)
}

func showHelp() {
	fmt.Printf(`SoundCloud TUI - Terminal User Interface for SoundCloud

Usage:
  %s [flags]

Flags:
  -search "query"    Search for tracks by keyword
  -track "url"       Get information for a specific track URL
  -play "url"        Play a specific track URL directly
  -help              Show this help message

Examples:
  %s -search "lofi hip hop"
  %s -track "https://soundcloud.com/artist/track"
  %s -play "https://soundcloud.com/artist/track"
  %s                 # Start interactive TUI

Note: This application uses SoundCloud's undocumented API.
See disclaimer above for important legal considerations.
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}