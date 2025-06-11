package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

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
		testAudioFlag = flag.String("test-audio", "", "Test audio playback without TUI")
		testTuiFlag   = flag.String("test-tui", "", "Test TUI message flow without interactive mode")
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

	if *testAudioFlag != "" {
		if err := testAudioPlayback(client, *testAudioFlag); err != nil {
			log.Fatalf("Failed to test audio: %v", err)
		}
		return
	}

	if *testTuiFlag != "" {
		if err := testTuiPlayback(client, *testTuiFlag); err != nil {
			log.Fatalf("Failed to test TUI: %v", err)
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
	
	// Create audio components with enhanced buffered streaming
	audioPlayer := audio.NewBufferedBeepPlayer()
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
	fmt.Printf("Starting TUI player interface...\n")
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
		
	case player.PlaybackStartedMsg:
		// Playback started successfully - just continue
		return a, nil
		
	case player.PlaybackFailedMsg:
		// Playback failed - show error and quit
		fmt.Printf("\n❌ Playback failed: %v\n", msg.Error)
		return a, tea.Quit
		
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

// testAudioPlayback tests audio playback without TUI interface
func testAudioPlayback(client *soundcloud.Client, url string) error {
	fmt.Printf("🔧 Testing audio playback without TUI for: %s\n\n", url)
	
	// Validate URL format
	if err := validateSoundCloudURL(url); err != nil {
		return err
	}
	
	// Get track information
	track, err := client.GetTrackInfo(url)
	if err != nil {
		return fmt.Errorf("failed to get track info: %w", err)
	}
	
	fmt.Printf("Track: %s by %s\n", track.Title, track.User.FullName())
	fmt.Printf("Duration: %s\n\n", formatDuration(track.Duration))
	
	// Create audio components with enhanced buffered streaming
	audioPlayer := audio.NewBufferedBeepPlayer()
	defer audioPlayer.Close()
	
	streamExtractor := audio.NewRealSoundCloudStreamExtractor(client)
	
	// Extract stream URL
	fmt.Printf("Extracting stream URL...\n")
	streamInfo, err := streamExtractor.ExtractStreamURL(context.Background(), track.ID)
	if err != nil {
		return fmt.Errorf("failed to extract stream URL: %w", err)
	}
	
	fmt.Printf("Stream URL obtained: %s\n", streamInfo.URL[:50]+"...")
	
	// Start playback
	fmt.Printf("Starting audio playback...\n")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	err = audioPlayer.Play(ctx, streamInfo.URL)
	if err != nil {
		return fmt.Errorf("failed to start playback: %w", err)
	}
	
	fmt.Printf("✅ Playback started successfully!\n")
	fmt.Printf("Playing for 10 seconds to test stability...\n\n")
	
	// Monitor playback for 10 seconds
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		
		state := audioPlayer.GetState()
		position := audioPlayer.GetPosition()
		duration := audioPlayer.GetDuration()
		
		fmt.Printf("Second %d: State=%s, Position=%v, Duration=%v\n", 
			i+1, state.String(), position.Truncate(time.Millisecond), duration.Truncate(time.Millisecond))
		
		if state == audio.StateStopped {
			fmt.Printf("❌ Playback stopped unexpectedly at second %d!\n", i+1)
			break
		}
	}
	
	// Stop playback
	fmt.Printf("\nStopping playback...\n")
	audioPlayer.Stop()
	
	return nil
}

// testTuiPlayback simulates TUI message flow to test for differences vs direct audio
func testTuiPlayback(client *soundcloud.Client, url string) error {
	fmt.Printf("🔧 Testing TUI message flow for: %s\n\n", url)
	
	// Validate URL format
	if err := validateSoundCloudURL(url); err != nil {
		return err
	}
	
	// Get track information
	track, err := client.GetTrackInfo(url)
	if err != nil {
		return fmt.Errorf("failed to get track info: %w", err)
	}
	
	fmt.Printf("Track: %s by %s\n", track.Title, track.User.FullName())
	fmt.Printf("Duration: %s\n\n", formatDuration(track.Duration))
	
	// Create audio components (same as TUI)
	audioPlayer := audio.NewBeepPlayer()
	defer audioPlayer.Close()
	
	streamExtractor := audio.NewRealSoundCloudStreamExtractor(client)
	playerComponent := player.NewPlayerComponent(audioPlayer, streamExtractor)
	
	// Simulate TUI initialization
	fmt.Printf("Simulating TUI message flow...\n")
	
	// Step 1: Init player component
	initCmd := playerComponent.Init()
	if initCmd != nil {
		fmt.Printf("Player component initialized\n")
	}
	
	// Step 2: Send PlayTrackMsg (like TUI does)
	fmt.Printf("Sending PlayTrackMsg...\n")
	playMsg := player.PlayTrackMsg{Track: track}
	updatedPlayer, cmd := playerComponent.Update(playMsg)
	playerComponent = updatedPlayer.(*player.PlayerComponent)
	
	// Step 3: Execute the command (stream extraction)
	if cmd != nil {
		fmt.Printf("Executing stream extraction command...\n")
		msg := cmd()
		
		// Step 4: Handle StreamInfoMsg
		if streamMsg, ok := msg.(player.StreamInfoMsg); ok {
			if streamMsg.Error != nil {
				return fmt.Errorf("stream extraction failed: %w", streamMsg.Error)
			}
			
			fmt.Printf("Stream URL extracted, sending StreamInfoMsg...\n")
			updatedPlayer, playCmd := playerComponent.Update(streamMsg)
			playerComponent = updatedPlayer.(*player.PlayerComponent)
			
			// Step 5: Execute play command (it's a batch)
			if playCmd != nil {
				fmt.Printf("Executing play command batch...\n")
				playResult := playCmd()
				fmt.Printf("Play command result type: %T\n", playResult)
				
				// Handle BatchMsg - execute all commands in the batch
				if batchMsg, ok := playResult.(tea.BatchMsg); ok {
					fmt.Printf("Handling batch with %d commands\n", len(batchMsg))
					for i, cmd := range batchMsg {
						fmt.Printf("Executing batch command %d...\n", i+1)
						cmdResult := cmd()
						fmt.Printf("Batch command %d result type: %T\n", i+1, cmdResult)
						
						// Update player with each result
						updatedPlayer, _ := playerComponent.Update(cmdResult)
						playerComponent = updatedPlayer.(*player.PlayerComponent)
					}
				} else {
					// Handle single command result
					updatedPlayer, progressCmd := playerComponent.Update(playResult)
					playerComponent = updatedPlayer.(*player.PlayerComponent)
					
					// Execute progress command if available
					if progressCmd != nil {
						fmt.Printf("Executing initial progress command...\n")
						progressResult := progressCmd()
						updatedPlayer, _ = playerComponent.Update(progressResult)
						playerComponent = updatedPlayer.(*player.PlayerComponent)
					}
				}
			}
		}
	}
	
	fmt.Printf("✅ TUI simulation started!\n")
	fmt.Printf("Waiting 1 second for playback to initialize...\n")
	time.Sleep(1 * time.Second)
	fmt.Printf("Monitoring for 10 seconds to compare with test-audio...\n\n")
	
	// Monitor playback for 10 seconds (same as test-audio)
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		
		state := audioPlayer.GetState()
		position := audioPlayer.GetPosition()
		duration := audioPlayer.GetDuration()
		
		fmt.Printf("Second %d: State=%s, Position=%v, Duration=%v\n", 
			i+1, state.String(), position.Truncate(time.Millisecond), duration.Truncate(time.Millisecond))
		
		if state == audio.StateStopped {
			fmt.Printf("❌ Playback stopped unexpectedly at second %d! (TUI simulation)\n", i+1)
			break
		}
	}
	
	// Stop playback
	fmt.Printf("\nStopping playback...\n")
	audioPlayer.Stop()
	
	return nil
}

func showHelp() {
	fmt.Printf(`SoundCloud TUI - Terminal User Interface for SoundCloud

Usage:
  %s [flags]

Flags:
  -search "query"    Search for tracks by keyword
  -track "url"       Get information for a specific track URL
  -play "url"        Play a specific track URL directly
  -test-audio "url"  Test audio playback without TUI (debug mode)
  -test-tui "url"    Test TUI message flow without interactive mode
  -help              Show this help message

Examples:
  %s -search "lofi hip hop"
  %s -track "https://soundcloud.com/artist/track"
  %s -play "https://soundcloud.com/artist/track"
  %s -test-audio "https://soundcloud.com/artist/track"
  %s -test-tui "https://soundcloud.com/artist/track"
  %s                 # Start interactive TUI

Note: This application uses SoundCloud's undocumented API.
See disclaimer above for important legal considerations.
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}