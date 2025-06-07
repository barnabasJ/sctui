package audio_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"soundcloud-tui/internal/audio"
)

func TestSoundCloudStreamExtractor_ExtractStreamURL(t *testing.T) {
	tests := []struct {
		name        string
		trackID     int64
		clientID    string
		wantErr     bool
		wantFormat  string
		wantQuality string
	}{
		{
			name:        "valid track ID returns stream info",
			trackID:     123456789,
			clientID:    "test-client-id",
			wantErr:     false,
			wantFormat:  "mp3",
			wantQuality: "sq",
		},
		{
			name:     "invalid track ID returns error",
			trackID:  -1,
			clientID: "test-client-id",
			wantErr:  true,
		},
		{
			name:     "empty client ID returns error",
			trackID:  123456789,
			clientID: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := audio.NewSoundCloudStreamExtractor(tt.clientID)
			
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			streamInfo, err := extractor.ExtractStreamURL(ctx, tt.trackID)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, streamInfo)
				return
			}
			
			require.NoError(t, err)
			require.NotNil(t, streamInfo)
			
			// Verify stream info structure
			assert.NotEmpty(t, streamInfo.URL, "Stream URL should not be empty")
			assert.Equal(t, tt.wantFormat, streamInfo.Format)
			assert.Equal(t, tt.wantQuality, streamInfo.Quality)
			assert.Greater(t, streamInfo.Duration, int64(0), "Duration should be positive")
			
			// Verify URL format
			assert.Contains(t, streamInfo.URL, "http", "Stream URL should be a valid HTTP URL")
		})
	}
}

func TestSoundCloudStreamExtractor_GetAvailableQualities(t *testing.T) {
	tests := []struct {
		name           string
		trackID        int64
		clientID       string
		wantQualities  []string
		wantErr        bool
	}{
		{
			name:          "valid track returns available qualities",
			trackID:       123456789,
			clientID:      "test-client-id",
			wantQualities: []string{"sq", "hq"},
			wantErr:       false,
		},
		{
			name:     "invalid track ID returns error",
			trackID:  -1,
			clientID: "test-client-id",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := audio.NewSoundCloudStreamExtractor(tt.clientID)
			
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			qualities, err := extractor.GetAvailableQualities(ctx, tt.trackID)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, qualities)
				return
			}
			
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.wantQualities, qualities)
			assert.NotEmpty(t, qualities, "Should return at least one quality option")
		})
	}
}

func TestSoundCloudStreamExtractor_ValidateStreamURL(t *testing.T) {
	tests := []struct {
		name      string
		streamURL string
		clientID  string
		wantValid bool
		wantErr   bool
	}{
		{
			name:      "valid stream URL returns true",
			streamURL: "https://cf-media.sndcdn.com/test.mp3",
			clientID:  "test-client-id",
			wantValid: true,
			wantErr:   false,
		},
		{
			name:      "invalid stream URL returns false",
			streamURL: "https://invalid-url.com/test.mp3",
			clientID:  "test-client-id",
			wantValid: false,
			wantErr:   false,
		},
		{
			name:      "empty URL returns error",
			streamURL: "",
			clientID:  "test-client-id",
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "malformed URL returns error",
			streamURL: "not-a-url",
			clientID:  "test-client-id",
			wantValid: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := audio.NewSoundCloudStreamExtractor(tt.clientID)
			
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			valid, err := extractor.ValidateStreamURL(ctx, tt.streamURL)
			
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			assert.Equal(t, tt.wantValid, valid)
		})
	}
}

func TestSoundCloudStreamExtractor_ContextCancellation(t *testing.T) {
	extractor := audio.NewSoundCloudStreamExtractor("test-client-id")
	
	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	_, err := extractor.ExtractStreamURL(ctx, 123456789)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestSoundCloudStreamExtractor_Timeout(t *testing.T) {
	extractor := audio.NewSoundCloudStreamExtractor("test-client-id")
	
	// Test with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	
	time.Sleep(10 * time.Millisecond) // Ensure timeout has passed
	
	_, err := extractor.ExtractStreamURL(ctx, 123456789)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}