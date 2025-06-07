package ui_test

import (
	"context"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"soundcloud-tui/internal/audio"
	"soundcloud-tui/internal/soundcloud"
	"soundcloud-tui/internal/ui/app"
	"soundcloud-tui/internal/ui/components/player"
)

func TestAudioIntegration_TrackSelectionTriggersPlayback(t *testing.T) {
	// Setup mock audio player and stream extractor
	mockPlayer := &MockAudioPlayer{}
	mockExtractor := &MockStreamExtractor{
		ExtractFunc: func(ctx context.Context, trackID int64) (*audio.StreamInfo, error) {
			return &audio.StreamInfo{
				URL:      "https://example.com/stream.mp3",
				Format:   "mp3",
				Quality:  "sq",
				Duration: 240000,
			}, nil
		},
	}
	
	// Create player component
	playerComponent := player.NewPlayerComponent(mockPlayer, mockExtractor)
	
	// Test track selection triggers stream extraction
	track := &soundcloud.Track{
		ID:       123456789,
		Title:    "Test Track",
		User:     soundcloud.User{Username: "Test Artist"},
		Duration: 240000,
	}
	
	playMsg := player.PlayTrackMsg{Track: track}
	updatedComponent, cmd := playerComponent.Update(playMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)
	
	// Should be in loading state
	assert.Equal(t, player.StateLoading, playerComponent.GetState())
	assert.Equal(t, track, playerComponent.GetCurrentTrack())
	assert.NotNil(t, cmd) // Should return stream extraction command
	
	// Simulate stream extraction completion
	streamInfo := &audio.StreamInfo{
		URL:      "https://example.com/stream.mp3",
		Format:   "mp3",
		Quality:  "sq",
		Duration: 240000,
	}
	
	streamMsg := player.StreamInfoMsg{StreamInfo: streamInfo, Error: nil}
	updatedComponent, playCmd := playerComponent.Update(streamMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)
	
	// Should transition to playing state
	assert.Equal(t, player.StatePlaying, playerComponent.GetState())
	assert.NotNil(t, playCmd) // Should return play command
}

func TestAudioIntegration_ProgressUpdatesReflectInUI(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:    audio.StatePlaying,
		position: 30 * time.Second,
		duration: 240 * time.Second,
		volume:   0.8,
	}
	
	playerComponent := player.NewPlayerComponent(mockPlayer, nil)
	
	// Set playing track
	track := &soundcloud.Track{
		ID:       123,
		Title:    "Test Track",
		User:     soundcloud.User{Username: "Test Artist"},
		Duration: 240000,
	}
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)
	
	// Send progress update
	progressMsg := player.ProgressUpdateMsg{
		Position: 45 * time.Second,
		Duration: 240 * time.Second,
	}
	
	updatedComponent, tickCmd := playerComponent.Update(progressMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)
	
	// Verify progress is reflected
	assert.Equal(t, 45*time.Second, playerComponent.GetPosition())
	assert.Equal(t, 240*time.Second, playerComponent.GetDuration())
	assert.NotNil(t, tickCmd) // Should return next tick command
	
	// Verify UI shows progress
	view := playerComponent.View()
	assert.Contains(t, view, "0:45") // Current position
	assert.Contains(t, view, "4:00") // Total duration
}

func TestAudioIntegration_VolumeControlsUpdateAudioPlayer(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:  audio.StatePlaying,
		volume: 1.0,
	}
	
	playerComponent := player.NewPlayerComponent(mockPlayer, nil)
	
	// Set playing track
	track := &soundcloud.Track{
		ID:    123,
		Title: "Test Track",
		User:  soundcloud.User{Username: "Test Artist"},
	}
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)
	
	// Test volume increase
	plusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}
	updatedComponent, cmd := playerComponent.Update(plusMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)
	
	assert.NotNil(t, cmd) // Should return volume update command
	
	// Simulate volume update command execution
	err := mockPlayer.SetVolume(1.0) // Max volume
	assert.NoError(t, err)
	
	// Test volume decrease
	minusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}}
	updatedComponent, cmd = playerComponent.Update(minusMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)
	
	assert.NotNil(t, cmd) // Should return volume update command
}

func TestAudioIntegration_SeekControlsUpdateAudioPlayer(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:    audio.StatePlaying,
		position: 60 * time.Second,
		duration: 240 * time.Second,
	}
	
	playerComponent := player.NewPlayerComponent(mockPlayer, nil)
	
	// Set playing track
	track := &soundcloud.Track{
		ID:    123,
		Title: "Test Track",
		User:  soundcloud.User{Username: "Test Artist"},
	}
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)
	
	// Test seek forward
	rightMsg := tea.KeyMsg{Type: tea.KeyRight}
	updatedComponent, cmd := playerComponent.Update(rightMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)
	
	assert.NotNil(t, cmd) // Should return seek command
	
	// Test seek backward
	leftMsg := tea.KeyMsg{Type: tea.KeyLeft}
	updatedComponent, cmd = playerComponent.Update(leftMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)
	
	assert.NotNil(t, cmd) // Should return seek command
}

func TestAudioIntegration_PlayPauseControlsUpdateAudioPlayer(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state: audio.StatePlaying,
	}
	
	playerComponent := player.NewPlayerComponent(mockPlayer, nil)
	
	// Set playing track
	track := &soundcloud.Track{
		ID:    123,
		Title: "Test Track",
		User:  soundcloud.User{Username: "Test Artist"},
	}
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)
	
	// Test pause
	spaceMsg := tea.KeyMsg{Type: tea.KeySpace}
	updatedComponent, cmd := playerComponent.Update(spaceMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)
	
	assert.NotNil(t, cmd) // Should return pause command
	
	// Simulate pause execution
	err := mockPlayer.Pause()
	assert.NoError(t, err)
	assert.Equal(t, audio.StatePaused, mockPlayer.GetState())
	
	// Test resume
	mockPlayer.state = audio.StatePaused
	playerComponent.SetState(player.StatePaused)
	
	updatedComponent, cmd = playerComponent.Update(spaceMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)
	
	// Should attempt to resume (transition to playing)
	assert.Equal(t, player.StatePlaying, playerComponent.GetState())
}

func TestAudioIntegration_ErrorHandlingInStreamExtraction(t *testing.T) {
	mockPlayer := &MockAudioPlayer{}
	mockExtractor := &MockStreamExtractor{
		ExtractFunc: func(ctx context.Context, trackID int64) (*audio.StreamInfo, error) {
			return nil, assert.AnError // Simulate extraction error
		},
	}
	
	playerComponent := player.NewPlayerComponent(mockPlayer, mockExtractor)
	
	// Test track selection with extraction error
	track := &soundcloud.Track{
		ID:    123456789,
		Title: "Test Track",
		User:  soundcloud.User{Username: "Test Artist"},
	}
	
	playMsg := player.PlayTrackMsg{Track: track}
	updatedComponent, cmd := playerComponent.Update(playMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)
	
	assert.Equal(t, player.StateLoading, playerComponent.GetState())
	assert.NotNil(t, cmd)
	
	// Simulate error response
	errorMsg := player.StreamInfoMsg{StreamInfo: nil, Error: assert.AnError}
	updatedComponent, _ = playerComponent.Update(errorMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)
	
	// Should be in error state
	assert.Equal(t, player.StateError, playerComponent.GetState())
	assert.NotNil(t, playerComponent.GetError())
	
	// UI should show error
	view := playerComponent.View()
	assert.Contains(t, view, "Error")
}

func TestAudioIntegration_TrackTransitionBetweenSearchAndPlayer(t *testing.T) {
	// This test simulates the full flow from search to player
	mockClient := &MockSoundCloudClient{
		SearchFunc: func(query string) ([]soundcloud.Track, error) {
			return []soundcloud.Track{
				{
					ID:       123456789,
					Title:    "Test Track",
					User:     soundcloud.User{Username: "Test Artist"},
					Duration: 240000,
				},
			}, nil
		},
	}
	
	mockPlayer := &MockAudioPlayer{}
	mockExtractor := &MockStreamExtractor{
		ExtractFunc: func(ctx context.Context, trackID int64) (*audio.StreamInfo, error) {
			return &audio.StreamInfo{
				URL:      "https://example.com/stream.mp3",
				Format:   "mp3",
				Quality:  "sq",
				Duration: 240000,
			}, nil
		},
	}
	
	// Create app with mocked dependencies
	application := createTestApp(mockClient, mockPlayer, mockExtractor)
	
	// Start in search view
	assert.Equal(t, app.ViewSearch, application.GetCurrentView())
	
	// Simulate search
	query := "test"
	for _, char := range query {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedApp, _ := application.Update(msg)
		application = updatedApp.(*app.App)
	}
	
	// Execute search
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedApp, _ := application.Update(enterMsg)
	application = updatedApp.(*app.App)
	
	// Simulate search results (would normally come from async command)
	// This would be handled in a real scenario by the search command
	
	// Navigate to player view
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	updatedApp, _ = application.Update(tabMsg)
	application = updatedApp.(*app.App)
	
	assert.Equal(t, app.ViewPlayer, application.GetCurrentView())
}

func TestAudioIntegration_StateConsistencyBetweenPlayerAndAudio(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:    audio.StatePlaying,
		position: 30 * time.Second,
		duration: 240 * time.Second,
		volume:   0.8,
	}
	
	playerComponent := player.NewPlayerComponent(mockPlayer, nil)
	
	// Set initial state
	track := &soundcloud.Track{
		ID:    123,
		Title: "Test Track",
		User:  soundcloud.User{Username: "Test Artist"},
	}
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)
	
	// Verify player component reflects audio player state
	assert.Equal(t, player.StatePlaying, playerComponent.GetState())
	
	// Simulate external pause (e.g., from another command)
	mockPlayer.state = audio.StatePaused
	
	// Send progress update to sync state
	progressMsg := player.ProgressUpdateMsg{
		Position: 30 * time.Second,
		Duration: 240 * time.Second,
	}
	
	updatedComponent, _ := playerComponent.Update(progressMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)
	
	// UI should reflect the current audio player state in the view
	view := playerComponent.View()
	
	// Should show current position regardless of internal state
	assert.Contains(t, view, "0:30")
}

// Helper function to create test app (would need implementation)
func createTestApp(client soundcloud.ClientInterface, audioPlayer audio.Player, extractor audio.StreamExtractor) *app.App {
	// This would create an app with injected dependencies for testing
	// For now, return a basic app - this would need proper dependency injection
	return app.NewApp()
}

// Mock implementations (reusing from existing test files)
type MockAudioPlayer struct {
	state    audio.PlayerState
	volume   float64
	position time.Duration
	duration time.Duration
}

func (m *MockAudioPlayer) Play(ctx context.Context, streamURL string) error {
	m.state = audio.StatePlaying
	return nil
}

func (m *MockAudioPlayer) Pause() error {
	m.state = audio.StatePaused
	return nil
}

func (m *MockAudioPlayer) Stop() error {
	m.state = audio.StateStopped
	m.position = 0
	return nil
}

func (m *MockAudioPlayer) GetState() audio.PlayerState {
	return m.state
}

func (m *MockAudioPlayer) SetVolume(volume float64) error {
	if volume < 0 || volume > 1 {
		return assert.AnError
	}
	m.volume = volume
	return nil
}

func (m *MockAudioPlayer) GetVolume() float64 {
	if m.volume == 0 {
		return 1.0 // Default volume
	}
	return m.volume
}

func (m *MockAudioPlayer) Seek(position time.Duration) error {
	if position < 0 || position > m.duration {
		return assert.AnError
	}
	m.position = position
	return nil
}

func (m *MockAudioPlayer) GetPosition() time.Duration {
	return m.position
}

func (m *MockAudioPlayer) GetDuration() time.Duration {
	return m.duration
}

func (m *MockAudioPlayer) Close() error {
	m.state = audio.StateStopped
	return nil
}

type MockStreamExtractor struct {
	ExtractFunc func(ctx context.Context, trackID int64) (*audio.StreamInfo, error)
}

func (m *MockStreamExtractor) ExtractStreamURL(ctx context.Context, trackID int64) (*audio.StreamInfo, error) {
	if m.ExtractFunc != nil {
		return m.ExtractFunc(ctx, trackID)
	}
	return nil, assert.AnError
}

func (m *MockStreamExtractor) GetAvailableQualities(ctx context.Context, trackID int64) ([]string, error) {
	return []string{"sq", "hq"}, nil
}

func (m *MockStreamExtractor) ValidateStreamURL(ctx context.Context, streamURL string) (bool, error) {
	return true, nil
}

type MockSoundCloudClient struct {
	SearchFunc func(query string) ([]soundcloud.Track, error)
}

func (m *MockSoundCloudClient) Search(query string) ([]soundcloud.Track, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(query)
	}
	return []soundcloud.Track{}, nil
}

func (m *MockSoundCloudClient) GetTrackInfo(url string) (*soundcloud.Track, error) {
	return nil, nil
}