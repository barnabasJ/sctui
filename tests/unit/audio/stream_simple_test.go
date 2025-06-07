package audio_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	soundcloudapi "github.com/zackradisic/soundcloud-api"
	
	"soundcloud-tui/internal/audio"
)

func TestStreamExtractor_WithMocks_Success(t *testing.T) {
	// Create mock API that returns valid track data
	mockAPI := &MockSoundCloudAPI{
		GetTrackInfoFunc: func(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error) {
			return []soundcloudapi.Track{
				{
					ID:         123456789,
					Title:      "Test Track",
					DurationMS: 240000, // 4 minutes
					Media: soundcloudapi.Media{
						Transcodings: []soundcloudapi.Transcoding{
							{
								URL: "https://api.soundcloud.com/tracks/123/stream",
								Format: soundcloudapi.TranscodingFormat{
									Protocol: "progressive",
									MimeType: "audio/mpeg",
								},
							},
						},
					},
				},
			}, nil
		},
	}
	
	// Create extractor with mock
	extractor := audio.NewSoundCloudStreamExtractorWithAPI(mockAPI)
	require.NotNil(t, extractor)
	
	// Test ExtractStreamURL
	ctx := context.Background()
	streamInfo, err := extractor.ExtractStreamURL(ctx, 123456789)
	
	require.NoError(t, err)
	require.NotNil(t, streamInfo)
	
	assert.NotEmpty(t, streamInfo.URL)
	assert.Equal(t, "mp3", streamInfo.Format)
	assert.Equal(t, "sq", streamInfo.Quality)
	assert.Equal(t, int64(240000), streamInfo.Duration)
	assert.Contains(t, streamInfo.URL, "http")
}

func TestStreamExtractor_WithMocks_InvalidTrackID(t *testing.T) {
	// Create mock API that returns error for invalid track ID
	mockAPI := &MockSoundCloudAPI{
		GetTrackInfoFunc: func(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error) {
			if len(options.ID) > 0 && options.ID[0] <= 0 {
				return nil, fmt.Errorf("invalid track ID")
			}
			return []soundcloudapi.Track{}, nil // No tracks found
		},
	}
	
	extractor := audio.NewSoundCloudStreamExtractorWithAPI(mockAPI)
	require.NotNil(t, extractor)
	
	ctx := context.Background()
	
	// Test with invalid track ID
	streamInfo, err := extractor.ExtractStreamURL(ctx, -1)
	assert.Error(t, err)
	assert.Nil(t, streamInfo)
	assert.Contains(t, err.Error(), "invalid track ID")
}

func TestStreamExtractor_GetAvailableQualities_Success(t *testing.T) {
	// Create mock API with multiple transcodings
	mockAPI := &MockSoundCloudAPI{
		GetTrackInfoFunc: func(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error) {
			return []soundcloudapi.Track{
				{
					ID:         123456789,
					Title:      "Test Track",
					DurationMS: 240000,
					Media: soundcloudapi.Media{
						Transcodings: []soundcloudapi.Transcoding{
							{
								Format: soundcloudapi.TranscodingFormat{
									Protocol: "progressive",
								},
							},
							{
								Format: soundcloudapi.TranscodingFormat{
									Protocol: "hls",
								},
							},
						},
					},
				},
			}, nil
		},
	}
	
	extractor := audio.NewSoundCloudStreamExtractorWithAPI(mockAPI)
	require.NotNil(t, extractor)
	
	ctx := context.Background()
	qualities, err := extractor.GetAvailableQualities(ctx, 123456789)
	
	require.NoError(t, err)
	assert.Contains(t, qualities, "sq")  // From progressive
	assert.Contains(t, qualities, "hq")  // From HLS
	assert.Len(t, qualities, 2)
}

func TestStreamExtractor_ContextCancellation_Works(t *testing.T) {
	mockAPI := &MockSoundCloudAPI{}
	extractor := audio.NewSoundCloudStreamExtractorWithAPI(mockAPI)
	
	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	_, err := extractor.ExtractStreamURL(ctx, 123456789)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}