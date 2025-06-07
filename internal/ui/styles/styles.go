package styles

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	PrimaryColor   = lipgloss.Color("#FF6B35")   // SoundCloud orange
	SecondaryColor = lipgloss.Color("#333333")   // Dark gray
	AccentColor    = lipgloss.Color("#00D4FF")   // Light blue
	TextColor      = lipgloss.Color("#FFFFFF")   // White
	MutedColor     = lipgloss.Color("#999999")   // Muted gray
	ErrorColor     = lipgloss.Color("#FF4444")   // Red
	SuccessColor   = lipgloss.Color("#44FF44")   // Green
	
	// Base styles
	BaseStyle = lipgloss.NewStyle().
			Foreground(TextColor).
			Background(lipgloss.Color("#000000"))
	
	// Title style
	TitleStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Background(lipgloss.Color("235")).
			Bold(true).
			Padding(0, 1).
			MarginBottom(1).
			Width(80).
			Align(lipgloss.Center)
	
	// Header style
	HeaderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(SecondaryColor).
			MarginBottom(1)
	
	// Footer style
	FooterStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(SecondaryColor).
			MarginTop(1).
			Padding(0, 1)
	
	// Tab styles
	ActiveTabStyle = lipgloss.NewStyle().
			Foreground(TextColor).
			Background(PrimaryColor).
			Bold(true).
			Padding(0, 2).
			MarginRight(1)
	
	InactiveTabStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Background(SecondaryColor).
			Padding(0, 2).
			MarginRight(1)
	
	// Input styles
	InputStyle = lipgloss.NewStyle().
			Foreground(TextColor).
			Background(lipgloss.Color("236")).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(AccentColor)
	
	InputFocusedStyle = InputStyle.Copy().
				BorderForeground(PrimaryColor).
				Bold(true)
	
	// List styles
	ListStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(SecondaryColor).
			Padding(1)
	
	ListItemStyle = lipgloss.NewStyle().
			Padding(0, 1).
			MarginBottom(0)
	
	SelectedListItemStyle = ListItemStyle.Copy().
				Foreground(TextColor).
				Background(PrimaryColor).
				Bold(true)
	
	// Player styles
	PlayerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(AccentColor).
			Padding(1).
			MarginBottom(1)
	
	TrackTitleStyle = lipgloss.NewStyle().
			Foreground(TextColor).
			Bold(true).
			MarginBottom(0)
	
	TrackArtistStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			MarginBottom(1)
	
	ProgressBarStyle = lipgloss.NewStyle().
			Foreground(AccentColor).
			Background(SecondaryColor)
	
	ProgressBarFillStyle = lipgloss.NewStyle().
				Foreground(TextColor).
				Background(PrimaryColor)
	
	// Control styles
	ControlStyle = lipgloss.NewStyle().
			Foreground(TextColor).
			Background(SecondaryColor).
			Padding(0, 1).
			MarginRight(1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(MutedColor)
	
	ControlActiveStyle = ControlStyle.Copy().
				Background(PrimaryColor).
				BorderForeground(PrimaryColor).
				Bold(true)
	
	// Status styles
	StatusStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Italic(true)
	
	PlayingStatusStyle = StatusStyle.Copy().
				Foreground(SuccessColor).
				Bold(true)
	
	PausedStatusStyle = StatusStyle.Copy().
				Foreground(AccentColor)
	
	ErrorStatusStyle = StatusStyle.Copy().
				Foreground(ErrorColor).
				Bold(true)
	
	LoadingStatusStyle = StatusStyle.Copy().
				Foreground(AccentColor).
				Bold(true)
	
	// Search styles
	SearchBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(AccentColor).
			Padding(1).
			MarginBottom(1)
	
	SearchResultsStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(SecondaryColor).
				Padding(1).
				Height(15) // Reserve space for results
	
	// Help styles
	HelpStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Italic(true).
			MarginTop(1)
)

// Helper functions for dynamic styling

// RenderProgressBar renders a progress bar with the given percentage
func RenderProgressBar(width int, progress float64) string {
	if width <= 0 {
		return ""
	}
	
	// Clamp progress between 0 and 1
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	
	fillWidth := int(float64(width) * progress)
	if fillWidth > width {
		fillWidth = width
	}
	
	// Use Unicode block characters for smoother progress bar
	filled := lipgloss.NewStyle().
		Background(PrimaryColor).
		Foreground(PrimaryColor).
		Render(strings.Repeat("█", fillWidth))
	
	empty := lipgloss.NewStyle().
		Background(SecondaryColor).
		Foreground(SecondaryColor).
		Render(strings.Repeat("█", width-fillWidth))
	
	return lipgloss.JoinHorizontal(lipgloss.Left, filled, empty)
}

// FormatDuration formats a duration in milliseconds to MM:SS format
func FormatDuration(durationMs int64) string {
	if durationMs <= 0 {
		return "0:00"
	}
	
	totalSeconds := durationMs / 1000
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	
	return lipgloss.NewStyle().
		Foreground(MutedColor).
		Render(formatTime(int(minutes), int(seconds)))
}

// FormatDurationFromTime formats a time.Duration to MM:SS format
func FormatDurationFromTime(d time.Duration) string {
	if d <= 0 {
		return "0:00"
	}
	
	totalSeconds := int(d.Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	
	return lipgloss.NewStyle().
		Foreground(MutedColor).
		Render(formatTime(minutes, seconds))
}

// formatTime is a helper to format minutes and seconds consistently
func formatTime(minutes, seconds int) string {
	if minutes >= 100 {
		// For very long tracks, show minutes without padding
		return lipgloss.NewStyle().Render(
			lipgloss.JoinHorizontal(lipgloss.Left,
				lipgloss.NewStyle().Render(fmt.Sprintf("%d", minutes)),
				":",
				lipgloss.NewStyle().Render(fmt.Sprintf("%02d", seconds)),
			))
	}
	
	return lipgloss.NewStyle().Render(
		lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Render(fmt.Sprintf("%d", minutes)),
			":",
			lipgloss.NewStyle().Render(fmt.Sprintf("%02d", seconds)),
		))
}

// TruncateText truncates text to fit within the specified width
func TruncateText(text string, width int) string {
	if len(text) <= width {
		return text
	}
	
	if width <= 3 {
		return "..."
	}
	
	return text[:width-3] + "..."
}

// RenderTrackTitle renders a track title with appropriate styling and truncation
func RenderTrackTitle(title string, maxWidth int) string {
	if title == "" {
		title = "Unknown Track"
	}
	
	truncated := TruncateText(title, maxWidth)
	return TrackTitleStyle.Render(truncated)
}

// RenderArtistName renders an artist name with appropriate styling and truncation
func RenderArtistName(artistName string, maxWidth int) string {
	if artistName == "" {
		artistName = "Unknown Artist"
	}
	
	truncated := TruncateText(artistName, maxWidth)
	return TrackArtistStyle.Render(truncated)
}

// RenderMetadataPanel renders a complete metadata panel for a track
func RenderMetadataPanel(title, artist string, width int) string {
	maxTitleWidth := width - 4 // Account for padding/borders
	maxArtistWidth := width - 4
	
	titleLine := RenderTrackTitle(title, maxTitleWidth)
	artistLine := RenderArtistName(artist, maxArtistWidth)
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleLine,
		artistLine,
	)
}