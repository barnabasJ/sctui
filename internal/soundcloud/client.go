package soundcloud

import (
	"fmt"

	soundcloudapi "github.com/zackradisic/soundcloud-api"
)

// Client wraps the SoundCloud API client
type Client struct {
	api *soundcloudapi.API
}

// Track represents a SoundCloud track
type Track struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Duration    int64  `json:"duration"` // Duration in milliseconds
	ArtworkURL  string `json:"artwork_url"`
	StreamURL   string `json:"stream_url"`
	PermalinkURL string `json:"permalink_url"`
	User        User   `json:"user"`
}

// User represents a SoundCloud user
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// FullName returns the combined first and last name
func (u User) FullName() string {
	if u.FirstName == "" && u.LastName == "" {
		return u.Username
	}
	if u.FirstName == "" {
		return u.LastName
	}
	if u.LastName == "" {
		return u.FirstName
	}
	return u.FirstName + " " + u.LastName
}

// NewClient creates a new SoundCloud client
func NewClient() (*Client, error) {
	api, err := soundcloudapi.New(soundcloudapi.APIOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create SoundCloud API client: %w", err)
	}

	return &Client{
		api: api,
	}, nil
}

// GetTrackInfo retrieves track information by URL
func (c *Client) GetTrackInfo(url string) (*Track, error) {
	tracks, err := c.api.GetTrackInfo(soundcloudapi.GetTrackInfoOptions{
		URL: url,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get track info: %w", err)
	}

	if len(tracks) == 0 {
		return nil, fmt.Errorf("no track found for URL: %s", url)
	}

	// Use the first track from the results
	track := tracks[0]

	// Convert to our Track struct
	return &Track{
		ID:          track.ID,
		Title:       track.Title,
		Description: track.Description,
		Duration:    track.DurationMS, // Use DurationMS field
		ArtworkURL:  track.ArtworkURL,
		PermalinkURL: track.PermalinkURL,
		User: User{
			ID:        track.User.ID,
			Username:  track.User.Username,
			FirstName: track.User.FirstName,
			LastName:  track.User.LastName,
		},
	}, nil
}

// Search searches for tracks on SoundCloud
func (c *Client) Search(query string) ([]Track, error) {
	paginatedQuery, err := c.api.Search(soundcloudapi.SearchOptions{
		Query: query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	tracks, err := paginatedQuery.GetTracks()
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks from search: %w", err)
	}

	// Convert to our Track structs
	result := make([]Track, len(tracks))
	for i, track := range tracks {
		result[i] = Track{
			ID:          track.ID,
			Title:       track.Title,
			Description: track.Description,
			Duration:    track.DurationMS, // Use DurationMS field
			ArtworkURL:  track.ArtworkURL,
			PermalinkURL: track.PermalinkURL,
			User: User{
				ID:        track.User.ID,
				Username:  track.User.Username,
				FirstName: track.User.FirstName,
				LastName:  track.User.LastName,
			},
		}
	}

	return result, nil
}