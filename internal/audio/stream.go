package audio

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	soundcloudapi "github.com/zackradisic/soundcloud-api"
)

// StreamInfo represents information about an audio stream
type StreamInfo struct {
	URL      string
	Format   string
	Quality  string
	Duration int64
}

// StreamExtractor defines the interface for extracting streaming URLs
type StreamExtractor interface {
	// ExtractStreamURL extracts the actual streaming URL from a track
	ExtractStreamURL(ctx context.Context, trackID int64) (*StreamInfo, error)
	
	// GetAvailableQualities returns available quality options for a track
	GetAvailableQualities(ctx context.Context, trackID int64) ([]string, error)
	
	// ValidateStreamURL checks if a streaming URL is still valid
	ValidateStreamURL(ctx context.Context, streamURL string) (bool, error)
}

// SoundCloudAPI interface for dependency injection and testing
type SoundCloudAPI interface {
	GetTrackInfo(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error)
}

// RealSoundCloudAPI interface includes methods for actual streaming URL extraction
type RealSoundCloudAPI interface {
	GetTrackInfoWithOptions(options soundcloudapi.GetTrackInfoOptions) ([]soundcloudapi.Track, error)
	GetDownloadURL(trackURL string, format string) (string, error)
}

// SoundCloudStreamExtractor implements StreamExtractor for SoundCloud
type SoundCloudStreamExtractor struct {
	api SoundCloudAPI
}

// NewSoundCloudStreamExtractor creates a new SoundCloud stream extractor
func NewSoundCloudStreamExtractor(clientID string) *SoundCloudStreamExtractor {
	var api *soundcloudapi.API
	var err error
	
	if clientID == "" {
		// Use default client (auto-fetches client ID)
		api, err = soundcloudapi.New(soundcloudapi.APIOptions{})
	} else {
		// Use provided client ID
		api, err = soundcloudapi.New(soundcloudapi.APIOptions{
			ClientID: clientID,
		})
	}
	
	if err != nil {
		// Return nil extractor if API creation fails
		return nil
	}
	
	return &SoundCloudStreamExtractor{
		api: api,
	}
}

// NewSoundCloudStreamExtractorWithAPI creates an extractor with a custom API (for testing)
func NewSoundCloudStreamExtractorWithAPI(api SoundCloudAPI) *SoundCloudStreamExtractor {
	return &SoundCloudStreamExtractor{
		api: api,
	}
}

// RealSoundCloudStreamExtractor implements StreamExtractor with actual API calls
type RealSoundCloudStreamExtractor struct {
	api RealSoundCloudAPI
}

// NewRealSoundCloudStreamExtractor creates a new real SoundCloud stream extractor
func NewRealSoundCloudStreamExtractor(api RealSoundCloudAPI) *RealSoundCloudStreamExtractor {
	return &RealSoundCloudStreamExtractor{
		api: api,
	}
}

// ExtractStreamURL extracts real streaming URLs from SoundCloud using GetDownloadURL
func (e *RealSoundCloudStreamExtractor) ExtractStreamURL(ctx context.Context, trackID int64) (*StreamInfo, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// Validate inputs
	if e.api == nil {
		return nil, fmt.Errorf("SoundCloud API client not initialized")
	}
	
	if trackID <= 0 {
		return nil, fmt.Errorf("invalid track ID: %d", trackID)
	}
	
	// Get track information to obtain permalink URL
	tracks, err := e.api.GetTrackInfoWithOptions(soundcloudapi.GetTrackInfoOptions{
		ID: []int64{trackID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get track info: %w", err)
	}
	
	if len(tracks) == 0 {
		return nil, fmt.Errorf("track not found: %d", trackID)
	}
	
	track := tracks[0]
	
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// Check if transcodings are available
	if len(track.Media.Transcodings) == 0 {
		return nil, fmt.Errorf("no transcodings available for track %d", trackID)
	}
	
	// Determine best format - prefer progressive over HLS for better audio player compatibility
	var preferredFormat string
	var selectedTranscoding *soundcloudapi.Transcoding
	
	// First pass: look for progressive format
	for _, transcoding := range track.Media.Transcodings {
		if transcoding.Format.Protocol == "progressive" {
			preferredFormat = "progressive"
			selectedTranscoding = &transcoding
			break
		}
	}
	
	// Second pass: fallback to HLS if no progressive available
	if selectedTranscoding == nil {
		for _, transcoding := range track.Media.Transcodings {
			if transcoding.Format.Protocol == "hls" {
				preferredFormat = "hls"
				selectedTranscoding = &transcoding
				break
			}
		}
	}
	
	if selectedTranscoding == nil {
		return nil, fmt.Errorf("no supported transcoding formats available for track %d", trackID)
	}
	
	// Get the actual download URL using the SoundCloud API
	streamURL, err := e.api.GetDownloadURL(track.PermalinkURL, preferredFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to get download URL: %w", err)
	}
	
	// Determine format from transcoding
	format := "mp3" // Default
	if selectedTranscoding.Format.MimeType == "audio/ogg" {
		format = "ogg"
	} else if selectedTranscoding.Format.Protocol == "hls" {
		format = "hls"
	}
	
	// Create StreamInfo
	streamInfo := &StreamInfo{
		URL:      streamURL,
		Format:   format,
		Quality:  preferredFormat,
		Duration: track.DurationMS,
	}
	
	return streamInfo, nil
}

// GetAvailableQualities returns available qualities for track using real API
func (e *RealSoundCloudStreamExtractor) GetAvailableQualities(ctx context.Context, trackID int64) ([]string, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// Validate inputs
	if e.api == nil {
		return nil, fmt.Errorf("SoundCloud API client not initialized")
	}
	
	if trackID <= 0 {
		return nil, fmt.Errorf("invalid track ID: %d", trackID)
	}
	
	// Get track information
	tracks, err := e.api.GetTrackInfoWithOptions(soundcloudapi.GetTrackInfoOptions{
		ID: []int64{trackID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get track info: %w", err)
	}
	
	if len(tracks) == 0 {
		return nil, fmt.Errorf("track not found: %d", trackID)
	}
	
	track := tracks[0]
	
	// Extract available qualities from transcodings
	qualityMap := make(map[string]bool)
	
	for _, transcoding := range track.Media.Transcodings {
		protocol := transcoding.Format.Protocol
		if protocol == "progressive" || protocol == "hls" {
			qualityMap[protocol] = true
		}
	}
	
	// Convert map to slice
	qualities := make([]string, 0, len(qualityMap))
	for quality := range qualityMap {
		qualities = append(qualities, quality)
	}
	
	// Ensure at least one quality is returned
	if len(qualities) == 0 {
		qualities = []string{"progressive", "hls"} // Default to both qualities
	}
	
	return qualities, nil
}

// ValidateStreamURL checks if stream URL is valid and not expired
func (e *RealSoundCloudStreamExtractor) ValidateStreamURL(ctx context.Context, streamURL string) (bool, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}
	
	// Validate input
	if streamURL == "" {
		return false, fmt.Errorf("stream URL cannot be empty")
	}
	
	// Parse URL to check if it's valid
	parsedURL, err := url.Parse(streamURL)
	if err != nil {
		return false, nil // Invalid URL format, but not an error
	}
	
	// Check if it's a valid HTTP/HTTPS URL
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false, nil // Invalid scheme
	}
	
	// Check if it's a SoundCloud CloudFront URL
	if !strings.Contains(parsedURL.Host, "sndcdn.com") {
		return false, nil // Not a SoundCloud URL
	}
	
	// Check for required authentication parameters
	queryParams := parsedURL.Query()
	if !queryParams.Has("Policy") || !queryParams.Has("Signature") || !queryParams.Has("Key-Pair-Id") {
		return false, nil // Missing authentication parameters
	}
	
	// URL passes basic validation
	return true, nil
}

// ExtractStreamURL extracts streaming URL from SoundCloud track
func (e *SoundCloudStreamExtractor) ExtractStreamURL(ctx context.Context, trackID int64) (*StreamInfo, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// Validate inputs
	if e.api == nil {
		return nil, fmt.Errorf("SoundCloud API client not initialized")
	}
	
	if trackID <= 0 {
		return nil, fmt.Errorf("invalid track ID: %d", trackID)
	}
	
	// Get track information
	tracks, err := e.api.GetTrackInfo(soundcloudapi.GetTrackInfoOptions{
		ID: []int64{trackID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get track info: %w", err)
	}
	
	if len(tracks) == 0 {
		return nil, fmt.Errorf("track not found: %d", trackID)
	}
	
	track := tracks[0]
	
	// Check for context cancellation again
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// Check if transcodings are available
	if len(track.Media.Transcodings) == 0 {
		return nil, fmt.Errorf("no transcodings available for track %d", trackID)
	}
	
	// For now, use a mock streaming URL since we need the full implementation
	// In a real implementation, we'd use the transcoding URL and call GetMediaURL
	streamURL := fmt.Sprintf("https://cf-media.sndcdn.com/track_%d.mp3", trackID)
	
	// Create StreamInfo
	streamInfo := &StreamInfo{
		URL:      streamURL,
		Format:   "mp3",  // Most SoundCloud tracks are MP3
		Quality:  "sq",   // Standard quality
		Duration: track.DurationMS,
	}
	
	return streamInfo, nil
}

// GetAvailableQualities returns available qualities for track
func (e *SoundCloudStreamExtractor) GetAvailableQualities(ctx context.Context, trackID int64) ([]string, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// Validate inputs
	if e.api == nil {
		return nil, fmt.Errorf("SoundCloud API client not initialized")
	}
	
	if trackID <= 0 {
		return nil, fmt.Errorf("invalid track ID: %d", trackID)
	}
	
	// Get track information
	tracks, err := e.api.GetTrackInfo(soundcloudapi.GetTrackInfoOptions{
		ID: []int64{trackID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get track info: %w", err)
	}
	
	if len(tracks) == 0 {
		return nil, fmt.Errorf("track not found: %d", trackID)
	}
	
	track := tracks[0]
	
	// Extract available qualities from transcodings
	// SoundCloud typically has different qualities based on transcoding format
	qualityMap := make(map[string]bool)
	
	for _, transcoding := range track.Media.Transcodings {
		// Check format to determine quality
		if strings.ToLower(transcoding.Format.Protocol) == "progressive" {
			qualityMap["sq"] = true  // Standard quality for progressive
		} else {
			qualityMap["hq"] = true  // High quality for HLS
		}
	}
	
	// Convert map to slice
	qualities := make([]string, 0, len(qualityMap))
	for quality := range qualityMap {
		qualities = append(qualities, quality)
	}
	
	// Ensure at least one quality is returned
	if len(qualities) == 0 {
		qualities = []string{"sq", "hq"} // Default to both qualities
	}
	
	return qualities, nil
}

// ValidateStreamURL checks if stream URL is valid
func (e *SoundCloudStreamExtractor) ValidateStreamURL(ctx context.Context, streamURL string) (bool, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}
	
	// Validate input
	if streamURL == "" {
		return false, fmt.Errorf("stream URL cannot be empty")
	}
	
	// Parse URL to check if it's valid
	parsedURL, err := url.Parse(streamURL)
	if err != nil {
		return false, fmt.Errorf("invalid URL format: %w", err)
	}
	
	// Check if it's a valid HTTP/HTTPS URL
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false, fmt.Errorf("invalid URL scheme: %s", parsedURL.Scheme)
	}
	
	// Check if it looks like a SoundCloud media URL
	if !strings.Contains(parsedURL.Host, "sndcdn.com") && 
	   !strings.Contains(parsedURL.Host, "soundcloud.com") {
		// Not a SoundCloud URL, but might still be valid
	}
	
	// Perform HEAD request to check if URL is accessible
	req, err := http.NewRequestWithContext(ctx, "HEAD", streamURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// URL is not accessible
		return false, nil
	}
	defer resp.Body.Close()
	
	// Consider 2xx status codes as valid
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}
	
	// URL exists but returned non-2xx status
	return false, nil
}