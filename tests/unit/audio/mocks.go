package audio_test

import (
	soundcloudapi "github.com/zackradisic/soundcloud-api"
)

// MockSoundCloudAPI implements audio.SoundCloudAPI for testing
type MockSoundCloudAPI struct {
	GetTrackInfoFunc func(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error)
}

func (m *MockSoundCloudAPI) GetTrackInfo(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error) {
	if m.GetTrackInfoFunc != nil {
		return m.GetTrackInfoFunc(options)
	}
	
	// Default mock behavior - return a valid track
	return []soundcloudapi.Track{
		{
			ID:         options.ID[0], // Use the requested ID
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
					{
						URL: "https://api.soundcloud.com/tracks/123/stream_hls",
						Format: soundcloudapi.TranscodingFormat{
							Protocol: "hls",
							MimeType: "audio/mpeg",
						},
					},
				},
			},
		},
	}, nil
}