package audio

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"github.com/gopxl/beep/wav"
)

// BufferedStreamPlayer implements Player with advanced buffering and streaming capabilities
type BufferedStreamPlayer struct {
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
	
	// Buffer management
	buffer          *StreamBuffer
	bufferSize      int64
	preloadSize     int64
	
	// Error recovery and robustness
	retryCount      int
	maxRetries      int
	backoffDuration time.Duration
	reconnectDelay  time.Duration
	isRecovering    bool
	lastError       error
	
	// Position tracking
	positionTracker *PositionTracker
	
	// Callbacks
	onStateChange   func(PlayerState)
	onError         func(error)
}

// StreamBuffer manages progressive audio streaming with buffering
type StreamBuffer struct {
	mu           sync.RWMutex
	data         []byte
	size         int64
	readPos      int64
	writePos     int64
	preloaded    bool
	completed    bool
	minBuffer    int64
	ctx          context.Context
	cancel       context.CancelFunc
	downloadDone chan bool
}

// PositionTracker provides accurate position tracking
type PositionTracker struct {
	mu           sync.RWMutex
	startTime    time.Time
	pausedTime   time.Time
	totalPaused  time.Duration
	lastPosition time.Duration
	sampleRate   beep.SampleRate
}

// NewBufferedStreamPlayer creates a new buffered streaming audio player
func NewBufferedStreamPlayer() *BufferedStreamPlayer {
	return &BufferedStreamPlayer{
		state:           StateStopped,
		volume:          1.0,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  false,
			},
		},
		bufferSize:      4 * 1024 * 1024, // 4MB buffer for more robustness
		preloadSize:     1024 * 1024,     // 1MB preload for smoother start
		maxRetries:      5,               // More retry attempts
		backoffDuration: 1 * time.Second, // Faster initial retry
		reconnectDelay:  5 * time.Second, // Delay before reconnection attempts
		positionTracker: &PositionTracker{},
	}
}

// Play starts or resumes playback from a streaming URL with buffering
func (p *BufferedStreamPlayer) Play(ctx context.Context, streamURL string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Validate input
	if streamURL == "" {
		return fmt.Errorf("stream URL cannot be empty")
	}
	
	// Stop any existing playback
	if err := p.stopLocked(); err != nil {
		return fmt.Errorf("failed to stop existing playback: %w", err)
	}
	
	p.streamURL = streamURL
	p.retryCount = 0
	
	// Initialize stream buffer
	bufferCtx, bufferCancel := context.WithCancel(ctx)
	p.buffer = &StreamBuffer{
		data:         make([]byte, p.bufferSize),
		size:         p.bufferSize,
		minBuffer:    p.preloadSize,
		ctx:          bufferCtx,
		cancel:       bufferCancel,
		downloadDone: make(chan bool, 1),
	}
	
	// Start progressive download
	go p.downloadStream()
	
	// Wait for initial buffer to fill with shorter timeout
	preloadCtx, preloadCancel := context.WithTimeout(ctx, 5*time.Second)
	defer preloadCancel()
	
	if err := p.waitForPreload(preloadCtx); err != nil {
		bufferCancel()
		return fmt.Errorf("failed to preload audio data: %w", err)
	}
	
	// Create audio stream from buffer
	streamer, format, err := p.createStreamFromBuffer()
	if err != nil {
		bufferCancel()
		return fmt.Errorf("failed to create audio stream: %w", err)
	}
	
	// Initialize speaker if needed
	p.speakerInit.Do(func() {
		p.speakerInitErr = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	})
	if p.speakerInitErr != nil {
		streamer.Close()
		bufferCancel()
		return fmt.Errorf("failed to initialize speaker: %w", p.speakerInitErr)
	}
	
	// Set up audio pipeline
	p.streamer = streamer
	p.format = format
	
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
	
	// Start position tracking
	p.positionTracker.Start(format.SampleRate)
	
	// Start playback with callback
	done := make(chan bool)
	speaker.Play(beep.Seq(p.ctrl, beep.Callback(func() {
		p.mu.Lock()
		p.state = StateStopped
		p.positionTracker.Stop()
		if p.onStateChange != nil {
			go p.onStateChange(p.state)
		}
		p.mu.Unlock()
		done <- true
	})))
	
	p.state = StatePlaying
	
	if p.onStateChange != nil {
		go p.onStateChange(p.state)
	}
	
	// Start position tracking goroutine
	go p.trackPositionWithBuffer(done)
	
	return nil
}

// downloadStream downloads the audio stream progressively with retry logic
func (p *BufferedStreamPlayer) downloadStream() {
	defer func() {
		p.buffer.downloadDone <- true
	}()
	
	for attempt := 0; attempt < p.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry with exponential backoff
			delay := time.Duration(attempt) * p.backoffDuration
			select {
			case <-p.buffer.ctx.Done():
				return
			case <-time.After(delay):
			}
		}
		
		if p.downloadStreamAttempt() {
			return // Success
		}
		
		// If this was the last attempt, mark as failed
		if attempt == p.maxRetries-1 {
			p.mu.Lock()
			p.lastError = fmt.Errorf("failed to download stream after %d attempts", p.maxRetries)
			if p.onError != nil {
				go p.onError(p.lastError)
			}
			p.mu.Unlock()
		}
	}
}

// downloadStreamAttempt makes a single attempt to download the stream
func (p *BufferedStreamPlayer) downloadStreamAttempt() bool {
	req, err := http.NewRequestWithContext(p.buffer.ctx, "GET", p.streamURL, nil)
	if err != nil {
		return false
	}
	
	// Add range header if we're resuming from a previous position
	p.buffer.mu.RLock()
	resumeFrom := p.buffer.writePos
	p.buffer.mu.RUnlock()
	
	if resumeFrom > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumeFrom))
	}
	
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	// Accept both 200 (full content) and 206 (partial content)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return false
	}
	
	
	// Read data in chunks with improved error handling
	chunk := make([]byte, 32*1024) // 32KB chunks
	consecutiveErrors := 0
	maxConsecutiveErrors := 3
	
	for {
		select {
		case <-p.buffer.ctx.Done():
			return false // Context cancelled
		default:
		}
		
		n, err := resp.Body.Read(chunk)
		if n > 0 {
			p.buffer.write(chunk[:n])
			consecutiveErrors = 0 // Reset error count on successful read
		}
		
		if err == io.EOF {
			p.buffer.mu.Lock()
			p.buffer.completed = true
			p.buffer.mu.Unlock()
			return true // Success
		}
		
		if err != nil {
			consecutiveErrors++
			if consecutiveErrors >= maxConsecutiveErrors {
				return false // Too many consecutive errors
			}
			// Brief pause before continuing
			time.Sleep(100 * time.Millisecond)
			continue
		}
	}
}

// waitForPreload waits for the initial buffer to fill
func (p *BufferedStreamPlayer) waitForPreload(ctx context.Context) error {
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()
	
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("preload timeout")
		case <-ticker.C:
			if p.buffer.isPreloaded() {
				// BufferedStreamPlayer.waitForPreload: Preload completed")
				return nil
			}
		}
	}
}

// createStreamFromBuffer creates a beep stream from the buffered data
func (p *BufferedStreamPlayer) createStreamFromBuffer() (beep.StreamSeekCloser, beep.Format, error) {
	// Create a reader that reads from our buffer
	reader := NewBufferReader(p.buffer)
	
	// Try to decode as MP3 first, then WAV
	if streamer, format, err := mp3.Decode(reader); err == nil {
		// BufferedStreamPlayer.createStreamFromBuffer: Successfully decoded as MP3")
		return streamer, format, nil
	}
	
	// Reset reader position for WAV attempt
	reader.Reset()
	if streamer, format, err := wav.Decode(reader); err == nil {
		// BufferedStreamPlayer.createStreamFromBuffer: Successfully decoded as WAV")
		return streamer, format, nil
	}
	
	return nil, beep.Format{}, fmt.Errorf("unsupported audio format")
}

// Pause pauses the current playback
func (p *BufferedStreamPlayer) Pause() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.state != StatePlaying {
		return fmt.Errorf("cannot pause: player is %s", p.state)
	}
	
	if p.ctrl != nil {
		speaker.Lock()
		p.ctrl.Paused = true
		speaker.Unlock()
		p.positionTracker.Pause()
	}
	
	p.state = StatePaused
	if p.onStateChange != nil {
		go p.onStateChange(p.state)
	}
	
	return nil
}

// Resume resumes paused playback
func (p *BufferedStreamPlayer) Resume() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.state != StatePaused {
		return fmt.Errorf("cannot resume: player is %s", p.state)
	}
	
	if p.ctrl != nil {
		speaker.Lock()
		p.ctrl.Paused = false
		speaker.Unlock()
		p.positionTracker.Resume()
	}
	
	p.state = StatePlaying
	if p.onStateChange != nil {
		go p.onStateChange(p.state)
	}
	
	return nil
}

// Stop stops playback and resets position
func (p *BufferedStreamPlayer) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stopLocked()
}

// GetState returns the current player state
func (p *BufferedStreamPlayer) GetState() PlayerState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// GetPosition returns current playback position with enhanced accuracy
func (p *BufferedStreamPlayer) GetPosition() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.positionTracker != nil {
		return p.positionTracker.GetPosition()
	}
	
	if p.streamer == nil || p.format.SampleRate == 0 {
		return 0
	}
	
	speaker.Lock()
	position := p.streamer.Position()
	speaker.Unlock()
	
	return p.format.SampleRate.D(position)
}

// GetDuration returns total track duration
func (p *BufferedStreamPlayer) GetDuration() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.streamer == nil || p.format.SampleRate == 0 {
		return 0
	}
	
	return p.format.SampleRate.D(p.streamer.Len())
}

// SetVolume sets playback volume (0.0 to 1.0)
func (p *BufferedStreamPlayer) SetVolume(volume float64) error {
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
func (p *BufferedStreamPlayer) GetVolume() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.volume
}

// Seek sets playback position with buffer management
func (p *BufferedStreamPlayer) Seek(position time.Duration) error {
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
	
	if err == nil && p.positionTracker != nil {
		p.positionTracker.SetPosition(position)
	}
	
	return err
}

// Close releases player resources
func (p *BufferedStreamPlayer) Close() error {
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

// SetStateChangeCallback sets a callback for state changes
func (p *BufferedStreamPlayer) SetStateChangeCallback(callback func(PlayerState)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onStateChange = callback
}

// SetErrorCallback sets a callback for errors
func (p *BufferedStreamPlayer) SetErrorCallback(callback func(error)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onError = callback
}

// Helper methods

// stopLocked stops playback without acquiring lock (caller must hold lock)
func (p *BufferedStreamPlayer) stopLocked() error {
	if p.ctrl != nil {
		speaker.Lock()
		p.ctrl.Paused = true
		speaker.Unlock()
	}
	
	if p.buffer != nil && p.buffer.cancel != nil {
		p.buffer.cancel()
	}
	
	if p.streamer != nil {
		if err := p.streamer.Close(); err != nil {
			return fmt.Errorf("failed to close streamer: %w", err)
		}
		p.streamer = nil
	}
	
	if p.positionTracker != nil {
		p.positionTracker.Stop()
	}
	
	p.ctrl = nil
	p.volumeCtrl = nil
	p.streamURL = ""
	p.state = StateStopped
	
	if p.onStateChange != nil {
		go p.onStateChange(p.state)
	}
	
	return nil
}

// volumeToBeepVolume converts linear volume (0-1) to Beep's logarithmic volume
func (p *BufferedStreamPlayer) volumeToBeepVolume(linearVolume float64) float64 {
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

// trackPositionWithBuffer tracks position and manages buffer health
func (p *BufferedStreamPlayer) trackPositionWithBuffer(done <-chan bool) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	bufferHealthTicker := time.NewTicker(1 * time.Second)
	defer bufferHealthTicker.Stop()
	
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			// Update position tracking
			if p.positionTracker != nil {
				p.positionTracker.Update()
			}
			
		case <-bufferHealthTicker.C:
			// Check buffer health and attempt recovery if needed
			if p.buffer != nil {
				if !p.buffer.isHealthy() && !p.isRecovering {
					go p.attemptBufferRecovery()
				}
			}
		}
	}
}

// attemptBufferRecovery tries to recover from buffer underrun
func (p *BufferedStreamPlayer) attemptBufferRecovery() {
	p.mu.Lock()
	if p.isRecovering {
		p.mu.Unlock()
		return
	}
	p.isRecovering = true
	p.mu.Unlock()
	
	defer func() {
		p.mu.Lock()
		p.isRecovering = false
		p.mu.Unlock()
	}()
	
	// Pause playback temporarily
	if p.ctrl != nil {
		speaker.Lock()
		wasPlaying := !p.ctrl.Paused
		p.ctrl.Paused = true
		speaker.Unlock()
		
		// Wait for buffer to recover
		time.Sleep(p.reconnectDelay)
		
		// Check if buffer is healthier now
		if p.buffer != nil && p.buffer.isHealthy() && wasPlaying {
			speaker.Lock()
			p.ctrl.Paused = false
			speaker.Unlock()
		}
	}
}

// StreamBuffer methods

func (b *StreamBuffer) write(data []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Check if we have space
	availableSpace := b.size - b.writePos
	if availableSpace <= 0 {
		return // Buffer full
	}
	
	// Limit write to available space
	toWrite := data
	if int64(len(data)) > availableSpace {
		toWrite = data[:availableSpace]
	}
	
	// Copy data into buffer
	copy(b.data[b.writePos:], toWrite)
	b.writePos += int64(len(toWrite))
	
	// Mark as preloaded when we reach minimum buffer
	if !b.preloaded && b.writePos >= b.minBuffer {
		b.preloaded = true
	}
}

func (b *StreamBuffer) isPreloaded() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.preloaded
}

func (b *StreamBuffer) isHealthy() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	// Buffer is healthy if we have enough data ahead of read position
	available := b.writePos - b.readPos
	healthThreshold := b.minBuffer / 4 // More lenient threshold
	
	// Also consider if download is complete
	if b.completed {
		return available > 0 // Any data is good if download is done
	}
	
	return available > healthThreshold
}

// getBufferHealth returns buffer health metrics
func (b *StreamBuffer) getBufferHealth() (available, total int64, completed bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.writePos - b.readPos, b.size, b.completed
}

// PositionTracker methods

func (pt *PositionTracker) Start(sampleRate beep.SampleRate) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.startTime = time.Now()
	pt.sampleRate = sampleRate
	pt.totalPaused = 0
}

func (pt *PositionTracker) Stop() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.startTime = time.Time{}
}

func (pt *PositionTracker) Pause() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.pausedTime = time.Now()
}

func (pt *PositionTracker) Resume() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	if !pt.pausedTime.IsZero() {
		pt.totalPaused += time.Since(pt.pausedTime)
		pt.pausedTime = time.Time{}
	}
}

func (pt *PositionTracker) Update() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	
	if pt.startTime.IsZero() {
		return
	}
	
	elapsed := time.Since(pt.startTime) - pt.totalPaused
	if !pt.pausedTime.IsZero() {
		elapsed -= time.Since(pt.pausedTime)
	}
	
	pt.lastPosition = elapsed
}

func (pt *PositionTracker) GetPosition() time.Duration {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.lastPosition
}

func (pt *PositionTracker) SetPosition(position time.Duration) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.lastPosition = position
	pt.startTime = time.Now()
	pt.totalPaused = 0
	pt.pausedTime = time.Time{}
}