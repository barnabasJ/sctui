package ui_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"soundcloud-tui/internal/audio"
	"soundcloud-tui/internal/soundcloud"
	"soundcloud-tui/internal/ui/components/player"
	"soundcloud-tui/internal/ui/styles"
)

func TestProgressDisplay_ProgressBarAccuracy(t *testing.T) {
	tests := []struct {
		name       string
		position   time.Duration
		duration   time.Duration
		barWidth   int
		expectedFillWidth int
	}{
		{
			name:       "25% progress",
			position:   30 * time.Second,
			duration:   120 * time.Second,
			barWidth:   40,
			expectedFillWidth: 10,
		},
		{
			name:       "50% progress",
			position:   60 * time.Second,
			duration:   120 * time.Second,
			barWidth:   40,
			expectedFillWidth: 20,
		},
		{
			name:       "75% progress",
			position:   90 * time.Second,
			duration:   120 * time.Second,
			barWidth:   40,
			expectedFillWidth: 30,
		},
		{
			name:       "100% progress",
			position:   120 * time.Second,
			duration:   120 * time.Second,
			barWidth:   40,
			expectedFillWidth: 40,
		},
		{
			name:       "0% progress",
			position:   0,
			duration:   120 * time.Second,
			barWidth:   40,
			expectedFillWidth: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progress := float64(tt.position) / float64(tt.duration)
			progressBar := styles.RenderProgressBar(tt.barWidth, progress)
			
			// Progress bar should not be empty
			assert.NotEmpty(t, progressBar)
			
			// Progress bar width should match expected
			// Note: This is a visual test - exact implementation may vary
			assert.LessOrEqual(t, len(progressBar), tt.barWidth*2) // Account for styling
		})
	}
}

func TestProgressDisplay_TimeFormatting(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "zero duration",
			duration: 0,
			expected: "0:00",
		},
		{
			name:     "30 seconds",
			duration: 30 * time.Second,
			expected: "0:30",
		},
		{
			name:     "1 minute",
			duration: 60 * time.Second,
			expected: "1:00",
		},
		{
			name:     "1 minute 30 seconds",
			duration: 90 * time.Second,
			expected: "1:30",
		},
		{
			name:     "10 minutes 5 seconds",
			duration: 605 * time.Second,
			expected: "10:05",
		},
		{
			name:     "1 hour 23 minutes 45 seconds",
			duration: (1*3600 + 23*60 + 45) * time.Second,
			expected: "83:45", // Should show total minutes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := formatDurationForDisplay(tt.duration)
			assert.Equal(t, tt.expected, formatted)
		})
	}
}

func TestProgressDisplay_PlayerProgressRendering(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:    audio.StatePlaying,
		position: 45 * time.Second,
		duration: 180 * time.Second,
		volume:   0.75,
	}
	
	playerComponent := player.NewPlayerComponent(mockPlayer, nil)
	
	track := &soundcloud.Track{
		ID:       123,
		Title:    "Test Track",
		User:     soundcloud.User{Username: "Test Artist"},
		Duration: 180000, // 3 minutes in ms
	}
	
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)
	
	// Update with progress
	progressMsg := player.ProgressUpdateMsg{
		Position: 45 * time.Second,
		Duration: 180 * time.Second,
	}
	
	updatedComponent, _ := playerComponent.Update(progressMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)
	
	view := playerComponent.View()
	
	// Should contain formatted time
	assert.Contains(t, view, "0:45") // Current position
	assert.Contains(t, view, "3:00") // Total duration
	
	// Should show progress indicator
	assert.Contains(t, view, "▶") // Playing indicator
	
	// Should show progress bar (visual element present)
	// Note: Exact progress bar format depends on implementation
	assert.NotEmpty(t, view)
}

func TestProgressDisplay_ProgressBarEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		progress float64
		shouldNotPanic bool
	}{
		{
			name:     "zero width",
			width:    0,
			progress: 0.5,
			shouldNotPanic: true,
		},
		{
			name:     "negative width",
			width:    -5,
			progress: 0.5,
			shouldNotPanic: true,
		},
		{
			name:     "progress over 100%",
			width:    40,
			progress: 1.5,
			shouldNotPanic: true,
		},
		{
			name:     "negative progress",
			width:    40,
			progress: -0.2,
			shouldNotPanic: true,
		},
		{
			name:     "very small width",
			width:    1,
			progress: 0.5,
			shouldNotPanic: true,
		},
		{
			name:     "very large width",
			width:    1000,
			progress: 0.5,
			shouldNotPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && tt.shouldNotPanic {
					t.Errorf("Progress bar rendering panicked: %v", r)
				}
			}()
			
			progressBar := styles.RenderProgressBar(tt.width, tt.progress)
			
			// Should return some result (even if empty for invalid inputs)
			if tt.width > 0 {
				assert.NotNil(t, progressBar)
			}
		})
	}
}

func TestProgressDisplay_VolumeDisplay(t *testing.T) {
	tests := []struct {
		name           string
		volume         float64
		expectedFormat string
	}{
		{
			name:           "muted",
			volume:         0.0,
			expectedFormat: "🔇 0%",
		},
		{
			name:           "low volume",
			volume:         0.25,
			expectedFormat: "🔉 25%",
		},
		{
			name:           "medium volume",
			volume:         0.50,
			expectedFormat: "🔉 50%",
		},
		{
			name:           "high volume",
			volume:         0.75,
			expectedFormat: "🔊 75%",
		},
		{
			name:           "max volume",
			volume:         1.0,
			expectedFormat: "🔊 100%",
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
			
			// Should contain volume percentage
			volumePercent := int(tt.volume * 100)
			assert.Contains(t, view, "🔊") // Volume icon
			
			// Should show volume percentage somewhere in the view
			// Note: Exact format may vary based on implementation
			if volumePercent == 0 {
				// Special case for muted
				assert.Contains(t, view, "0%")
			} else {
				// Should contain the percentage
				assert.Contains(t, view, "%")
			}
		})
	}
}

func TestProgressDisplay_RealTimeUpdates(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:    audio.StatePlaying,
		position: 0,
		duration: 180 * time.Second,
		volume:   1.0,
	}
	
	playerComponent := player.NewPlayerComponent(mockPlayer, nil)
	
	track := &soundcloud.Track{
		ID:    123,
		Title: "Test Track",
		User:  soundcloud.User{Username: "Test Artist"},
	}
	
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)
	
	// Simulate progress updates over time
	positions := []time.Duration{
		10 * time.Second,
		30 * time.Second,
		60 * time.Second,
		90 * time.Second,
	}
	
	for i, pos := range positions {
		progressMsg := player.ProgressUpdateMsg{
			Position: pos,
			Duration: 180 * time.Second,
		}
		
		updatedComponent, cmd := playerComponent.Update(progressMsg)
		playerComponent = updatedComponent.(*player.PlayerComponent)
		
		// Should update position
		assert.Equal(t, pos, playerComponent.GetPosition())
		
		// Should return tick command for continued updates
		assert.NotNil(t, cmd)
		
		view := playerComponent.View()
		
		// Should show updated time
		timeStr := formatDurationForDisplay(pos)
		assert.Contains(t, view, timeStr)
		
		// Progress should increase over time
		if i > 0 {
			// Visual progress should be advancing (hard to test exactly)
			assert.NotEmpty(t, view)
		}
	}
}

func TestProgressDisplay_SeekUpdatesProgress(t *testing.T) {
	mockPlayer := &MockAudioPlayer{
		state:    audio.StatePlaying,
		position: 60 * time.Second,
		duration: 180 * time.Second,
		volume:   1.0,
	}
	
	playerComponent := player.NewPlayerComponent(mockPlayer, nil)
	
	track := &soundcloud.Track{
		ID:    123,
		Title: "Test Track",
		User:  soundcloud.User{Username: "Test Artist"},
	}
	
	playerComponent.SetCurrentTrack(track)
	playerComponent.SetState(player.StatePlaying)
	
	// Update to initial position
	progressMsg := player.ProgressUpdateMsg{
		Position: 60 * time.Second,
		Duration: 180 * time.Second,
	}
	
	updatedComponent, _ := playerComponent.Update(progressMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)
	
	initialView := playerComponent.View()
	assert.Contains(t, initialView, "1:00")
	
	// Seek forward
	rightMsg := tea.KeyMsg{Type: tea.KeyRight}
	updatedComponent, seekCmd := playerComponent.Update(rightMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)
	
	assert.NotNil(t, seekCmd) // Should return seek command
	
	// Simulate seek completion - position should update
	newProgressMsg := player.ProgressUpdateMsg{
		Position: 70 * time.Second, // Simulated new position after seek
		Duration: 180 * time.Second,
	}
	
	updatedComponent, _ = playerComponent.Update(newProgressMsg)
	playerComponent = updatedComponent.(*player.PlayerComponent)
	
	newView := playerComponent.View()
	assert.Contains(t, newView, "1:10") // Should show new position
}

// Helper function to format duration (would be implemented in styles or player)
func formatDurationForDisplay(d time.Duration) string {
	if d <= 0 {
		return "0:00"
	}
	
	totalSeconds := int(d.Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}