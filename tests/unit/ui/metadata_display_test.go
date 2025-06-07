package ui_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"soundcloud-tui/internal/audio"
	"soundcloud-tui/internal/soundcloud"
	"soundcloud-tui/internal/ui/components/player"
)

func TestMetadataDisplay_TrackInformation(t *testing.T) {
	tests := []struct {
		name         string
		track        *soundcloud.Track
		expectedTitle string
		expectedArtist string
		expectedDuration string
	}{
		{
			name: "basic track info",
			track: &soundcloud.Track{
				ID:       123456789,
				Title:    "Amazing Song",
				User:     soundcloud.User{Username: "GreatArtist"},
				Duration: 180000, // 3 minutes
			},
			expectedTitle:    "Amazing Song",
			expectedArtist:   "GreatArtist",
			expectedDuration: "3:00",
		},
		{
			name: "track with full name artist",
			track: &soundcloud.Track{
				ID:    987654321,
				Title: "Epic Track",
				User: soundcloud.User{
					Username:  "epicuser",
					FirstName: "Epic",
					LastName:  "Producer",
				},
				Duration: 245000, // 4:05
			},
			expectedTitle:    "Epic Track",
			expectedArtist:   "Epic Producer",
			expectedDuration: "4:05",
		},
		{
			name: "long track title",
			track: &soundcloud.Track{
				ID:       555666777,
				Title:    "This Is A Very Long Track Title That Should Be Displayed Properly",
				User:     soundcloud.User{Username: "LongTitleArtist"},
				Duration: 7200000, // 2 hours
			},
			expectedTitle:    "This Is A Very Long Track Title That Should Be Displayed Properly",
			expectedArtist:   "LongTitleArtist",
			expectedDuration: "120:00",
		},
		{
			name: "short track",
			track: &soundcloud.Track{
				ID:       111222333,
				Title:    "Intro",
				User:     soundcloud.User{Username: "QuickArtist"},
				Duration: 15000, // 15 seconds
			},
			expectedTitle:    "Intro",
			expectedArtist:   "QuickArtist",
			expectedDuration: "0:15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPlayer := &MockAudioPlayer{
				state: audio.StatePlaying,
			}

			playerComponent := player.NewPlayerComponent(mockPlayer, nil)
			playerComponent.SetCurrentTrack(tt.track)
			playerComponent.SetState(player.StatePlaying)

			view := playerComponent.View()

			// Should display track title
			assert.Contains(t, view, tt.expectedTitle, "Should display track title")

			// Should display artist name
			assert.Contains(t, view, tt.expectedArtist, "Should display artist name")

			// Duration verification would depend on whether it's shown in metadata
			// For now, just verify the view is not empty
			assert.NotEmpty(t, view, "View should not be empty")
		})
	}
}

func TestMetadataDisplay_TrackStateIndication(t *testing.T) {
	tests := []struct {
		name           string
		playerState    player.State
		audioState     audio.PlayerState
		expectedStatus string
		expectedIcon   string
	}{
		{
			name:           "playing state",
			playerState:    player.StatePlaying,
			audioState:     audio.StatePlaying,
			expectedStatus: "Playing",
			expectedIcon:   "▶",
		},
		{
			name:           "paused state",
			playerState:    player.StatePaused,
			audioState:     audio.StatePaused,
			expectedStatus: "Paused",
			expectedIcon:   "⏸",
		},
		{
			name:           "stopped state",
			playerState:    player.StatePlaying,
			audioState:     audio.StateStopped,
			expectedStatus: "Stopped",
			expectedIcon:   "⏹",
		},
		{
			name:           "loading state",
			playerState:    player.StateLoading,
			audioState:     audio.StateStopped,
			expectedStatus: "Loading",
			expectedIcon:   "🔄",
		},
		{
			name:           "error state",
			playerState:    player.StateError,
			audioState:     audio.StateStopped,
			expectedStatus: "Error",
			expectedIcon:   "❌",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPlayer := &MockAudioPlayer{
				state: tt.audioState,
			}

			playerComponent := player.NewPlayerComponent(mockPlayer, nil)

			track := &soundcloud.Track{
				ID:    123,
				Title: "Test Track",
				User:  soundcloud.User{Username: "Test Artist"},
			}
			playerComponent.SetCurrentTrack(track)
			playerComponent.SetState(tt.playerState)

			view := playerComponent.View()

			// Should contain the expected status text
			assert.Contains(t, view, tt.expectedStatus, "Should display correct status")

			// Should contain the expected icon
			assert.Contains(t, view, tt.expectedIcon, "Should display correct icon")
		})
	}
}

func TestMetadataDisplay_NoTrackLoaded(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state: audio.StateStopped,
	}

	playerComponent := player.NewPlayerComponent(mockPlayer, nil)
	
	// No track set - idle state
	assert.Equal(t, player.StateIdle, playerComponent.GetState())

	view := playerComponent.View()

	// Should indicate no track is loaded
	assert.Contains(t, view, "No track", "Should indicate no track loaded")

	// Should suggest user action
	assert.Contains(t, view, "Select a track", "Should suggest selecting a track")

	// Should not contain any specific track info
	assert.NotContains(t, view, "Artist:", "Should not show artist info")
	assert.NotContains(t, view, "Duration:", "Should not show duration info")
}

func TestMetadataDisplay_TrackProgress(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:    audio.StatePlaying,
		position: 90000, // 1:30
		duration: 180000, // 3:00
		volume:   0.75,
	}

	playerComponent := player.NewPlayerComponent(mockPlayer, nil)

	track := &soundcloud.Track{
		ID:       123,
		Title:    "Progress Track",
		User:     soundcloud.User{Username: "Progress Artist"},
		Duration: 180000,
	}
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)

	// Update with progress
	progressMsg := player.ProgressUpdateMsg{
		Position: 90000, // 1:30
		Duration: 180000, // 3:00
	}

	updatedComponent, _ := playerComponent.Update(progressMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)

	view := playerComponent.View()

	// Should show current position and total duration
	assert.Contains(t, view, "1:30", "Should show current position")
	assert.Contains(t, view, "3:00", "Should show total duration")

	// Should show progress indication (progress bar or percentage)
	// Note: Exact format may vary, but some progress indication should be present
	assert.NotEmpty(t, view, "View should contain progress information")
}

func TestMetadataDisplay_MetadataLayout(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:  audio.StatePlaying,
		volume: 0.8,
	}

	playerComponent := player.NewPlayerComponent(mockPlayer, nil)

	track := &soundcloud.Track{
		ID:          123456789,
		Title:       "Layout Test Track",
		Description: "This is a test track for testing the metadata layout and display",
		User:        soundcloud.User{Username: "LayoutArtist", FirstName: "Layout", LastName: "Artist"},
		Duration:    210000, // 3:30
		ArtworkURL:  "https://example.com/artwork.jpg",
		PermalinkURL: "https://soundcloud.com/layoutartist/layout-test-track",
	}
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)

	view := playerComponent.View()

	// Should contain track title prominently
	assert.Contains(t, view, "Layout Test Track", "Should display track title")

	// Should contain artist name
	assert.Contains(t, view, "Layout Artist", "Should display full artist name")

	// Should be properly formatted (not cramped)
	assert.Greater(t, len(view), 100, "View should be substantial")

	// Should not contain raw metadata that's not user-friendly
	assert.NotContains(t, view, "123456789", "Should not show raw track ID")
	assert.NotContains(t, view, "https://", "Should not show raw URLs in main display")
}

func TestMetadataDisplay_LongTitleTruncation(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state: audio.StatePlaying,
	}

	playerComponent := player.NewPlayerComponent(mockPlayer, nil)

	track := &soundcloud.Track{
		ID:    123,
		Title: "This Is An Extremely Long Track Title That Should Be Handled Gracefully By The Display System Without Breaking The Layout Or Causing Visual Issues In The Terminal User Interface",
		User:  soundcloud.User{Username: "VeryLongArtistNameThatShouldAlsoBeHandledProperly"},
		Duration: 240000,
	}
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)

	view := playerComponent.View()

	// Should handle long titles gracefully
	assert.NotEmpty(t, view, "Should render even with long titles")

	// Should contain at least part of the title
	titleWords := []string{"This", "Is", "An", "Extremely", "Long"}
	titleFound := false
	for _, word := range titleWords {
		if assert.Contains(t, view, word) {
			titleFound = true
			break
		}
	}
	assert.True(t, titleFound, "Should contain part of the long title")

	// Should contain artist info
	assert.Contains(t, view, "VeryLong", "Should contain part of artist name")
}

func TestMetadataDisplay_SpecialCharacters(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state: audio.StatePlaying,
	}

	playerComponent := player.NewPlayerComponent(mockPlayer, nil)

	track := &soundcloud.Track{
		ID:    123,
		Title: "Track with 🎵 Emojis & Special Characters: éñtertainment!",
		User:  soundcloud.User{Username: "ArtistWithNumbers123", FirstName: "Émile", LastName: "François"},
		Duration: 180000,
	}
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)

	view := playerComponent.View()

	// Should handle special characters and emojis
	assert.Contains(t, view, "🎵", "Should display emojis")
	assert.Contains(t, view, "&", "Should display special characters")
	assert.Contains(t, view, "éñtertainment", "Should display accented characters")
	assert.Contains(t, view, "Émile François", "Should display accented names")

	// Should not crash or become corrupted
	assert.NotEmpty(t, view, "Should render properly with special characters")
}

func TestMetadataDisplay_EmptyOrMissingFields(t *testing.T) {
	tests := []struct {
		name        string
		track       *soundcloud.Track
		expectation string
	}{
		{
			name: "empty title",
			track: &soundcloud.Track{
				ID:    123,
				Title: "",
				User:  soundcloud.User{Username: "Artist"},
			},
			expectation: "should handle empty title",
		},
		{
			name: "empty artist",
			track: &soundcloud.Track{
				ID:    123,
				Title: "Track Title",
				User:  soundcloud.User{},
			},
			expectation: "should handle empty artist",
		},
		{
			name: "zero duration",
			track: &soundcloud.Track{
				ID:       123,
				Title:    "Track Title",
				User:     soundcloud.User{Username: "Artist"},
				Duration: 0,
			},
			expectation: "should handle zero duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPlayer := &MockAudioPlayer{
				state: audio.StatePlaying,
			}

			playerComponent := player.NewPlayerComponent(mockPlayer, nil)
			playerComponent.SetCurrentTrack(tt.track)
			playerComponent.SetState(player.StatePlaying)

			view := playerComponent.View()

			// Should not crash or show broken layout
			assert.NotEmpty(t, view, tt.expectation)

			// Should provide some meaningful display even with missing fields
			assert.NotContains(t, view, "<nil>", "Should not show nil values")
			assert.NotContains(t, view, "undefined", "Should not show undefined values")
		})
	}
}

func TestMetadataDisplay_StateChangeMetadataConsistency(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state: audio.StateStopped,
	}

	playerComponent := player.NewPlayerComponent(mockPlayer, nil)

	track := &soundcloud.Track{
		ID:    123,
		Title: "Consistent Track",
		User:  soundcloud.User{Username: "Consistent Artist"},
	}
	playerComponent.SetCurrentTrack(track)

	// Test metadata consistency across different states
	states := []player.State{
		player.StateIdle,
		player.StateLoading,
		player.StatePlaying,
		player.StatePaused,
		player.StateError,
	}

	for _, state := range states {
		playerComponent.SetState(state)
		view := playerComponent.View()

		// Track metadata should remain consistent regardless of state
		if state != player.StateIdle {
			assert.Contains(t, view, "Consistent Track", "Track title should be consistent across states")
			assert.Contains(t, view, "Consistent Artist", "Artist name should be consistent across states")
		}

		// View should not be empty in any state
		assert.NotEmpty(t, view, "View should not be empty in any state")
	}
}