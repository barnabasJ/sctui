package player

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"soundcloud-tui/internal/audio"
	"soundcloud-tui/internal/soundcloud"
	"soundcloud-tui/internal/ui/styles"
)

// State represents the current state of the player component
type State int

const (
	StateIdle State = iota
	StateLoading
	StatePlaying
	StatePaused
	StateCompleted
	StateError
)

// String returns the string representation of State
func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateLoading:
		return "loading"
	case StatePlaying:
		return "playing"
	case StatePaused:
		return "paused"
	case StateCompleted:
		return "completed"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// PlayTrackMsg represents a message to play a track
type PlayTrackMsg struct {
	Track *soundcloud.Track
}

// StreamInfoMsg represents stream info message
type StreamInfoMsg struct {
	StreamInfo *audio.StreamInfo
	Error      error
}

// ProgressUpdateMsg represents progress update message
type ProgressUpdateMsg struct {
	Position time.Duration
	Duration time.Duration
}

// PlayerComponent represents the player view component
type PlayerComponent struct {
	// Size
	width  int
	height int
	
	// State
	state           State
	currentTrack    *soundcloud.Track
	position        time.Duration
	duration        time.Duration
	expectedDuration time.Duration // Duration from SoundCloud metadata
	volume          float64
	error           error
	
	// Dependencies
	audioPlayer     audio.Player
	streamExtractor audio.StreamExtractor
}

// NewPlayerComponent creates a new player component
func NewPlayerComponent(audioPlayer audio.Player, streamExtractor audio.StreamExtractor) *PlayerComponent {
	return &PlayerComponent{
		width:           80,
		height:          20,
		state:           StateIdle,
		currentTrack:    nil,
		position:        0,
		duration:        0,
		volume:          1.0,
		error:           nil,
		audioPlayer:     audioPlayer,
		streamExtractor: streamExtractor,
	}
}

// Init initializes the player component
func (p *PlayerComponent) Init() tea.Cmd {
	return p.tickProgress()
}

// Update handles messages and updates the player component
func (p *PlayerComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return p.handleKeyMsg(msg)
		
	case PlayTrackMsg:
		return p.handlePlayTrack(msg)
		
	case StreamInfoMsg:
		return p.handleStreamInfo(msg)
		
	case ProgressUpdateMsg:
		p.position = msg.Position
		p.duration = msg.Duration
		// Sync state with audio player if available
		if p.audioPlayer != nil {
			p.syncStateWithAudioPlayer()
		}
		return p, p.tickProgress()
		
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		return p, nil
		
	case error:
		// Handle error messages from commands
		return p.handleError(msg)
		
	default:
		// No special handling needed - let ProgressUpdateMsg handle updates
	}
	
	return p, nil
}

// handleKeyMsg handles key messages
func (p *PlayerComponent) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if p.audioPlayer == nil {
		return p, nil
	}
	
	switch msg.Type {
	case tea.KeySpace:
		return p.togglePlayPause()
		
	case tea.KeyLeft:
		return p.seekBackward()
		
	case tea.KeyRight:
		return p.seekForward()
		
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "+", "=":
			return p.increaseVolume()
		case "-":
			return p.decreaseVolume()
		}
	}
	
	return p, nil
}

// handlePlayTrack handles play track message
func (p *PlayerComponent) handlePlayTrack(msg PlayTrackMsg) (tea.Model, tea.Cmd) {
	p.currentTrack = msg.Track
	p.state = StateLoading
	p.error = nil
	
	if p.streamExtractor == nil {
		p.state = StateError
		p.error = fmt.Errorf("no stream extractor available")
		return p, nil
	}
	
	return p, p.extractStreamURL(msg.Track.ID)
}

// handleStreamInfo handles stream info message
func (p *PlayerComponent) handleStreamInfo(msg StreamInfoMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		p.state = StateError
		p.error = msg.Error
		return p, nil
	}
	
	// Store expected duration from SoundCloud metadata
	if msg.StreamInfo != nil && msg.StreamInfo.Duration > 0 {
		p.expectedDuration = time.Duration(msg.StreamInfo.Duration) * time.Millisecond
	}
	
	p.state = StatePlaying
	return p, p.playStream(msg.StreamInfo.URL)
}

// togglePlayPause toggles between play and pause
func (p *PlayerComponent) togglePlayPause() (tea.Model, tea.Cmd) {
	if p.audioPlayer == nil {
		return p, nil
	}
	
	switch p.audioPlayer.GetState() {
	case audio.StatePlaying:
		return p, func() tea.Msg {
			err := p.audioPlayer.Pause()
			if err != nil {
				return fmt.Errorf("failed to pause: %w", err)
			}
			return ProgressUpdateMsg{
				Position: p.audioPlayer.GetPosition(),
				Duration: p.audioPlayer.GetDuration(),
			}
		}
	case audio.StatePaused:
		// Resume playback without restarting the stream
		return p, func() tea.Msg {
			err := p.audioPlayer.Resume()
			if err != nil {
				return fmt.Errorf("failed to resume: %w", err)
			}
			return ProgressUpdateMsg{
				Position: p.audioPlayer.GetPosition(),
				Duration: p.audioPlayer.GetDuration(),
			}
		}
	case audio.StateStopped:
		// Handle completed/stopped state - replay the track
		if p.currentTrack != nil {
			p.state = StateLoading
			return p, p.extractStreamURL(p.currentTrack.ID)
		}
		return p, nil
	default:
		return p, nil
	}
}

// seekBackward seeks backward by 10 seconds
func (p *PlayerComponent) seekBackward() (tea.Model, tea.Cmd) {
	if p.audioPlayer == nil {
		return p, nil
	}
	
	newPos := p.position - 10*time.Second
	if newPos < 0 {
		newPos = 0
	}
	
	return p, func() tea.Msg {
		err := p.audioPlayer.Seek(newPos)
		if err != nil {
			return fmt.Errorf("failed to seek: %w", err)
		}
		return ProgressUpdateMsg{
			Position: p.audioPlayer.GetPosition(),
			Duration: p.audioPlayer.GetDuration(),
		}
	}
}

// seekForward seeks forward by 10 seconds
func (p *PlayerComponent) seekForward() (tea.Model, tea.Cmd) {
	if p.audioPlayer == nil {
		return p, nil
	}
	
	newPos := p.position + 10*time.Second
	if newPos > p.duration {
		newPos = p.duration
	}
	
	return p, func() tea.Msg {
		err := p.audioPlayer.Seek(newPos)
		if err != nil {
			return fmt.Errorf("failed to seek: %w", err)
		}
		return ProgressUpdateMsg{
			Position: p.audioPlayer.GetPosition(),
			Duration: p.audioPlayer.GetDuration(),
		}
	}
}

// increaseVolume increases volume by 10%
func (p *PlayerComponent) increaseVolume() (tea.Model, tea.Cmd) {
	if p.audioPlayer == nil {
		// Even without audio player, update local volume for UI feedback
		p.volume = p.volume + 0.1
		if p.volume > 1.0 {
			p.volume = 1.0
		}
		return p, nil
	}
	
	newVolume := p.volume + 0.1
	if newVolume > 1.0 {
		newVolume = 1.0
	}
	
	return p, func() tea.Msg {
		err := p.audioPlayer.SetVolume(newVolume)
		if err != nil {
			return fmt.Errorf("failed to set volume: %w", err)
		}
		// Update local volume tracking
		p.volume = p.audioPlayer.GetVolume()
		return ProgressUpdateMsg{
			Position: p.audioPlayer.GetPosition(),
			Duration: p.audioPlayer.GetDuration(),
		}
	}
}

// decreaseVolume decreases volume by 10%
func (p *PlayerComponent) decreaseVolume() (tea.Model, tea.Cmd) {
	if p.audioPlayer == nil {
		// Even without audio player, update local volume for UI feedback
		p.volume = p.volume - 0.1
		if p.volume < 0.0 {
			p.volume = 0.0
		}
		return p, nil
	}
	
	newVolume := p.volume - 0.1
	if newVolume < 0.0 {
		newVolume = 0.0
	}
	
	return p, func() tea.Msg {
		err := p.audioPlayer.SetVolume(newVolume)
		if err != nil {
			return fmt.Errorf("failed to set volume: %w", err)
		}
		// Update local volume tracking
		p.volume = p.audioPlayer.GetVolume()
		return ProgressUpdateMsg{
			Position: p.audioPlayer.GetPosition(),
			Duration: p.audioPlayer.GetDuration(),
		}
	}
}

// extractStreamURL extracts the stream URL for a track
func (p *PlayerComponent) extractStreamURL(trackID int64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		streamInfo, err := p.streamExtractor.ExtractStreamURL(ctx, trackID)
		return StreamInfoMsg{
			StreamInfo: streamInfo,
			Error:      err,
		}
	}
}

// playStream starts playing a stream
func (p *PlayerComponent) playStream(streamURL string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		err := p.audioPlayer.Play(ctx, streamURL)
		if err != nil {
			return fmt.Errorf("failed to play stream: %w", err)
		}
		
		return ProgressUpdateMsg{
			Position: p.audioPlayer.GetPosition(),
			Duration: p.audioPlayer.GetDuration(),
		}
	}
}

// tickProgress returns a command that sends progress updates
func (p *PlayerComponent) tickProgress() tea.Cmd {
	// Use shorter interval for smoother progress updates
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		if p.audioPlayer != nil && (p.state == StatePlaying || p.state == StatePaused) {
			return ProgressUpdateMsg{
				Position: p.audioPlayer.GetPosition(),
				Duration: p.audioPlayer.GetDuration(),
			}
		}
		return nil
	})
}

// syncStateWithAudioPlayer synchronizes the UI state with the audio player state
func (p *PlayerComponent) syncStateWithAudioPlayer() {
	if p.audioPlayer == nil {
		return
	}

	audioState := p.audioPlayer.GetState()
	switch audioState {
	case audio.StatePlaying:
		if p.state != StatePlaying && p.state != StateLoading {
			p.state = StatePlaying
		}
	case audio.StatePaused:
		if p.state == StatePlaying {
			p.state = StatePaused
		}
	case audio.StateStopped:
		if p.state == StatePlaying || p.state == StatePaused {
			// If we have a current track, it completed successfully
			if p.currentTrack != nil {
				p.state = StateCompleted
			} else {
				// No track means we should be idle
				p.state = StateIdle
			}
		}
	}

	// Update volume to stay in sync
	p.volume = p.audioPlayer.GetVolume()
}

// handleError handles error messages and transitions to error state
func (p *PlayerComponent) handleError(err error) (tea.Model, tea.Cmd) {
	p.state = StateError
	p.error = err
	return p, nil
}

// View renders the player component
func (p *PlayerComponent) View() string {
	switch p.state {
	case StateIdle:
		return p.renderIdleView()
	case StateLoading:
		return p.renderLoadingView()
	case StatePlaying, StatePaused:
		return p.renderPlayingView()
	case StateCompleted:
		return p.renderCompletedView()
	case StateError:
		return p.renderErrorView()
	default:
		return "Unknown player state"
	}
}

// renderIdleView renders the idle view
func (p *PlayerComponent) renderIdleView() string {
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		styles.StatusStyle.Render("🎵 No track loaded"),
		"",
		styles.HelpStyle.Render("Select a track from the search to start playing"),
	)
	
	return styles.PlayerStyle.Width(p.width-4).Height(p.height-4).Render(
		lipgloss.Place(p.width-8, p.height-8, lipgloss.Center, lipgloss.Center, content),
	)
}

// renderLoadingView renders the loading view
func (p *PlayerComponent) renderLoadingView() string {
	if p.currentTrack == nil {
		return p.renderIdleView()
	}
	
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		styles.TrackTitleStyle.Render(p.currentTrack.Title),
		styles.TrackArtistStyle.Render(p.currentTrack.Artist()),
		"",
		styles.LoadingStatusStyle.Render("🔄 Loading..."),
	)
	
	return styles.PlayerStyle.Width(p.width-4).Height(p.height-4).Render(
		lipgloss.Place(p.width-8, p.height-8, lipgloss.Center, lipgloss.Center, content),
	)
}

// renderPlayingView renders the playing/paused view
func (p *PlayerComponent) renderPlayingView() string {
	if p.currentTrack == nil {
		return p.renderIdleView()
	}
	
	// Track info with enhanced metadata display
	metadata := styles.RenderMetadataPanel(
		p.currentTrack.Title,
		p.currentTrack.Artist(),
		p.width-8, // Account for player panel padding
	)
	
	// Status
	var status string
	if p.audioPlayer != nil {
		switch p.audioPlayer.GetState() {
		case audio.StatePlaying:
			status = styles.PlayingStatusStyle.Render("▶ Playing")
		case audio.StatePaused:
			status = styles.PausedStatusStyle.Render("⏸ Paused")
		default:
			status = styles.StatusStyle.Render("⏹ Stopped")
		}
	} else {
		status = styles.StatusStyle.Render("⏹ Stopped")
	}
	
	// Progress bar
	var progressBar string
	var timeInfo string
	
	// Use actual duration from audio player, fallback to expected duration from metadata
	displayDuration := p.duration
	if displayDuration <= 0 && p.expectedDuration > 0 {
		displayDuration = p.expectedDuration
	}
	
	if displayDuration > 0 {
		progress := float64(p.position) / float64(displayDuration)
		progressBar = styles.RenderProgressBar(p.width-12, progress)
		
		posStr := styles.FormatDurationFromTime(p.position)
		durStr := styles.FormatDurationFromTime(displayDuration)
		timeInfo = fmt.Sprintf("%s / %s", posStr, durStr)
	} else {
		progressBar = styles.RenderProgressBar(p.width-12, 0)
		timeInfo = styles.FormatDurationFromTime(0) + " / " + styles.FormatDurationFromTime(0)
	}
	
	// Volume info with appropriate icon
	volumePercent := int(p.volume * 100)
	var volumeIcon string
	switch {
	case p.volume == 0:
		volumeIcon = "🔇" // Muted
	case p.volume < 0.5:
		volumeIcon = "🔉" // Low volume
	default:
		volumeIcon = "🔊" // High volume
	}
	volumeInfo := fmt.Sprintf("%s %d%%", volumeIcon, volumePercent)
	
	// Controls help
	controls := styles.HelpStyle.Render("Space: Play/Pause • ←→: Seek • +/-: Volume")
	
	// Combine everything
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		metadata,
		"",
		status,
		"",
		progressBar,
		styles.StatusStyle.Render(timeInfo),
		"",
		styles.StatusStyle.Render(volumeInfo),
		"",
		controls,
	)
	
	return styles.PlayerStyle.Width(p.width-4).Render(content)
}

// renderCompletedView renders the completed view
func (p *PlayerComponent) renderCompletedView() string {
	if p.currentTrack == nil {
		return p.renderIdleView()
	}
	
	// Track info with enhanced metadata display
	metadata := styles.RenderMetadataPanel(
		p.currentTrack.Title,
		p.currentTrack.Artist(),
		p.width-8, // Account for player panel padding
	)
	
	// Status
	status := styles.StatusStyle.Render("✅ Track Completed")
	
	// Progress bar (show as full)
	var progressBar string
	var timeInfo string
	
	// Use actual duration from audio player, fallback to expected duration from metadata
	displayDuration := p.duration
	if displayDuration <= 0 && p.expectedDuration > 0 {
		displayDuration = p.expectedDuration
	}
	
	if displayDuration > 0 {
		progressBar = styles.RenderProgressBar(p.width-12, 1.0) // 100% complete
		
		durStr := styles.FormatDurationFromTime(displayDuration)
		timeInfo = fmt.Sprintf("%s / %s", durStr, durStr)
	} else {
		progressBar = styles.RenderProgressBar(p.width-12, 1.0)
		timeInfo = "Completed"
	}
	
	// Volume info
	volumePercent := int(p.volume * 100)
	var volumeIcon string
	switch {
	case p.volume == 0:
		volumeIcon = "🔇" // Muted
	case p.volume < 0.5:
		volumeIcon = "🔉" // Low volume
	default:
		volumeIcon = "🔊" // High volume
	}
	volumeInfo := fmt.Sprintf("%s %d%%", volumeIcon, volumePercent)
	
	// Controls help
	controls := styles.HelpStyle.Render("Space: Replay • Search for another track")
	
	// Combine everything
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		metadata,
		"",
		status,
		"",
		progressBar,
		styles.StatusStyle.Render(timeInfo),
		"",
		styles.StatusStyle.Render(volumeInfo),
		"",
		controls,
	)
	
	return styles.PlayerStyle.Width(p.width-4).Render(content)
}

// renderErrorView renders the error view
func (p *PlayerComponent) renderErrorView() string {
	var trackInfo string
	if p.currentTrack != nil {
		trackInfo = fmt.Sprintf("Track: %s - %s", p.currentTrack.Title, p.currentTrack.Artist())
	} else {
		trackInfo = "Unknown track"
	}
	
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		styles.ErrorStatusStyle.Render("❌ Playback Error"),
		"",
		styles.StatusStyle.Render(trackInfo),
		"",
		styles.ErrorStatusStyle.Render(p.error.Error()),
		"",
		styles.HelpStyle.Render("Try selecting another track"),
	)
	
	return styles.PlayerStyle.Width(p.width-4).Height(p.height-4).Render(
		lipgloss.Place(p.width-8, p.height-8, lipgloss.Center, lipgloss.Center, content),
	)
}

// formatDuration formats a duration to MM:SS format
func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// Getter and setter methods for testing and integration
func (p *PlayerComponent) GetCurrentTrack() *soundcloud.Track {
	return p.currentTrack
}

func (p *PlayerComponent) SetCurrentTrack(track *soundcloud.Track) {
	p.currentTrack = track
}

func (p *PlayerComponent) GetState() State {
	return p.state
}

func (p *PlayerComponent) SetState(state State) {
	p.state = state
}

func (p *PlayerComponent) GetVolume() float64 {
	if p.audioPlayer != nil {
		p.volume = p.audioPlayer.GetVolume()
	}
	return p.volume
}

func (p *PlayerComponent) GetPosition() time.Duration {
	return p.position
}

func (p *PlayerComponent) GetDuration() time.Duration {
	return p.duration
}

func (p *PlayerComponent) GetError() error {
	return p.error
}

func (p *PlayerComponent) SetSize(width, height int) {
	p.width = width
	p.height = height
}