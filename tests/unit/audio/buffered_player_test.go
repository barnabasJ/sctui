package audio_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"soundcloud-tui/internal/audio"
)

func TestBufferedStreamPlayer_NewPlayer(t *testing.T) {
	player := audio.NewBufferedStreamPlayer()
	
	require.NotNil(t, player)
	assert.Equal(t, audio.StateStopped, player.GetState())
	assert.Equal(t, float64(1.0), player.GetVolume())
	assert.Equal(t, time.Duration(0), player.GetPosition())
}

func TestBufferedStreamPlayer_ErrorHandling(t *testing.T) {
	player := audio.NewBufferedStreamPlayer()
	defer player.Close()
	
	tests := []struct {
		name        string
		streamURL   string
		expectError bool
	}{
		{
			name:        "empty URL returns error",
			streamURL:   "",
			expectError: true,
		},
		{
			name:        "invalid URL returns error",
			streamURL:   "invalid-url",
			expectError: true,
		},
		{
			name:        "non-existent URL returns error after retries",
			streamURL:   "https://example.com/nonexistent.mp3",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			err := player.Play(ctx, tt.streamURL)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBufferedStreamPlayer_StateManagement(t *testing.T) {
	player := audio.NewBufferedStreamPlayer()
	defer player.Close()
	
	// Initial state should be stopped
	assert.Equal(t, audio.StateStopped, player.GetState())
	
	// Test pause when stopped (should return error)
	err := player.Pause()
	assert.Error(t, err)
	
	// Test resume when stopped (should return error)
	err = player.Resume()
	assert.Error(t, err)
}

func TestBufferedStreamPlayer_VolumeControl(t *testing.T) {
	player := audio.NewBufferedStreamPlayer()
	defer player.Close()
	
	// Test initial volume
	assert.Equal(t, float64(1.0), player.GetVolume())
	
	// Test setting valid volume
	err := player.SetVolume(0.5)
	assert.NoError(t, err)
	assert.Equal(t, float64(0.5), player.GetVolume())
	
	// Test setting invalid volume (too high)
	err = player.SetVolume(1.5)
	assert.Error(t, err)
	
	// Test setting invalid volume (negative)
	err = player.SetVolume(-0.1)
	assert.Error(t, err)
	
	// Volume should remain unchanged after invalid attempts
	assert.Equal(t, float64(0.5), player.GetVolume())
}

func TestBufferedStreamPlayer_ContextCancellation(t *testing.T) {
	player := audio.NewBufferedStreamPlayer()
	defer player.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	// This should timeout/cancel quickly
	err := player.Play(ctx, "https://example.com/long-audio.mp3")
	assert.Error(t, err)
}

func TestBufferedStreamPlayer_SeekOperations(t *testing.T) {
	player := audio.NewBufferedStreamPlayer()
	defer player.Close()
	
	// Test seek when no stream loaded
	err := player.Seek(time.Second)
	assert.Error(t, err)
	
	// Test seek with negative position
	err = player.Seek(-time.Second)
	assert.Error(t, err)
}

func TestBufferedStreamPlayer_CallbacksAndCleanup(t *testing.T) {
	player := audio.NewBufferedStreamPlayer()
	
	stateChangeCalled := false
	errorCalled := false
	
	// Set callbacks
	player.SetStateChangeCallback(func(state audio.PlayerState) {
		stateChangeCalled = true
	})
	
	player.SetErrorCallback(func(err error) {
		errorCalled = true
	})
	
	// Try to play invalid URL to trigger error callback
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	err := player.Play(ctx, "https://example.com/nonexistent.mp3")
	assert.Error(t, err)
	
	// Give callbacks time to execute
	time.Sleep(100 * time.Millisecond)
	
	// Close should not panic
	assert.NoError(t, player.Close())
}