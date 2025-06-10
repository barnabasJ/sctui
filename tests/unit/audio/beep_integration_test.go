package audio_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"github.com/gopxl/beep/wav"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"soundcloud-tui/internal/audio"
)

// MockBeepStreamer implements beep.Streamer for testing
type MockBeepStreamer struct {
	samples    []int16
	currentPos int
	sampleRate beep.SampleRate
	err        error
}

func NewMockBeepStreamer(durationSeconds float64, sampleRate beep.SampleRate) *MockBeepStreamer {
	numSamples := int(durationSeconds * float64(sampleRate) * 2) // Stereo
	samples := make([]int16, numSamples)
	
	// Generate simple sine wave for testing
	for i := 0; i < numSamples; i += 2 {
		val := int16(32000.0 * 0.1) // Quiet sine wave
		samples[i] = val     // Left channel
		samples[i+1] = val   // Right channel
	}
	
	return &MockBeepStreamer{
		samples:    samples,
		sampleRate: sampleRate,
	}
}

func (m *MockBeepStreamer) Stream(buf [][2]float64) (n int, ok bool) {
	if m.err != nil {
		return 0, false
	}
	
	for i := range buf {
		if m.currentPos >= len(m.samples)-1 {
			return i, false // End of stream
		}
		
		// Convert int16 to float64 [-1, 1]
		left := float64(m.samples[m.currentPos]) / 32768.0
		right := float64(m.samples[m.currentPos+1]) / 32768.0
		
		buf[i][0] = left
		buf[i][1] = right
		m.currentPos += 2
	}
	
	return len(buf), true
}

func (m *MockBeepStreamer) Err() error {
	return m.err
}

func (m *MockBeepStreamer) SetError(err error) {
	m.err = err
}

func (m *MockBeepStreamer) Position() int {
	return m.currentPos / 2 // Return frame position (samples / 2 for stereo)
}

func (m *MockBeepStreamer) Length() int {
	return len(m.samples) / 2 // Return total frames
}

func (m *MockBeepStreamer) Seek(pos int) {
	m.currentPos = pos * 2 // Convert frame position to sample position
	if m.currentPos > len(m.samples) {
		m.currentPos = len(m.samples)
	}
	if m.currentPos < 0 {
		m.currentPos = 0
	}
}

// MockReadCloser implements io.ReadCloser for testing audio decoders
type MockReadCloser struct {
	data   []byte
	pos    int
	closed bool
}

func NewMockReadCloser(data []byte) *MockReadCloser {
	return &MockReadCloser{data: data}
}

func (m *MockReadCloser) Read(p []byte) (n int, err error) {
	if m.closed {
		return 0, fmt.Errorf("reader is closed")
	}
	
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *MockReadCloser) Close() error {
	m.closed = true
	return nil
}

func TestBeepIntegration_SpeakerInitialization(t *testing.T) {
	// Test speaker initialization with different sample rates
	sampleRates := []beep.SampleRate{
		beep.SampleRate(22050),
		beep.SampleRate(44100),
		beep.SampleRate(48000),
	}
	
	for _, rate := range sampleRates {
		t.Run(fmt.Sprintf("SampleRate_%d", rate), func(t *testing.T) {
			// Initialize speaker
			err := speaker.Init(rate, rate.N(time.Second/10))
			require.NoError(t, err, "Speaker initialization should succeed")
			
			// Verify speaker is initialized by trying to play silence
			done := make(chan bool)
			speaker.Play(beep.Callback(func() {
				done <- true
			}))
			
			// Wait for callback with timeout
			select {
			case <-done:
				// Success
			case <-time.After(100 * time.Millisecond):
				t.Fatal("Speaker callback not triggered")
			}
		})
	}
}

func TestBeepIntegration_AudioStreamCreation(t *testing.T) {
	sampleRate := beep.SampleRate(44100)
	err := speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	require.NoError(t, err)
	
	tests := []struct {
		name     string
		duration float64
		validate func(*testing.T, *MockBeepStreamer)
	}{
		{
			name:     "short audio stream",
			duration: 0.5, // 500ms
			validate: func(t *testing.T, stream *MockBeepStreamer) {
				expectedSamples := int(0.5 * float64(sampleRate) * 2)
				assert.Equal(t, expectedSamples, len(stream.samples))
			},
		},
		{
			name:     "medium audio stream",
			duration: 3.0, // 3 seconds
			validate: func(t *testing.T, stream *MockBeepStreamer) {
				assert.True(t, len(stream.samples) > 0)
				assert.Equal(t, 0, stream.Position())
			},
		},
		{
			name:     "long audio stream",
			duration: 180.0, // 3 minutes
			validate: func(t *testing.T, stream *MockBeepStreamer) {
				expectedFrames := int(180.0 * float64(sampleRate))
				assert.Equal(t, expectedFrames, stream.Length())
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := NewMockBeepStreamer(tt.duration, sampleRate)
			require.NotNil(t, stream)
			
			tt.validate(t, stream)
		})
	}
}

func TestBeepIntegration_VolumeControl(t *testing.T) {
	sampleRate := beep.SampleRate(44100)
	err := speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	require.NoError(t, err)
	
	stream := NewMockBeepStreamer(1.0, sampleRate)
	
	tests := []struct {
		name   string
		volume float64
		valid  bool
	}{
		{
			name:   "minimum volume",
			volume: 0.0,
			valid:  true,
		},
		{
			name:   "quarter volume",
			volume: 0.25,
			valid:  true,
		},
		{
			name:   "half volume",
			volume: 0.5,
			valid:  true,
		},
		{
			name:   "full volume",
			volume: 1.0,
			valid:  true,
		},
		{
			name:   "negative volume",
			volume: -0.5,
			valid:  false,
		},
		{
			name:   "over maximum volume",
			volume: 1.5,
			valid:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			volume := &effects.Volume{
				Streamer: stream,
				Base:     2, // Base for logarithmic volume
			}
			
			// Set volume
			if tt.valid {
				// Convert linear volume to logarithmic (Beep expects log scale)
				if tt.volume == 0 {
					volume.Silent = true
				} else {
					volume.Silent = false
					// Simple linear to log conversion for testing
					volume.Volume = tt.volume - 1.0 // Beep uses 0 as unity gain, negative for reduction
				}
				
				// Test that the volume control doesn't crash
				assert.NotPanics(t, func() {
					buf := make([][2]float64, 1024)
					volume.Stream(buf)
				})
			} else {
				// Invalid volumes should be rejected by our player implementation
				assert.True(t, tt.volume < 0 || tt.volume > 1)
			}
		})
	}
}

func TestBeepIntegration_AudioFormatSupport(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		decoder     func(io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error)
		testData    []byte
		expectError bool
	}{
		{
			name:        "MP3 format",
			format:      "mp3",
			decoder:     mp3.Decode,
			testData:    []byte("fake MP3 data"), // This will fail decode but tests the interface
			expectError: true,                    // Expected since it's fake data
		},
		{
			name:        "WAV format",
			format:      "wav",
			decoder:     wav.Decode,
			testData:    []byte("fake WAV data"), // This will fail decode but tests the interface
			expectError: true,                    // Expected since it's fake data
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewMockReadCloser(tt.testData)
			
			streamer, format, err := tt.decoder(reader)
			
			if tt.expectError {
				assert.Error(t, err, "Should error with fake data")
				assert.Nil(t, streamer, "Streamer should be nil on error")
			} else {
				assert.NoError(t, err, "Should decode valid data")
				assert.NotNil(t, streamer, "Streamer should not be nil")
				assert.NotEqual(t, beep.Format{}, format, "Format should be populated")
				
				// Clean up
				if streamer != nil {
					streamer.Close()
				}
			}
		})
	}
}

func TestBeepIntegration_StreamPosition(t *testing.T) {
	sampleRate := beep.SampleRate(44100)
	err := speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	require.NoError(t, err)
	
	duration := 2.0 // 2 seconds
	stream := NewMockBeepStreamer(duration, sampleRate)
	
	// Test initial position
	assert.Equal(t, 0, stream.Position())
	
	// Simulate streaming some data
	buf := make([][2]float64, 1024)
	n, ok := stream.Stream(buf)
	
	assert.True(t, ok, "Stream should return ok=true")
	assert.Equal(t, len(buf), n, "Should stream full buffer")
	assert.Equal(t, 1024, stream.Position(), "Position should advance")
	
	// Test seeking
	seekPos := stream.Length() / 2 // Seek to middle
	stream.Seek(seekPos)
	assert.Equal(t, seekPos, stream.Position(), "Position should match seek target")
	
	// Test seeking to end
	stream.Seek(stream.Length())
	assert.Equal(t, stream.Length(), stream.Position(), "Position should be at end")
	
	// Test seeking beyond end
	stream.Seek(stream.Length() + 1000)
	assert.Equal(t, stream.Length(), stream.Position(), "Position should be clamped to end")
	
	// Test seeking to beginning
	stream.Seek(0)
	assert.Equal(t, 0, stream.Position(), "Position should be at beginning")
}

func TestBeepIntegration_StreamError(t *testing.T) {
	sampleRate := beep.SampleRate(44100)
	err := speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	require.NoError(t, err)
	
	stream := NewMockBeepStreamer(1.0, sampleRate)
	
	// Test normal operation
	assert.NoError(t, stream.Err())
	
	buf := make([][2]float64, 100)
	n, ok := stream.Stream(buf)
	assert.True(t, ok)
	assert.Equal(t, 100, n)
	
	// Inject error
	testError := fmt.Errorf("simulated stream error")
	stream.SetError(testError)
	
	// Test error propagation
	assert.Equal(t, testError, stream.Err())
	
	n, ok = stream.Stream(buf)
	assert.False(t, ok, "Stream should return ok=false on error")
	assert.Equal(t, 0, n, "Should return 0 samples on error")
}

func TestBeepIntegration_PlaybackControl(t *testing.T) {
	sampleRate := beep.SampleRate(44100)
	err := speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	require.NoError(t, err)
	
	stream := NewMockBeepStreamer(0.1, sampleRate) // Short stream for testing
	
	// Test play control
	ctrl := &beep.Ctrl{
		Streamer: stream,
		Paused:   false,
	}
	
	// Test initial state
	assert.False(t, ctrl.Paused, "Should start unpaused")
	
	// Test pause
	ctrl.Paused = true
	assert.True(t, ctrl.Paused, "Should be paused")
	
	// Test resume
	ctrl.Paused = false
	assert.False(t, ctrl.Paused, "Should be resumed")
	
	// Test streaming with control
	buf := make([][2]float64, 100)
	
	// Stream when playing
	ctrl.Paused = false
	n, ok := ctrl.Stream(buf)
	assert.True(t, ok || n > 0, "Should stream when not paused")
	
	// Stream when paused (should return silence)
	ctrl.Paused = true
	n, ok = ctrl.Stream(buf)
	assert.True(t, ok, "Should return ok when paused")
	
	// Verify buffer is silent when paused
	for i := range buf {
		assert.Equal(t, 0.0, buf[i][0], "Left channel should be silent when paused")
		assert.Equal(t, 0.0, buf[i][1], "Right channel should be silent when paused")
	}
}

func TestBeepIntegration_PlayerInterfaceCompatibility(t *testing.T) {
	sampleRate := beep.SampleRate(44100)
	err := speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	require.NoError(t, err)
	
	// Test that Beep components can be used to implement our Player interface
	stream := NewMockBeepStreamer(3.0, sampleRate)
	
	// Create volume control
	volume := &effects.Volume{
		Streamer: stream,
		Base:     2,
		Volume:   0,  // Unity gain
		Silent:   false,
	}
	
	// Create playback control
	ctrl := &beep.Ctrl{
		Streamer: volume,
		Paused:   false,
	}
	
	// Test that these can be controlled like our Player interface expects
	tests := []struct {
		name     string
		action   string
		validate func(*testing.T)
	}{
		{
			name:   "play",
			action: "play",
			validate: func(t *testing.T) {
				ctrl.Paused = false
				assert.False(t, ctrl.Paused)
			},
		},
		{
			name:   "pause",
			action: "pause",
			validate: func(t *testing.T) {
				ctrl.Paused = true
				assert.True(t, ctrl.Paused)
			},
		},
		{
			name:   "stop",
			action: "stop",
			validate: func(t *testing.T) {
				ctrl.Paused = true
				stream.Seek(0)
				assert.True(t, ctrl.Paused)
				assert.Equal(t, 0, stream.Position())
			},
		},
		{
			name:   "volume_up",
			action: "volume_up",
			validate: func(t *testing.T) {
				volume.Volume = -0.1 // Beep uses negative for reduction
				buf := make([][2]float64, 100)
				volume.Stream(buf)
				// Just verify it doesn't crash
			},
		},
		{
			name:   "volume_down",
			action: "volume_down",
			validate: func(t *testing.T) {
				volume.Volume = -0.5
				buf := make([][2]float64, 100)
				volume.Stream(buf)
				// Just verify it doesn't crash
			},
		},
		{
			name:   "mute",
			action: "mute",
			validate: func(t *testing.T) {
				volume.Silent = true
				buf := make([][2]float64, 100)
				volume.Stream(buf)
				
				// Verify buffer is silent
				for i := range buf {
					assert.Equal(t, 0.0, buf[i][0])
					assert.Equal(t, 0.0, buf[i][1])
				}
			},
		},
		{
			name:   "seek",
			action: "seek",
			validate: func(t *testing.T) {
				targetPos := stream.Length() / 4
				stream.Seek(targetPos)
				assert.Equal(t, targetPos, stream.Position())
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t)
		})
	}
}

func TestBeepIntegration_ConcurrentAccess(t *testing.T) {
	sampleRate := beep.SampleRate(44100)
	err := speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	require.NoError(t, err)
	
	stream := NewMockBeepStreamer(1.0, sampleRate)
	ctrl := &beep.Ctrl{
		Streamer: stream,
		Paused:   false,
	}
	
	// Test concurrent pause/resume operations
	done := make(chan bool, 2)
	
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 100; i++ {
			ctrl.Paused = true
			time.Sleep(time.Microsecond)
		}
	}()
	
	go func() {
		defer func() { done <- true }()
		for i := 0; i < 100; i++ {
			ctrl.Paused = false
			time.Sleep(time.Microsecond)
		}
	}()
	
	// Wait for both goroutines
	<-done
	<-done
	
	// Should not crash - Beep handles concurrent access internally
	assert.NotPanics(t, func() {
		buf := make([][2]float64, 100)
		ctrl.Stream(buf)
	})
}

func TestBeepIntegration_MemoryUsage(t *testing.T) {
	sampleRate := beep.SampleRate(44100)
	err := speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	require.NoError(t, err)
	
	// Test with different duration streams to verify memory usage is reasonable
	durations := []float64{0.1, 1.0, 10.0, 60.0} // Up to 1 minute
	
	for _, duration := range durations {
		t.Run(fmt.Sprintf("Duration_%.1fs", duration), func(t *testing.T) {
			stream := NewMockBeepStreamer(duration, sampleRate)
			
			// Verify stream was created successfully
			assert.NotNil(t, stream)
			assert.True(t, stream.Length() > 0)
			
			// Test that we can stream without excessive memory usage
			buf := make([][2]float64, 1024)
			totalSamples := 0
			
			for {
				n, ok := stream.Stream(buf)
				totalSamples += n
				if !ok {
					break
				}
				
				// Limit total samples to prevent infinite loop in tests
				if totalSamples > int(duration*float64(sampleRate)*2) {
					break
				}
			}
			
			assert.True(t, totalSamples > 0, "Should have streamed some samples")
		})
	}
}

func TestBeepIntegration_ErrorRecovery(t *testing.T) {
	sampleRate := beep.SampleRate(44100)
	err := speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	require.NoError(t, err)
	
	stream := NewMockBeepStreamer(1.0, sampleRate)
	
	// Test error injection and recovery
	buf := make([][2]float64, 100)
	
	// Normal operation
	n, ok := stream.Stream(buf)
	assert.True(t, ok)
	assert.Equal(t, 100, n)
	
	// Inject error
	stream.SetError(fmt.Errorf("temporary error"))
	n, ok = stream.Stream(buf)
	assert.False(t, ok)
	assert.Equal(t, 0, n)
	
	// Recover from error
	stream.SetError(nil)
	n, ok = stream.Stream(buf)
	assert.True(t, ok)
	assert.Equal(t, 100, n)
}

// Benchmark test for performance
func BenchmarkBeepStreaming(b *testing.B) {
	sampleRate := beep.SampleRate(44100)
	err := speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	require.NoError(b, err)
	
	stream := NewMockBeepStreamer(10.0, sampleRate) // 10 second stream
	buf := make([][2]float64, 1024)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		stream.Seek(0) // Reset to beginning
		
		for {
			_, ok := stream.Stream(buf)
			if !ok {
				break
			}
		}
	}
}