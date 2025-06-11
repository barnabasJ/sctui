package ui_test

import (
	"context"
	"time"

	"github.com/stretchr/testify/assert"

	"soundcloud-tui/internal/audio"
	"soundcloud-tui/internal/soundcloud"
)

// MockAudioPlayer implements audio.Player for testing
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

func (m *MockAudioPlayer) Resume() error {
	if m.state == audio.StatePaused {
		m.state = audio.StatePlaying
	}
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

// MockStreamExtractor implements audio.StreamExtractor for testing
type MockStreamExtractor struct {
	ExtractFunc func(ctx context.Context, trackID int64) (*audio.StreamInfo, error)
}

func (m *MockStreamExtractor) ExtractStreamURL(ctx context.Context, trackID int64) (*audio.StreamInfo, error) {
	if m.ExtractFunc != nil {
		return m.ExtractFunc(ctx, trackID)
	}
	return &audio.StreamInfo{
		URL:      "https://example.com/stream.mp3",
		Format:   "mp3",
		Quality:  "sq",
		Duration: 240000,
	}, nil
}

func (m *MockStreamExtractor) GetAvailableQualities(ctx context.Context, trackID int64) ([]string, error) {
	return []string{"sq", "hq"}, nil
}

func (m *MockStreamExtractor) ValidateStreamURL(ctx context.Context, streamURL string) (bool, error) {
	return true, nil
}

// MockSoundCloudClient implements soundcloud.ClientInterface for testing
type MockSoundCloudClient struct {
	SearchFunc func(query string) ([]soundcloud.Track, error)
}

func (m *MockSoundCloudClient) Search(query string) ([]soundcloud.Track, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(query)
	}
	return []soundcloud.Track{}, nil
}

func (m *MockSoundCloudClient) GetTrackInfo(url string) (*soundcloud.Track, error) {
	return &soundcloud.Track{
		ID:    123,
		Title: "Test Track",
		User:  soundcloud.User{Username: "Test Artist"},
	}, nil
}