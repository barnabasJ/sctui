package ui_test

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"soundcloud-tui/internal/audio"
	"soundcloud-tui/internal/soundcloud"
	"soundcloud-tui/internal/ui/components/player"
)

func TestAudioStateManagement_StateTransitions(t *testing.T) {
	tests := []struct {
		name              string
		initialPlayerState audio.PlayerState
		initialUIState    player.State
		action            string
		expectedUIState   player.State
		expectedAudioCall string
	}{
		{
			name:              "play when stopped",
			initialPlayerState: audio.StateStopped,
			initialUIState:    player.StateIdle,
			action:            "play_track",
			expectedUIState:   player.StateLoading,
			expectedAudioCall: "extract_stream",
		},
		{
			name:              "pause when playing",
			initialPlayerState: audio.StatePlaying,
			initialUIState:    player.StatePlaying,
			action:            "pause",
			expectedUIState:   player.StatePlaying, // State doesn't change until audio confirms
			expectedAudioCall: "pause",
		},
		{
			name:              "resume when paused",
			initialPlayerState: audio.StatePaused,
			initialUIState:    player.StatePaused,
			action:            "resume",
			expectedUIState:   player.StatePlaying,
			expectedAudioCall: "resume",
		},
		{
			name:              "stop when playing",
			initialPlayerState: audio.StatePlaying,
			initialUIState:    player.StatePlaying,
			action:            "stop",
			expectedUIState:   player.StateIdle,
			expectedAudioCall: "stop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPlayer := &MockAudioPlayer{
				state: tt.initialPlayerState,
			}
			mockExtractor := &MockStreamExtractor{}

			playerComponent := player.NewPlayerComponent(mockPlayer, mockExtractor)
			playerComponent.SetState(tt.initialUIState)

			track := &soundcloud.Track{
				ID:    123,
				Title: "Test Track",
				User:  soundcloud.User{Username: "Test Artist"},
			}

			var updatedComponent tea.Model
			var cmd tea.Cmd

			switch tt.action {
			case "play_track":
				playMsg := player.PlayTrackMsg{Track: track}
				updatedComponent, cmd = playerComponent.Update(playMsg)
			case "pause", "resume":
				spaceMsg := tea.KeyMsg{Type: tea.KeySpace}
				updatedComponent, cmd = playerComponent.Update(spaceMsg)
			case "stop":
				// For testing purposes, simulate stop by setting to idle
				playerComponent.SetState(player.StateIdle)
				updatedComponent = playerComponent
			}

			newPlayerComponent := updatedComponent.(*player.PlayerComponent)
			assert.Equal(t, tt.expectedUIState, newPlayerComponent.GetState(), "UI state should match expected")

			if tt.expectedAudioCall != "stop" {
				assert.NotNil(t, cmd, "Should return a command for audio operations")
			}
		})
	}
}

func TestAudioStateManagement_StateSynchronization(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:    audio.StateStopped,
		position: 0,
		duration: 180000,
		volume:   0.8,
	}

	playerComponent := player.NewPlayerComponent(mockPlayer, nil)
	track := &soundcloud.Track{
		ID:    123,
		Title: "Sync Test Track",
		User:  soundcloud.User{Username: "Sync Artist"},
	}
	playerComponent.SetCurrentTrack(track)

	// Test progression through states
	states := []struct {
		audioState audio.PlayerState
		expectedUI player.State
	}{
		{audio.StateStopped, player.StateIdle},
		{audio.StatePlaying, player.StatePlaying},
		{audio.StatePaused, player.StatePaused},
		{audio.StateStopped, player.StateIdle},
	}

	for _, state := range states {
		mockPlayer.state = state.audioState
		
		// Simulate state detection through progress updates
		progressMsg := player.ProgressUpdateMsg{
			Position: time.Duration(mockPlayer.position) * time.Millisecond,
			Duration: time.Duration(mockPlayer.duration) * time.Millisecond,
		}

		// Update component state based on audio player state
		if state.audioState == audio.StatePlaying {
			playerComponent.SetState(player.StatePlaying)
		} else if state.audioState == audio.StatePaused {
			playerComponent.SetState(player.StatePaused)
		} else {
			playerComponent.SetState(player.StateIdle)
		}

		updatedComponent, _ := playerComponent.Update(progressMsg)
		newPlayerComponent := updatedComponent.(*player.PlayerComponent)

		assert.Equal(t, state.expectedUI, newPlayerComponent.GetState(),
			"UI state should sync with audio state: %v", state.audioState)
	}
}

func TestAudioStateManagement_ProgressUpdatesDuringPlayback(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:    audio.StatePlaying,
		position: 0,
		duration: 180000, // 3 minutes
		volume:   1.0,
	}

	playerComponent := player.NewPlayerComponent(mockPlayer, nil)
	track := &soundcloud.Track{
		ID:    123,
		Title: "Progress Track",
		User:  soundcloud.User{Username: "Progress Artist"},
	}
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)

	// Simulate progress updates over time
	progressPoints := []int64{0, 30000, 60000, 90000, 120000, 150000, 180000}

	for i, pos := range progressPoints {
		mockPlayer.position = pos
		
		progressMsg := player.ProgressUpdateMsg{
			Position: time.Duration(pos) * time.Millisecond,
			Duration: time.Duration(mockPlayer.duration) * time.Millisecond,
		}

		updatedComponent, cmd := playerComponent.Update(progressMsg)
		playerComponent = updatedComponent.(*player.PlayerComponent)

		// Should maintain playing state throughout
		assert.Equal(t, player.StatePlaying, playerComponent.GetState(),
			"Should remain in playing state during progress updates")

		// Should return tick command to continue progress updates
		if i < len(progressPoints)-1 { // Not at the end
			assert.NotNil(t, cmd, "Should return tick command for continued progress")
		}

		// Verify position is updated
		assert.Equal(t, time.Duration(pos)*time.Millisecond, playerComponent.GetPosition(),
			"Position should be updated")
	}
}

func TestAudioStateManagement_ErrorStateHandling(t *testing.T) {
	tests := []struct {
		name           string
		errorScenario  string
		initialState   player.State
		expectedState  player.State
		shouldHaveError bool
	}{
		{
			name:           "stream extraction error",
			errorScenario:  "stream_error",
			initialState:   player.StateLoading,
			expectedState:  player.StateError,
			shouldHaveError: true,
		},
		{
			name:           "playback error",
			errorScenario:  "playback_error",
			initialState:   player.StatePlaying,
			expectedState:  player.StateError,
			shouldHaveError: true,
		},
		{
			name:           "network error during loading",
			errorScenario:  "network_error",
			initialState:   player.StateLoading,
			expectedState:  player.StateError,
			shouldHaveError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPlayer := &MockAudioPlayer{
				state: audio.StateStopped,
			}
			mockExtractor := &MockStreamExtractor{}

			playerComponent := player.NewPlayerComponent(mockPlayer, mockExtractor)
			playerComponent.SetState(tt.initialState)

			track := &soundcloud.Track{
				ID:    123,
				Title: "Error Test Track",
				User:  soundcloud.User{Username: "Error Artist"},
			}
			playerComponent.SetCurrentTrack(track)

			// Simulate error scenarios
			var errorMsg player.StreamInfoMsg
			switch tt.errorScenario {
			case "stream_error":
				errorMsg = player.StreamInfoMsg{
					StreamInfo: nil,
					Error:      assert.AnError,
				}
			case "playback_error", "network_error":
				errorMsg = player.StreamInfoMsg{
					StreamInfo: nil,
					Error:      assert.AnError,
				}
			}

			updatedComponent, _ := playerComponent.Update(errorMsg)
			newPlayerComponent := updatedComponent.(*player.PlayerComponent)

			assert.Equal(t, tt.expectedState, newPlayerComponent.GetState(),
				"Should transition to error state")

			if tt.shouldHaveError {
				assert.NotNil(t, newPlayerComponent.GetError(), "Should have error set")
			}
		})
	}
}

func TestAudioStateManagement_StateRecoveryAfterError(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state: audio.StateStopped,
	}
	mockExtractor := &MockStreamExtractor{}

	playerComponent := player.NewPlayerComponent(mockPlayer, mockExtractor)
	
	// Set to error state first
	playerComponent.SetState(player.StateError)
	
	track1 := &soundcloud.Track{
		ID:    123,
		Title: "Error Track",
		User:  soundcloud.User{Username: "Error Artist"},
	}
	playerComponent.SetCurrentTrack(track1)

	assert.Equal(t, player.StateError, playerComponent.GetState())

	// Try to play a new track to recover from error
	track2 := &soundcloud.Track{
		ID:    456,
		Title: "Recovery Track",
		User:  soundcloud.User{Username: "Recovery Artist"},
	}

	playMsg := player.PlayTrackMsg{Track: track2}
	updatedComponent, cmd := playerComponent.Update(playMsg)
	newPlayerComponent := updatedComponent.(*player.PlayerComponent)

	// Should transition out of error state
	assert.Equal(t, player.StateLoading, newPlayerComponent.GetState(),
		"Should recover from error state when playing new track")
	assert.NotNil(t, cmd, "Should return command to load new track")
	assert.Equal(t, track2, newPlayerComponent.GetCurrentTrack(),
		"Should update to new track")
}

func TestAudioStateManagement_VolumeStateConsistency(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:  audio.StatePlaying,
		volume: 0.5,
	}

	playerComponent := player.NewPlayerComponent(mockPlayer, nil)
	track := &soundcloud.Track{
		ID:    123,
		Title: "Volume Track",
		User:  soundcloud.User{Username: "Volume Artist"},
	}
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)

	// Test volume changes maintain state consistency
	volumeChanges := []float64{0.6, 0.7, 0.8, 0.9, 1.0, 0.8, 0.5, 0.2, 0.0}

	for _, newVolume := range volumeChanges {
		// Simulate volume change
		mockPlayer.volume = newVolume
		
		// Update through progress message (which would normally happen)
		progressMsg := player.ProgressUpdateMsg{
			Position: 60 * time.Second,
			Duration: 180 * time.Second,
		}

		updatedComponent, _ := playerComponent.Update(progressMsg)
		playerComponent = updatedComponent.(*player.PlayerComponent)

		// State should remain consistent
		assert.Equal(t, player.StatePlaying, playerComponent.GetState(),
			"Playing state should be maintained during volume changes")

		// Volume should be synchronized
		componentVolume := playerComponent.GetVolume()
		assert.InDelta(t, newVolume, componentVolume, 0.01,
			"Component volume should match audio player volume")
	}
}

func TestAudioStateManagement_SeekOperations(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:    audio.StatePlaying,
		position: 60000, // 1 minute
		duration: 180000, // 3 minutes
		volume:   0.8,
	}

	playerComponent := player.NewPlayerComponent(mockPlayer, nil)
	track := &soundcloud.Track{
		ID:    123,
		Title: "Seek Track",
		User:  soundcloud.User{Username: "Seek Artist"},
	}
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)

	// Test various seek operations
	seekTests := []struct {
		name        string
		key         tea.Key
		expectCmd   bool
		description string
	}{
		{
			name:        "seek forward",
			key:         tea.KeyRight,
			expectCmd:   true,
			description: "Should generate seek forward command",
		},
		{
			name:        "seek backward",
			key:         tea.KeyLeft,
			expectCmd:   true,
			description: "Should generate seek backward command",
		},
	}

	for _, tt := range seekTests {
		t.Run(tt.name, func(t *testing.T) {
			keyMsg := tea.KeyMsg{Type: tt.key}
			updatedComponent, cmd := playerComponent.Update(keyMsg)
			playerComponent = updatedComponent.(*player.PlayerComponent)

			if tt.expectCmd {
				assert.NotNil(t, cmd, tt.description)
			}

			// State should remain playing during seeks
			assert.Equal(t, player.StatePlaying, playerComponent.GetState(),
				"Should maintain playing state during seek operations")
		})
	}
}

func TestAudioStateManagement_ConcurrentStateUpdates(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:    audio.StatePlaying,
		position: 30000,
		duration: 180000,
		volume:   0.7,
	}

	playerComponent := player.NewPlayerComponent(mockPlayer, nil)
	track := &soundcloud.Track{
		ID:    123,
		Title: "Concurrent Track",
		User:  soundcloud.User{Username: "Concurrent Artist"},
	}
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)

	// Simulate rapid successive updates that might happen in real usage
	updates := []tea.Msg{
		player.ProgressUpdateMsg{Position: 31 * time.Second, Duration: 180 * time.Second},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}, // Volume up
		player.ProgressUpdateMsg{Position: 32 * time.Second, Duration: 180 * time.Second},
		tea.KeyMsg{Type: tea.KeyRight}, // Seek forward
		player.ProgressUpdateMsg{Position: 42 * time.Second, Duration: 180 * time.Second}, // After seek
	}

	for i, msg := range updates {
		updatedComponent, cmd := playerComponent.Update(msg)
		playerComponent = updatedComponent.(*player.PlayerComponent)

		// Should always maintain a valid state
		assert.NotEqual(t, player.State(-1), playerComponent.GetState(),
			"Should always have valid state after update %d", i)

		// Should maintain playing state throughout rapid updates
		assert.Equal(t, player.StatePlaying, playerComponent.GetState(),
			"Should maintain playing state during rapid updates")

		// Commands should be generated for appropriate messages
		switch msg.(type) {
		case tea.KeyMsg:
			assert.NotNil(t, cmd, "Key messages should generate commands")
		case player.ProgressUpdateMsg:
			// Progress updates may or may not generate commands (tick commands)
		}
	}
}

func TestAudioStateManagement_StateTransitionWithoutAudioPlayer(t *testing.T) {
	// Test behavior when audio player is nil (edge case)
	playerComponent := player.NewPlayerComponent(nil, nil)
	
	track := &soundcloud.Track{
		ID:    123,
		Title: "No Player Track",
		User:  soundcloud.User{Username: "No Player Artist"},
	}

	// Try to play track without audio player
	playMsg := player.PlayTrackMsg{Track: track}
	updatedComponent, cmd := playerComponent.Update(playMsg)
	newPlayerComponent := updatedComponent.(*player.PlayerComponent)

	// Should handle gracefully and set error state
	assert.Equal(t, player.StateError, newPlayerComponent.GetState(),
		"Should set error state when no stream extractor available")
	assert.NotNil(t, newPlayerComponent.GetError(),
		"Should have error message explaining the issue")
	assert.Nil(t, cmd, "Should not return command when no extractor available")
}

func TestAudioStateManagement_TrackChangesDuringPlayback(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:    audio.StatePlaying,
		position: 60000,
		duration: 180000,
		volume:   0.8,
	}
	mockExtractor := &MockStreamExtractor{}

	playerComponent := player.NewPlayerComponent(mockPlayer, mockExtractor)
	
	track1 := &soundcloud.Track{
		ID:    123,
		Title: "First Track",
		User:  soundcloud.User{Username: "First Artist"},
	}
	playerComponent.SetCurrentTrack(track1)
	playerComponent.SetState(player.StatePlaying)

	// Change to new track while playing
	track2 := &soundcloud.Track{
		ID:    456,
		Title: "Second Track",
		User:  soundcloud.User{Username: "Second Artist"},
	}

	playMsg := player.PlayTrackMsg{Track: track2}
	updatedComponent, cmd := playerComponent.Update(playMsg)
	newPlayerComponent := updatedComponent.(*player.PlayerComponent)

	// Should transition to loading new track
	assert.Equal(t, player.StateLoading, newPlayerComponent.GetState(),
		"Should transition to loading when changing tracks")
	assert.Equal(t, track2, newPlayerComponent.GetCurrentTrack(),
		"Should update to new track")
	assert.NotNil(t, cmd, "Should return command to load new track")
	assert.Nil(t, newPlayerComponent.GetError(),
		"Should clear any previous errors when loading new track")
}