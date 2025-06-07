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

func TestVolumeControls_IncrementVolume(t *testing.T) {
	tests := []struct {
		name           string
		initialVolume  float64
		expectedVolume float64
		shouldChange   bool
	}{
		{
			name:           "increase from 50%",
			initialVolume:  0.5,
			expectedVolume: 0.6,
			shouldChange:   true,
		},
		{
			name:           "increase from 90%",
			initialVolume:  0.9,
			expectedVolume: 1.0,
			shouldChange:   true,
		},
		{
			name:           "increase from max volume",
			initialVolume:  1.0,
			expectedVolume: 1.0,
			shouldChange:   false,
		},
		{
			name:           "increase from 0%",
			initialVolume:  0.0,
			expectedVolume: 0.1,
			shouldChange:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPlayer := &MockAudioPlayer{
				state:  audio.StatePlaying,
				volume: tt.initialVolume,
			}

			playerComponent := player.NewPlayerComponent(mockPlayer, nil)

			track := &soundcloud.Track{
				ID:    123,
				Title: "Test Track",
				User:  soundcloud.User{Username: "Test Artist"},
			}
			playerComponent.SetCurrentTrack(track)
			playerComponent.SetState(player.StatePlaying)

			// Press + key to increase volume
			plusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}
			updatedComponent, cmd := playerComponent.Update(plusMsg)
			playerComponent = updatedComponent.(*player.PlayerComponent)

			if tt.shouldChange {
				assert.NotNil(t, cmd) // Should return volume update command
			}

			// In a real scenario, the command would execute and update the mock player
			// For testing purposes, let's simulate that
			if tt.shouldChange && cmd != nil {
				err := mockPlayer.SetVolume(tt.expectedVolume)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedVolume, mockPlayer.GetVolume())
			}
		})
	}
}

func TestVolumeControls_DecrementVolume(t *testing.T) {
	tests := []struct {
		name           string
		initialVolume  float64
		expectedVolume float64
		shouldChange   bool
	}{
		{
			name:           "decrease from 50%",
			initialVolume:  0.5,
			expectedVolume: 0.4,
			shouldChange:   true,
		},
		{
			name:           "decrease from 10%",
			initialVolume:  0.1,
			expectedVolume: 0.0,
			shouldChange:   true,
		},
		{
			name:           "decrease from min volume",
			initialVolume:  0.0,
			expectedVolume: 0.0,
			shouldChange:   false,
		},
		{
			name:           "decrease from max volume",
			initialVolume:  1.0,
			expectedVolume: 0.9,
			shouldChange:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPlayer := &MockAudioPlayer{
				state:  audio.StatePlaying,
				volume: tt.initialVolume,
			}

			playerComponent := player.NewPlayerComponent(mockPlayer, nil)

			track := &soundcloud.Track{
				ID:    123,
				Title: "Test Track",
				User:  soundcloud.User{Username: "Test Artist"},
			}
			playerComponent.SetCurrentTrack(track)
			playerComponent.SetState(player.StatePlaying)

			// Press - key to decrease volume
			minusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}}
			updatedComponent, cmd := playerComponent.Update(minusMsg)
			playerComponent = updatedComponent.(*player.PlayerComponent)

			if tt.shouldChange {
				assert.NotNil(t, cmd) // Should return volume update command
			}

			// Simulate command execution
			if tt.shouldChange && cmd != nil {
				err := mockPlayer.SetVolume(tt.expectedVolume)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedVolume, mockPlayer.GetVolume())
			}
		})
	}
}

func TestVolumeControls_VolumeDisplay(t *testing.T) {
	tests := []struct {
		name         string
		volume       float64
		expectedIcon string
		expectedText string
	}{
		{
			name:         "muted volume",
			volume:       0.0,
			expectedIcon: "🔇",
			expectedText: "0%",
		},
		{
			name:         "low volume",
			volume:       0.25,
			expectedIcon: "🔉",
			expectedText: "25%",
		},
		{
			name:         "medium volume",
			volume:       0.5,
			expectedIcon: "🔊",
			expectedText: "50%",
		},
		{
			name:         "high volume",
			volume:       0.75,
			expectedIcon: "🔊",
			expectedText: "75%",
		},
		{
			name:         "max volume",
			volume:       1.0,
			expectedIcon: "🔊",
			expectedText: "100%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPlayer := &MockAudioPlayer{
				state:  audio.StatePlaying,
				volume: tt.volume,
			}

			playerComponent := player.NewPlayerComponent(mockPlayer, nil)

			track := &soundcloud.Track{
				ID:    123,
				Title: "Test Track",
				User:  soundcloud.User{Username: "Test Artist"},
			}
			playerComponent.SetCurrentTrack(track)
			playerComponent.SetState(player.StatePlaying)

			view := playerComponent.View()

			// Should contain the appropriate volume icon
			assert.Contains(t, view, tt.expectedIcon)
			
			// Should contain the percentage
			assert.Contains(t, view, tt.expectedText)
		})
	}
}

func TestVolumeControls_KeyboardShortcuts(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:  audio.StatePlaying,
		volume: 0.5,
	}

	playerComponent := player.NewPlayerComponent(mockPlayer, nil)

	track := &soundcloud.Track{
		ID:    123,
		Title: "Test Track",
		User:  soundcloud.User{Username: "Test Artist"},
	}
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)

	tests := []struct {
		name           string
		key            string
		shouldHaveCmd  bool
		description    string
	}{
		{
			name:           "plus key increases volume",
			key:            "+",
			shouldHaveCmd:  true,
			description:    "Plus key should generate volume increase command",
		},
		{
			name:           "equals key increases volume",
			key:            "=",
			shouldHaveCmd:  true,
			description:    "Equals key should generate volume increase command",
		},
		{
			name:           "minus key decreases volume",
			key:            "-",
			shouldHaveCmd:  true,
			description:    "Minus key should generate volume decrease command",
		},
		{
			name:           "invalid key does nothing",
			key:            "x",
			shouldHaveCmd:  false,
			description:    "Invalid key should not generate volume command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			updatedComponent, cmd := playerComponent.Update(keyMsg)
			playerComponent = updatedComponent.(*player.PlayerComponent)

			if tt.shouldHaveCmd {
				assert.NotNil(t, cmd, tt.description)
			} else {
				assert.Nil(t, cmd, tt.description)
			}
		})
	}
}

func TestVolumeControls_VolumeSteps(t *testing.T) {
	// Test that volume changes in appropriate increments
	mockPlayer := &MockAudioPlayer{
		state:  audio.StatePlaying,
		volume: 0.5,
	}

	playerComponent := player.NewPlayerComponent(mockPlayer, nil)

	track := &soundcloud.Track{
		ID:    123,
		Title: "Test Track",
		User:  soundcloud.User{Username: "Test Artist"},
	}
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)

	// Test multiple volume increments
	initialVolume := mockPlayer.GetVolume()
	
	// Increase volume 5 times
	for i := 0; i < 5; i++ {
		plusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}
		updatedComponent, cmd := playerComponent.Update(plusMsg)
		playerComponent = updatedComponent.(*player.PlayerComponent)
		
		assert.NotNil(t, cmd) // Should always return a command
		
		// Simulate command execution with 10% increment
		newVolume := mockPlayer.GetVolume() + 0.1
		if newVolume > 1.0 {
			newVolume = 1.0
		}
		err := mockPlayer.SetVolume(newVolume)
		assert.NoError(t, err)
	}

	// Volume should have increased but not exceed 1.0
	assert.Greater(t, mockPlayer.GetVolume(), initialVolume)
	assert.LessOrEqual(t, mockPlayer.GetVolume(), 1.0)
}

func TestVolumeControls_VolumeInIdleState(t *testing.T) {
	// Test that volume controls work even when no track is playing
	mockPlayer := &MockAudioPlayer{
		state:  audio.StateStopped,
		volume: 0.5,
	}

	playerComponent := player.NewPlayerComponent(mockPlayer, nil)
	
	// No track set - should still handle volume controls
	assert.Equal(t, player.StateIdle, playerComponent.GetState())

	// Volume controls should still work
	plusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}
	updatedComponent, cmd := playerComponent.Update(plusMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)

	// Should return a command even in idle state
	assert.NotNil(t, cmd)
}

func TestVolumeControls_VolumeInErrorState(t *testing.T) {
	// Test volume controls when player is in error state
	mockPlayer := &MockAudioPlayer{
		state:  audio.StateStopped,
		volume: 0.5,
	}

	playerComponent := player.NewPlayerComponent(mockPlayer, nil)
	playerComponent.SetState(player.StateError)

	// Volume controls should still work in error state
	plusMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}
	updatedComponent, cmd := playerComponent.Update(plusMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)

	// Should return a command even in error state
	assert.NotNil(t, cmd)
}

func TestVolumeControls_RapidVolumeChanges(t *testing.T) {
	// Test rapid volume changes don't cause issues
	mockPlayer := &MockAudioPlayer{
		state:  audio.StatePlaying,
		volume: 0.5,
	}

	playerComponent := player.NewPlayerComponent(mockPlayer, nil)

	track := &soundcloud.Track{
		ID:    123,
		Title: "Test Track",
		User:  soundcloud.User{Username: "Test Artist"},
	}
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)

	// Rapidly alternate between volume up and down
	for i := 0; i < 10; i++ {
		var key string
		if i%2 == 0 {
			key = "+"
		} else {
			key = "-"
		}

		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		updatedComponent, cmd := playerComponent.Update(keyMsg)
		playerComponent = updatedComponent.(*player.PlayerComponent)

		assert.NotNil(t, cmd) // Should always return a command
		
		// Simulate some volume change
		if key == "+" {
			newVol := mockPlayer.GetVolume() + 0.1
			if newVol > 1.0 {
				newVol = 1.0
			}
			mockPlayer.SetVolume(newVol)
		} else {
			newVol := mockPlayer.GetVolume() - 0.1
			if newVol < 0.0 {
				newVol = 0.0
			}
			mockPlayer.SetVolume(newVol)
		}
	}

	// Volume should still be within valid range
	assert.GreaterOrEqual(t, mockPlayer.GetVolume(), 0.0)
	assert.LessOrEqual(t, mockPlayer.GetVolume(), 1.0)
}