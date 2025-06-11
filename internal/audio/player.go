package audio

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"github.com/gopxl/beep/wav"
)

// PlayerState represents the current state of the audio player
type PlayerState int

const (
	StateStopped PlayerState = iota
	StatePlaying
	StatePaused
)

func (s PlayerState) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StatePlaying:
		return "playing"
	case StatePaused:
		return "paused"
	default:
		return "unknown"
	}
}

// Player defines the interface for audio playback
type Player interface {
	// Play starts or resumes playback from a streaming URL
	Play(ctx context.Context, streamURL string) error

	// Pause pauses the current playback
	Pause() error

	// Resume resumes paused playback
	Resume() error

	// Stop stops playback and resets position
	Stop() error

	// GetState returns the current player state
	GetState() PlayerState

	// GetPosition returns current playback position
	GetPosition() time.Duration

	// GetDuration returns total track duration
	GetDuration() time.Duration

	// SetVolume sets playback volume (0.0 to 1.0)
	SetVolume(volume float64) error

	// GetVolume returns current volume level
	GetVolume() float64

	// Seek sets playback position
	Seek(position time.Duration) error

	// Close releases player resources
	Close() error
}

// BeepPlayer implements Player using the Beep audio library
type BeepPlayer struct {
	mu              sync.RWMutex
	state           PlayerState
	volume          float64
	
	// Beep components
	streamer        beep.StreamSeekCloser
	format          beep.Format
	ctrl            *beep.Ctrl
	volumeCtrl      *effects.Volume
	
	// Speaker management
	speakerInit     sync.Once
	speakerInitErr  error
	
	// Stream information
	streamURL       string
	httpClient      *http.Client
}

// NewBeepPlayer creates a new Beep-based audio player
func NewBeepPlayer() *BeepPlayer {
	return &BeepPlayer{
		state:      StateStopped,
		volume:     1.0, // Default full volume
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Play starts or resumes playback from a streaming URL
func (p *BeepPlayer) Play(ctx context.Context, streamURL string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Validate input
	if streamURL == "" {
		return fmt.Errorf("stream URL cannot be empty")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Stop any existing playback
	if err := p.stopLocked(); err != nil {
		return fmt.Errorf("failed to stop existing playback: %w", err)
	}

	// Download and decode audio stream
	streamer, format, err := p.loadAudioStream(ctx, streamURL)
	if err != nil {
		return fmt.Errorf("failed to load audio stream: %w", err)
	}

	// Initialize speaker if needed
	p.speakerInit.Do(func() {
		p.speakerInitErr = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	})
	if p.speakerInitErr != nil {
		streamer.Close()
		return fmt.Errorf("failed to initialize speaker: %w", p.speakerInitErr)
	}

	// Set up audio pipeline
	p.streamer = streamer
	p.format = format
	p.streamURL = streamURL

	// Create volume control
	p.volumeCtrl = &effects.Volume{
		Streamer: p.streamer,
		Base:     2,
		Volume:   p.volumeToBeepVolume(p.volume),
		Silent:   p.volume == 0,
	}

	// Create playback control
	p.ctrl = &beep.Ctrl{
		Streamer: p.volumeCtrl,
		Paused:   false,
	}

	// Start playback
	done := make(chan bool)
	speaker.Play(beep.Seq(p.ctrl, beep.Callback(func() {
		p.mu.Lock()
		p.state = StateStopped
		p.mu.Unlock()
		done <- true
	})))

	p.state = StatePlaying
	
	// Start position tracking
	go p.trackPosition(done)

	return nil
}

// Pause pauses the current playback
func (p *BeepPlayer) Pause() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.state != StatePlaying {
		return fmt.Errorf("cannot pause: player is %s", p.state)
	}

	if p.ctrl != nil {
		speaker.Lock()
		p.ctrl.Paused = true
		speaker.Unlock()
	}

	p.state = StatePaused
	return nil
}

// Resume resumes paused playback
func (p *BeepPlayer) Resume() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.state != StatePaused {
		return fmt.Errorf("cannot resume: player is %s", p.state)
	}

	if p.ctrl != nil {
		speaker.Lock()
		p.ctrl.Paused = false
		speaker.Unlock()
	}

	p.state = StatePlaying
	return nil
}

// Stop stops playback and resets position
func (p *BeepPlayer) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stopLocked()
}

// GetState returns the current player state
func (p *BeepPlayer) GetState() PlayerState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// GetPosition returns current playback position
func (p *BeepPlayer) GetPosition() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.streamer == nil || p.format.SampleRate == 0 {
		return 0
	}
	
	speaker.Lock()
	position := p.streamer.Position()
	speaker.Unlock()
	
	return p.format.SampleRate.D(position)
}

// GetDuration returns total track duration
func (p *BeepPlayer) GetDuration() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.streamer == nil || p.format.SampleRate == 0 {
		return 0
	}
	
	return p.format.SampleRate.D(p.streamer.Len())
}

// SetVolume sets playback volume (0.0 to 1.0)
func (p *BeepPlayer) SetVolume(volume float64) error {
	if volume < 0.0 || volume > 1.0 {
		return fmt.Errorf("volume must be between 0.0 and 1.0, got %f", volume)
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.volume = volume
	
	if p.volumeCtrl != nil {
		speaker.Lock()
		p.volumeCtrl.Volume = p.volumeToBeepVolume(volume)
		p.volumeCtrl.Silent = volume == 0
		speaker.Unlock()
	}
	
	return nil
}

// GetVolume returns current volume level
func (p *BeepPlayer) GetVolume() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.volume
}

// Seek sets playback position
func (p *BeepPlayer) Seek(position time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if position < 0 {
		return fmt.Errorf("position cannot be negative")
	}

	if p.streamer == nil {
		return fmt.Errorf("no audio stream loaded")
	}
	
	duration := p.GetDuration()
	if position > duration {
		return fmt.Errorf("position %s exceeds duration %s", position, duration)
	}

	// Convert time position to sample position
	samplePos := p.format.SampleRate.N(position)
	
	speaker.Lock()
	err := p.streamer.Seek(samplePos)
	speaker.Unlock()
	
	return err
}

// Close releases player resources
func (p *BeepPlayer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if err := p.stopLocked(); err != nil {
		return err
	}
	
	if p.httpClient != nil {
		p.httpClient.CloseIdleConnections()
	}
	
	return nil
}

// Helper methods

// stopLocked stops playback without acquiring lock (caller must hold lock)
func (p *BeepPlayer) stopLocked() error {
	if p.ctrl != nil {
		speaker.Lock()
		p.ctrl.Paused = true
		speaker.Unlock()
	}
	
	if p.streamer != nil {
		if err := p.streamer.Close(); err != nil {
			return fmt.Errorf("failed to close streamer: %w", err)
		}
		p.streamer = nil
	}
	
	p.ctrl = nil
	p.volumeCtrl = nil
	p.streamURL = ""
	p.state = StateStopped
	
	return nil
}

// loadAudioStream downloads and decodes an audio stream from URL
func (p *BeepPlayer) loadAudioStream(ctx context.Context, streamURL string) (beep.StreamSeekCloser, beep.Format, error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", streamURL, nil)
	if err != nil {
		return nil, beep.Format{}, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Download stream
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, beep.Format{}, fmt.Errorf("failed to download stream: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, beep.Format{}, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}
	
	// Detect format and decode
	contentType := resp.Header.Get("Content-Type")
	streamURL = strings.ToLower(streamURL)
	
	var streamer beep.StreamSeekCloser
	var format beep.Format
	
	if strings.Contains(contentType, "audio/mpeg") || strings.Contains(streamURL, ".mp3") {
		streamer, format, err = mp3.Decode(resp.Body)
	} else if strings.Contains(contentType, "audio/wav") || strings.Contains(streamURL, ".wav") {
		streamer, format, err = wav.Decode(resp.Body)
	} else {
		// Default to MP3 for SoundCloud streams
		streamer, format, err = mp3.Decode(resp.Body)
	}
	
	if err != nil {
		resp.Body.Close()
		return nil, beep.Format{}, fmt.Errorf("failed to decode audio: %w", err)
	}
	
	return streamer, format, nil
}

// volumeToBeepVolume converts linear volume (0-1) to Beep's logarithmic volume
func (p *BeepPlayer) volumeToBeepVolume(linearVolume float64) float64 {
	if linearVolume <= 0 {
		return -10 // Very quiet
	}
	if linearVolume >= 1 {
		return 0 // Unity gain
	}
	
	// Convert linear to dB: 20 * log10(volume)
	// Beep uses base-2 logarithmic scale, so we adjust
	return (linearVolume - 1.0) * 2.0 // Simple approximation
}

// trackPosition runs in a goroutine to track playback position
func (p *BeepPlayer) trackPosition(done <-chan bool) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			// Position tracking is handled by GetPosition() which queries the streamer directly
			// This goroutine mainly exists for any future position-based logic
		}
	}
}

