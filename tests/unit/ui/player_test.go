package ui_test

import (
	"context"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"soundcloud-tui/internal/audio"
	"soundcloud-tui/internal/soundcloud"
	"soundcloud-tui/internal/ui/components/player"
)

func TestPlayerComponent_NewPlayerComponent(t *testing.T) {
	component := player.NewPlayerComponent(nil, nil)
	
	require.NotNil(t, component)
	assert.Nil(t, component.GetCurrentTrack())
	assert.Equal(t, player.StateIdle, component.GetState())
	assert.Equal(t, float64(1.0), component.GetVolume())
	assert.Equal(t, time.Duration(0), component.GetPosition())
}

func TestPlayerComponent_PlayTrack(t *testing.T) {
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
	
	component := player.NewPlayerComponent(mockPlayer, mockExtractor)
	
	track := &soundcloud.Track{
		ID:       123456789,
		Title:    "Test Track",
		User:     soundcloud.User{Username: "Test Artist"},
		Duration: 240000,
	}
	
	// Send play track message
	playMsg := player.PlayTrackMsg{Track: track}
	updatedComponent, cmd := component.Update(playMsg)
	component = updatedComponent.(*player.PlayerComponent)
	
	assert.Equal(t, player.StateLoading, component.GetState())
	assert.Equal(t, track, component.GetCurrentTrack())
	assert.NotNil(t, cmd) // Should return stream extraction command
}

func TestPlayerComponent_PlaybackControls(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state: audio.StatePlaying,
	}
	component := player.NewPlayerComponent(mockPlayer, nil)
	
	// Set up playing state
	track := &soundcloud.Track{ID: 123, Title: "Test Track", User: soundcloud.User{Username: "Test Artist"}}
	component.SetCurrentTrack(track)
	component.SetState(player.StatePlaying)
	
	tests := []struct {
		name        string
		key         tea.Key
		expectedCmd bool
		description string
	}{
		{
			name:        "spacebar toggles play/pause",
			key:         tea.Key{Type: tea.KeySpace},
			expectedCmd: true,
			description: "should pause when playing",
		},
		{
			name:        "left arrow seeks backward",
			key:         tea.Key{Type: tea.KeyLeft},
			expectedCmd: true,
			description: "should seek backward",
		},
		{
			name:        "right arrow seeks forward",
			key:         tea.Key{Type: tea.KeyRight},
			expectedCmd: true,
			description: "should seek forward",
		},
		{
			name:        "plus increases volume",
			key:         tea.Key{Type: tea.KeyRunes, Runes: []rune{'+'}},
			expectedCmd: true,
			description: "should increase volume",
		},
		{
			name:        "minus decreases volume",
			key:         tea.Key{Type: tea.KeyRunes, Runes: []rune{'-'}},
			expectedCmd: true,
			description: "should decrease volume",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyMsg := tea.KeyMsg(tt.key)
			updatedComponent, cmd := component.Update(keyMsg)
			
			if tt.expectedCmd {
				assert.NotNil(t, cmd, tt.description)
			} else {
				assert.Nil(t, cmd, tt.description)
			}
			
			component = updatedComponent.(*player.PlayerComponent)
		})
	}
}

func TestPlayerComponent_VolumeControl(t *testing.T) {
	mockPlayer := &MockAudioPlayer{}
	component := player.NewPlayerComponent(mockPlayer, nil)
	
	initialVolume := component.GetVolume()
	assert.Equal(t, float64(1.0), initialVolume)
	
	// Test volume increase
	plusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}
	updatedComponent, cmd := component.Update(plusMsg)
	component = updatedComponent.(*player.PlayerComponent)
	
	assert.NotNil(t, cmd)
	// Note: Volume change would be handled by the command result
	
	// Test volume decrease
	minusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}}
	updatedComponent, cmd = component.Update(minusMsg)
	component = updatedComponent.(*player.PlayerComponent)
	
	assert.NotNil(t, cmd)
}

func TestPlayerComponent_SeekControls(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:    audio.StatePlaying,
		duration: 240 * time.Second,
		position: 60 * time.Second,
	}
	component := player.NewPlayerComponent(mockPlayer, nil)
	
	// Test seek backward
	leftMsg := tea.KeyMsg{Type: tea.KeyLeft}
	updatedComponent, cmd := component.Update(leftMsg)
	component = updatedComponent.(*player.PlayerComponent)
	
	assert.NotNil(t, cmd) // Should return seek command
	
	// Test seek forward
	rightMsg := tea.KeyMsg{Type: tea.KeyRight}
	updatedComponent, cmd = component.Update(rightMsg)
	component = updatedComponent.(*player.PlayerComponent)
	
	assert.NotNil(t, cmd) // Should return seek command
}

func TestPlayerComponent_StateTransitions(t *testing.T) {
	mockPlayer := &MockAudioPlayer{}
	component := player.NewPlayerComponent(mockPlayer, nil)
	
	// Initial state
	assert.Equal(t, player.StateIdle, component.GetState())
	
	// Loading state
	component.SetState(player.StateLoading)
	assert.Equal(t, player.StateLoading, component.GetState())
	
	// Playing state
	component.SetState(player.StatePlaying)
	assert.Equal(t, player.StatePlaying, component.GetState())
	
	// Paused state
	component.SetState(player.StatePaused)
	assert.Equal(t, player.StatePaused, component.GetState())
	
	// Error state
	component.SetState(player.StateError)
	assert.Equal(t, player.StateError, component.GetState())
}

func TestPlayerComponent_StreamInfoHandling(t *testing.T) {
	mockPlayer := &MockAudioPlayer{}
	component := player.NewPlayerComponent(mockPlayer, nil)
	
	streamInfo := &audio.StreamInfo{
		URL:      "https://example.com/stream.mp3",
		Format:   "mp3",
		Quality:  "sq",
		Duration: 240000,
	}
	
	// Send stream info message
	streamMsg := player.StreamInfoMsg{StreamInfo: streamInfo, Error: nil}
	updatedComponent, cmd := component.Update(streamMsg)
	component = updatedComponent.(*player.PlayerComponent)
	
	assert.Equal(t, player.StatePlaying, component.GetState())
	assert.NotNil(t, cmd) // Should return play command
}

func TestPlayerComponent_ErrorHandling(t *testing.T) {
	mockPlayer := &MockAudioPlayer{}
	component := player.NewPlayerComponent(mockPlayer, nil)
	
	// Send error message
	errorMsg := player.StreamInfoMsg{
		StreamInfo: nil,
		Error:      assert.AnError,
	}
	
	updatedComponent, _ := component.Update(errorMsg)
	component = updatedComponent.(*player.PlayerComponent)
	
	assert.Equal(t, player.StateError, component.GetState())
	assert.NotNil(t, component.GetError())
}

func TestPlayerComponent_ProgressUpdates(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:    audio.StatePlaying,
		duration: 240 * time.Second,
		position: 30 * time.Second,
	}
	component := player.NewPlayerComponent(mockPlayer, nil)
	
	// Send progress update message
	progressMsg := player.ProgressUpdateMsg{
		Position: 45 * time.Second,
		Duration: 240 * time.Second,
	}
	
	updatedComponent, _ := component.Update(progressMsg)
	component = updatedComponent.(*player.PlayerComponent)
	
	// The position should be updated based on the message
	assert.Equal(t, 45*time.Second, component.GetPosition())
	assert.Equal(t, 240*time.Second, component.GetDuration())
}

func TestPlayerComponent_ViewRendering(t *testing.T) {
	mockPlayer := &MockAudioPlayer{}
	component := player.NewPlayerComponent(mockPlayer, nil)
	
	// Test idle view
	view := component.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "No track") // Should show no track message
	
	// Set current track
	track := &soundcloud.Track{
		ID:       123,
		Title:    "Test Track",
		User:     soundcloud.User{Username: "Test Artist"},
		Duration: 240000,
	}
	component.SetCurrentTrack(track)
	component.SetState(player.StatePlaying)
	// Set the mock player to playing state
	mockPlayer.state = audio.StatePlaying
	
	view = component.View()
	assert.Contains(t, view, "Test Track")
	assert.Contains(t, view, "Test Artist")
	assert.Contains(t, view, "Playing") // Should show playing state
}

func TestPlayerComponent_BubbleTeaIntegration(t *testing.T) {
	mockPlayer := &MockAudioPlayer{}
	component := player.NewPlayerComponent(mockPlayer, nil)
	
	// Test that component implements tea.Model interface
	var _ tea.Model = component
	
	// Test Init returns expected command
	cmd := component.Init()
	assert.NotNil(t, cmd) // Should return progress ticker command
	
	// Test Update handles various message types
	_, cmd = component.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	assert.Nil(t, cmd) // Window size should not generate commands
	
	// Test View returns non-empty string
	view := component.View()
	assert.NotEmpty(t, view)
}

// Mock implementations for testing
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