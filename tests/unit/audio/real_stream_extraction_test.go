package audio_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	soundcloudapi "github.com/zackradisic/soundcloud-api"

	"soundcloud-tui/internal/audio"
)

// RealSoundCloudAPI defines the interface for the actual SoundCloud API
type RealSoundCloudAPI interface {
	GetTrackInfo(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error)
	GetDownloadURL(trackURL string, format string) (string, error)
}

// MockRealSoundCloudAPI provides a mock implementation for testing real API calls
type MockRealSoundCloudAPI struct {
	GetTrackInfoFunc   func(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error)
	GetDownloadURLFunc func(trackURL string, format string) (string, error)
}

func (m *MockRealSoundCloudAPI) GetTrackInfo(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error) {
	if m.GetTrackInfoFunc != nil {
		return m.GetTrackInfoFunc(options)
	}
	return nil, fmt.Errorf("mock GetTrackInfo not implemented")
}

func (m *MockRealSoundCloudAPI) GetDownloadURL(trackURL string, format string) (string, error) {
	if m.GetDownloadURLFunc != nil {
		return m.GetDownloadURLFunc(trackURL, format)
	}
	return "", fmt.Errorf("mock GetDownloadURL not implemented")
}

func (m *MockRealSoundCloudAPI) GetTrackInfoWithOptions(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error) {
	// Delegate to GetTrackInfo for consistency
	return m.GetTrackInfo(options)
}

func TestRealStreamExtraction_ValidTrackWithProgressiveFormat(t *testing.T) {
	// Create a track with progressive transcoding
	mockTrack := soundcloudapi.Track{
		ID:         123456789,
		Title:      "Test Track",
		DurationMS: 180000, // 3 minutes
		PermalinkURL: "https://soundcloud.com/artist/test-track",
		Media: soundcloudapi.Media{
			Transcodings: []soundcloudapi.Transcoding{
				{
					URL:    "https://api-v2.soundcloud.com/media/soundcloud:tracks:123456789/stream/progressive",
					Preset: "mp3_1_0",
					Format: soundcloudapi.TranscodingFormat{
						Protocol: "progressive",
						MimeType: "audio/mpeg",
					},
				},
			},
		},
	}

	// Expected signed CloudFront URL
	expectedURL := "https://cf-media.sndcdn.com/abc123def.128.mp3?Policy=eyJTdGF0ZW1lbnQiOlt7IlJlc291cmNlIjoiaHR0cHM6Ly9jZi1tZWRpYS5zbmRjZG4uY29tL2FiYzEyM2RlZi4xMjgubXAzIiwiQ29uZGl0aW9uIjp7IkRhdGVMZXNzVGhhbiI6eyJBV1M6RXBvY2hUaW1lIjoxNzAzNTI5NjAwfX19XX0_&Signature=abc123def&Key-Pair-Id=APKAJ123DEF456"

	mockAPI := &MockRealSoundCloudAPI{
		GetTrackInfoFunc: func(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error) {
			assert.Equal(t, []int64{123456789}, options.ID)
			return []soundcloudapi.Track{mockTrack}, nil
		},
		GetDownloadURLFunc: func(trackURL string, format string) (string, error) {
			assert.Equal(t, mockTrack.PermalinkURL, trackURL)
			assert.Equal(t, "progressive", format)
			return expectedURL, nil
		},
	}

	extractor := audio.NewRealSoundCloudStreamExtractor(mockAPI)
	ctx := context.Background()

	streamInfo, err := extractor.ExtractStreamURL(ctx, 123456789)

	require.NoError(t, err)
	require.NotNil(t, streamInfo)
	assert.Equal(t, expectedURL, streamInfo.URL)
	assert.Equal(t, "mp3", streamInfo.Format)
	assert.Equal(t, "progressive", streamInfo.Quality)
	assert.Equal(t, int64(180000), streamInfo.Duration)

	// Validate that URL is a proper CloudFront signed URL
	parsedURL, err := url.Parse(streamInfo.URL)
	require.NoError(t, err)
	assert.Equal(t, "cf-media.sndcdn.com", parsedURL.Host)
	assert.Contains(t, parsedURL.Path, ".mp3")
	assert.Contains(t, parsedURL.RawQuery, "Policy=")
	assert.Contains(t, parsedURL.RawQuery, "Signature=")
	assert.Contains(t, parsedURL.RawQuery, "Key-Pair-Id=")
}

func TestRealStreamExtraction_FallbackToHLSWhenNoProgressive(t *testing.T) {
	mockTrack := soundcloudapi.Track{
		ID:         987654321,
		Title:      "HLS Only Track",
		DurationMS: 240000,
		PermalinkURL: "https://soundcloud.com/artist/hls-track",
		Media: soundcloudapi.Media{
			Transcodings: []soundcloudapi.Transcoding{
				{
					URL:    "https://api-v2.soundcloud.com/media/soundcloud:tracks:987654321/stream/hls",
					Preset: "mp3_1_0",
					Format: soundcloudapi.TranscodingFormat{
						Protocol: "hls",
						MimeType: "audio/mpeg",
					},
				},
			},
		},
	}

	expectedHLSURL := "https://cf-media.sndcdn.com/playlist.m3u8?Policy=abc&Signature=def&Key-Pair-Id=ghi"

	mockAPI := &MockRealSoundCloudAPI{
		GetTrackInfoFunc: func(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error) {
			return []soundcloudapi.Track{mockTrack}, nil
		},
		GetDownloadURLFunc: func(trackURL string, format string) (string, error) {
			// Should fallback to HLS when progressive isn't available
			assert.Equal(t, "hls", format)
			return expectedHLSURL, nil
		},
	}

	extractor := audio.NewRealSoundCloudStreamExtractor(mockAPI)
	ctx := context.Background()

	streamInfo, err := extractor.ExtractStreamURL(ctx, 987654321)

	require.NoError(t, err)
	assert.Equal(t, expectedHLSURL, streamInfo.URL)
	assert.Equal(t, "hls", streamInfo.Format)
	assert.Equal(t, "hls", streamInfo.Quality)
}

func TestRealStreamExtraction_PreferProgressiveOverHLS(t *testing.T) {
	mockTrack := soundcloudapi.Track{
		ID:         555666777,
		Title:      "Multi-Format Track",
		DurationMS: 200000,
		PermalinkURL: "https://soundcloud.com/artist/multi-format",
		Media: soundcloudapi.Media{
			Transcodings: []soundcloudapi.Transcoding{
				{
					URL:    "https://api-v2.soundcloud.com/media/soundcloud:tracks:555666777/stream/hls",
					Preset: "mp3_1_0",
					Format: soundcloudapi.TranscodingFormat{
						Protocol: "hls",
						MimeType: "audio/mpeg",
					},
				},
				{
					URL:    "https://api-v2.soundcloud.com/media/soundcloud:tracks:555666777/stream/progressive",
					Preset: "mp3_1_0",
					Format: soundcloudapi.TranscodingFormat{
						Protocol: "progressive",
						MimeType: "audio/mpeg",
					},
				},
			},
		},
	}

	mockAPI := &MockRealSoundCloudAPI{
		GetTrackInfoFunc: func(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error) {
			return []soundcloudapi.Track{mockTrack}, nil
		},
		GetDownloadURLFunc: func(trackURL string, format string) (string, error) {
			// Should prefer progressive format even when HLS is listed first
			assert.Equal(t, "progressive", format)
			return "https://cf-media.sndcdn.com/progressive.mp3?auth=params", nil
		},
	}

	extractor := audio.NewRealSoundCloudStreamExtractor(mockAPI)
	ctx := context.Background()

	streamInfo, err := extractor.ExtractStreamURL(ctx, 555666777)

	require.NoError(t, err)
	assert.Equal(t, "progressive", streamInfo.Quality)
}

func TestRealStreamExtraction_HandleAPIErrors(t *testing.T) {
	tests := []struct {
		name          string
		trackID       int64
		apiError      error
		downloadError error
		expectedError string
	}{
		{
			name:          "track info API error",
			trackID:       123,
			apiError:      fmt.Errorf("API rate limit exceeded"),
			expectedError: "failed to get track info",
		},
		{
			name:          "download URL API error",
			trackID:       456,
			downloadError: fmt.Errorf("track not available in your region"),
			expectedError: "failed to get download URL",
		},
		{
			name:          "track not found",
			trackID:       789,
			apiError:      fmt.Errorf("track not found"),
			expectedError: "failed to get track info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := &MockRealSoundCloudAPI{
				GetTrackInfoFunc: func(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error) {
					if tt.apiError != nil {
						return nil, tt.apiError
					}
					return []soundcloudapi.Track{{
						ID: tt.trackID,
						PermalinkURL: "https://soundcloud.com/test/track",
						Media: soundcloudapi.Media{
							Transcodings: []soundcloudapi.Transcoding{
								{
									Format: soundcloudapi.TranscodingFormat{Protocol: "progressive"},
								},
							},
						},
					}}, nil
				},
				GetDownloadURLFunc: func(trackURL string, format string) (string, error) {
					if tt.downloadError != nil {
						return "", tt.downloadError
					}
					return "https://cf-media.sndcdn.com/test.mp3", nil
				},
			}

			extractor := audio.NewRealSoundCloudStreamExtractor(mockAPI)
			ctx := context.Background()

			streamInfo, err := extractor.ExtractStreamURL(ctx, tt.trackID)

			assert.Error(t, err)
			assert.Nil(t, streamInfo)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestRealStreamExtraction_HandleExpiredURLs(t *testing.T) {
	mockTrack := soundcloudapi.Track{
		ID: 111222333,
		PermalinkURL: "https://soundcloud.com/test/expired",
		Media: soundcloudapi.Media{
			Transcodings: []soundcloudapi.Transcoding{
				{
					Format: soundcloudapi.TranscodingFormat{Protocol: "progressive"},
				},
			},
		},
	}

	callCount := 0
	mockAPI := &MockRealSoundCloudAPI{
		GetTrackInfoFunc: func(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error) {
			return []soundcloudapi.Track{mockTrack}, nil
		},
		GetDownloadURLFunc: func(trackURL string, format string) (string, error) {
			callCount++
			if callCount == 1 {
				// First call returns expired URL (simulate URL that worked when created but expired)
				expiredTime := time.Now().Add(-1 * time.Hour).Unix()
				return fmt.Sprintf("https://cf-media.sndcdn.com/test.mp3?Policy=expired&Signature=abc&Key-Pair-Id=123&Expires=%d", expiredTime), nil
			}
			// Second call returns fresh URL
			return "https://cf-media.sndcdn.com/test.mp3?Policy=fresh&Signature=def&Key-Pair-Id=456", nil
		},
	}

	extractor := audio.NewRealSoundCloudStreamExtractor(mockAPI)
	ctx := context.Background()

	// First call should succeed
	streamInfo, err := extractor.ExtractStreamURL(ctx, 111222333)
	require.NoError(t, err)
	assert.Contains(t, streamInfo.URL, "Policy=expired")

	// URL validation should pass basic format checks (expiration would require HTTP calls)
	isValid, err := extractor.ValidateStreamURL(ctx, streamInfo.URL)
	assert.NoError(t, err)
	assert.True(t, isValid, "URL should pass basic validation checks")
}

func TestRealStreamExtraction_ContextCancellation(t *testing.T) {
	mockAPI := &MockRealSoundCloudAPI{
		GetTrackInfoFunc: func(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error) {
			// Simulate slow API response
			time.Sleep(100 * time.Millisecond)
			return nil, fmt.Errorf("should not reach here")
		},
	}

	extractor := audio.NewRealSoundCloudStreamExtractor(mockAPI)
	
	// Create context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	streamInfo, err := extractor.ExtractStreamURL(ctx, 123456789)

	assert.Error(t, err)
	assert.Nil(t, streamInfo)
	assert.Equal(t, context.Canceled, err)
}

func TestRealStreamExtraction_NoTranscodingsAvailable(t *testing.T) {
	mockTrack := soundcloudapi.Track{
		ID: 999888777,
		PermalinkURL: "https://soundcloud.com/test/no-transcodings",
		Media: soundcloudapi.Media{
			Transcodings: []soundcloudapi.Transcoding{}, // Empty transcodings
		},
	}

	mockAPI := &MockRealSoundCloudAPI{
		GetTrackInfoFunc: func(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error) {
			return []soundcloudapi.Track{mockTrack}, nil
		},
	}

	extractor := audio.NewRealSoundCloudStreamExtractor(mockAPI)
	ctx := context.Background()

	streamInfo, err := extractor.ExtractStreamURL(ctx, 999888777)

	assert.Error(t, err)
	assert.Nil(t, streamInfo)
	assert.Contains(t, err.Error(), "no transcodings available")
}

func TestRealStreamExtraction_URLValidation(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		shouldValid bool
		description string
	}{
		{
			name:        "valid CloudFront URL",
			url:         "https://cf-media.sndcdn.com/track.mp3?Policy=abc&Signature=def&Key-Pair-Id=ghi",
			shouldValid: true,
			description: "Should validate CloudFront URLs with proper auth parameters",
		},
		{
			name:        "valid HLS playlist URL",
			url:         "https://cf-media.sndcdn.com/playlist.m3u8?Policy=abc&Signature=def&Key-Pair-Id=ghi",
			shouldValid: true,
			description: "Should validate HLS playlist URLs",
		},
		{
			name:        "invalid domain",
			url:         "https://example.com/track.mp3",
			shouldValid: false,
			description: "Should reject URLs from non-SoundCloud domains",
		},
		{
			name:        "missing auth parameters",
			url:         "https://cf-media.sndcdn.com/track.mp3",
			shouldValid: false,
			description: "Should reject URLs without authentication parameters",
		},
		{
			name:        "malformed URL",
			url:         "not-a-url",
			shouldValid: false,
			description: "Should reject malformed URLs",
		},
	}

	mockAPI := &MockRealSoundCloudAPI{}
	extractor := audio.NewRealSoundCloudStreamExtractor(mockAPI)
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid, err := extractor.ValidateStreamURL(ctx, tt.url)
			
			require.NoError(t, err, "Validation should not return error for URL format checking")
			assert.Equal(t, tt.shouldValid, isValid, tt.description)
		})
	}
}

func TestRealStreamExtraction_QualitySelection(t *testing.T) {
	mockTrack := soundcloudapi.Track{
		ID: 123456789,
		PermalinkURL: "https://soundcloud.com/test/quality",
		Media: soundcloudapi.Media{
			Transcodings: []soundcloudapi.Transcoding{
				{
					Preset: "opus_0_0",
					Format: soundcloudapi.TranscodingFormat{
						Protocol: "hls",
						MimeType: "audio/ogg",
					},
				},
				{
					Preset: "mp3_1_0", // Standard quality MP3
					Format: soundcloudapi.TranscodingFormat{
						Protocol: "progressive",
						MimeType: "audio/mpeg",
					},
				},
				{
					Preset: "abr_sq", // Adaptive bitrate
					Format: soundcloudapi.TranscodingFormat{
						Protocol: "hls",
						MimeType: "audio/mpegurl",
					},
				},
			},
		},
	}

	mockAPI := &MockRealSoundCloudAPI{
		GetTrackInfoFunc: func(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error) {
			return []soundcloudapi.Track{mockTrack}, nil
		},
		GetDownloadURLFunc: func(trackURL string, format string) (string, error) {
			// Should select progressive format for better audio player compatibility
			assert.Equal(t, "progressive", format)
			return "https://cf-media.sndcdn.com/quality.mp3?auth=params", nil
		},
	}

	extractor := audio.NewRealSoundCloudStreamExtractor(mockAPI)
	ctx := context.Background()

	qualities, err := extractor.GetAvailableQualities(ctx, 123456789)

	require.NoError(t, err)
	assert.Contains(t, qualities, "progressive")
	assert.Contains(t, qualities, "hls")
	
	// Progressive should be preferred
	streamInfo, err := extractor.ExtractStreamURL(ctx, 123456789)
	require.NoError(t, err)
	assert.Equal(t, "progressive", streamInfo.Quality)
}