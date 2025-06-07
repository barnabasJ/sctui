package ui_test

import (
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"soundcloud-tui/internal/soundcloud"
	"soundcloud-tui/internal/ui/components/search"
)

func TestSearchComponent_NewSearchComponent(t *testing.T) {
	component := search.NewSearchComponent(nil)
	
	require.NotNil(t, component)
	assert.Equal(t, "", component.GetQuery())
	assert.Empty(t, component.GetResults())
	assert.False(t, component.IsSearching())
	assert.Equal(t, search.StateInput, component.GetState())
}

func TestSearchComponent_InputHandling(t *testing.T) {
	component := search.NewSearchComponent(nil)
	
	// Test typing characters
	testInput := "test query"
	for _, char := range testInput {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedComponent, _ := component.Update(msg)
		component = updatedComponent.(*search.SearchComponent)
	}
	
	assert.Equal(t, testInput, component.GetQuery())
	assert.Equal(t, search.StateInput, component.GetState())
}

func TestSearchComponent_SearchExecution(t *testing.T) {
	// Mock SoundCloud client
	mockClient := &MockSoundCloudClient{
		SearchFunc: func(query string) ([]soundcloud.Track, error) {
			return []soundcloud.Track{
				{
					ID:       123456789,
					Title:    "Test Track",
					User:     soundcloud.User{Username: "Test Artist"},
					Duration: 240000,
				},
			}, nil
		},
	}
	
	component := search.NewSearchComponent(mockClient)
	
	// Type a query
	query := "test"
	for _, char := range query {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedComponent, _ := component.Update(msg)
		component = updatedComponent.(*search.SearchComponent)
	}
	
	// Press Enter to search
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedComponent, cmd := component.Update(enterMsg)
	component = updatedComponent.(*search.SearchComponent)
	
	assert.True(t, component.IsSearching())
	assert.Equal(t, search.StateSearching, component.GetState())
	assert.NotNil(t, cmd) // Should return search command
}

func TestSearchComponent_ResultsHandling(t *testing.T) {
	component := search.NewSearchComponent(nil)
	
	// Simulate search results
	results := []soundcloud.Track{
		{
			ID:       123,
			Title:    "Track 1",
			User:     soundcloud.User{Username: "Artist 1"},
		},
		{
			ID:       456,
			Title:    "Track 2", 
			User:     soundcloud.User{Username: "Artist 2"},
		},
	}
	
	// Send search results message
	resultsMsg := search.SearchResultsMsg{Results: results, Error: nil}
	updatedComponent, _ := component.Update(resultsMsg)
	component = updatedComponent.(*search.SearchComponent)
	
	assert.False(t, component.IsSearching())
	assert.Equal(t, search.StateResults, component.GetState())
	assert.Len(t, component.GetResults(), 2)
	assert.Equal(t, "Track 1", component.GetResults()[0].Title)
}

func TestSearchComponent_ResultNavigation(t *testing.T) {
	component := search.NewSearchComponent(nil)
	
	// Set up results
	results := []soundcloud.Track{
		{ID: 1, Title: "Track 1", User: soundcloud.User{Username: "Artist 1"}},
		{ID: 2, Title: "Track 2", User: soundcloud.User{Username: "Artist 2"}},
		{ID: 3, Title: "Track 3", User: soundcloud.User{Username: "Artist 3"}},
	}
	
	resultsMsg := search.SearchResultsMsg{Results: results, Error: nil}
	updatedComponent, _ := component.Update(resultsMsg)
	component = updatedComponent.(*search.SearchComponent)
	
	// Test navigation
	assert.Equal(t, 0, component.GetSelectedIndex())
	
	// Move down
	downMsg := tea.KeyMsg{Type: tea.KeyDown}
	updatedComponent, _ = component.Update(downMsg)
	component = updatedComponent.(*search.SearchComponent)
	assert.Equal(t, 1, component.GetSelectedIndex())
	
	// Move down again
	updatedComponent, _ = component.Update(downMsg)
	component = updatedComponent.(*search.SearchComponent)
	assert.Equal(t, 2, component.GetSelectedIndex())
	
	// Move down at end (should wrap or stay)
	updatedComponent, _ = component.Update(downMsg)
	component = updatedComponent.(*search.SearchComponent)
	assert.Equal(t, 2, component.GetSelectedIndex()) // Should stay at end
	
	// Move up
	upMsg := tea.KeyMsg{Type: tea.KeyUp}
	updatedComponent, _ = component.Update(upMsg)
	component = updatedComponent.(*search.SearchComponent)
	assert.Equal(t, 1, component.GetSelectedIndex())
}

func TestSearchComponent_TrackSelection(t *testing.T) {
	component := search.NewSearchComponent(nil)
	
	// Set up results
	results := []soundcloud.Track{
		{ID: 123, Title: "Selected Track", User: soundcloud.User{Username: "Test Artist"}},
	}
	
	resultsMsg := search.SearchResultsMsg{Results: results, Error: nil}
	updatedComponent, _ := component.Update(resultsMsg)
	component = updatedComponent.(*search.SearchComponent)
	
	// Press Enter to select track
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedComponent, cmd := component.Update(enterMsg)
	component = updatedComponent.(*search.SearchComponent)
	
	// Command can be nil since selection is tracked internally
	_ = cmd
	
	// Verify selected track
	selectedTrack := component.GetSelectedTrack()
	require.NotNil(t, selectedTrack)
	assert.Equal(t, int64(123), selectedTrack.ID)
	assert.Equal(t, "Selected Track", selectedTrack.Title)
}

func TestSearchComponent_ErrorHandling(t *testing.T) {
	component := search.NewSearchComponent(nil)
	
	// Simulate search error
	errorMsg := search.SearchResultsMsg{
		Results: nil,
		Error:   assert.AnError,
	}
	
	updatedComponent, _ := component.Update(errorMsg)
	component = updatedComponent.(*search.SearchComponent)
	
	assert.False(t, component.IsSearching())
	assert.Equal(t, search.StateError, component.GetState())
	assert.Empty(t, component.GetResults())
	assert.NotNil(t, component.GetError())
}

func TestSearchComponent_StateTransitions(t *testing.T) {
	component := search.NewSearchComponent(nil)
	
	// Initial state
	assert.Equal(t, search.StateInput, component.GetState())
	
	// Type a query first
	query := "test"
	for _, char := range query {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedComponent, _ := component.Update(msg)
		component = updatedComponent.(*search.SearchComponent)
	}
	
	// Start search
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedComponent, _ := component.Update(enterMsg)
	component = updatedComponent.(*search.SearchComponent)
	assert.Equal(t, search.StateSearching, component.GetState())
	
	// Receive results
	resultsMsg := search.SearchResultsMsg{
		Results: []soundcloud.Track{{ID: 1, Title: "Test", User: soundcloud.User{Username: "Test Artist"}}},
		Error:   nil,
	}
	updatedComponent, _ = component.Update(resultsMsg)
	component = updatedComponent.(*search.SearchComponent)
	assert.Equal(t, search.StateResults, component.GetState())
	
	// Clear search (Escape)
	escapeMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedComponent, _ = component.Update(escapeMsg)
	component = updatedComponent.(*search.SearchComponent)
	assert.Equal(t, search.StateInput, component.GetState())
	assert.Empty(t, component.GetResults())
}

func TestSearchComponent_ViewRendering(t *testing.T) {
	component := search.NewSearchComponent(nil)
	
	// Test view in input state
	view := component.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Search") // Should contain search prompt
	
	// Type a query and set searching state
	query := "test"
	for _, char := range query {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		updatedComponent, _ := component.Update(msg)
		component = updatedComponent.(*search.SearchComponent)
	}
	
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedComponent, _ := component.Update(enterMsg)
	component = updatedComponent.(*search.SearchComponent)
	
	view = component.View()
	assert.Contains(t, view, "Searching") // Should show searching indicator
	
	// Set results state
	resultsMsg := search.SearchResultsMsg{
		Results: []soundcloud.Track{{ID: 1, Title: "Test Track", User: soundcloud.User{Username: "Test Artist"}}},
		Error:   nil,
	}
	updatedComponent, _ = component.Update(resultsMsg)
	component = updatedComponent.(*search.SearchComponent)
	
	view = component.View()
	assert.Contains(t, view, "Test Track") // Should show results
}

// Mock SoundCloud client for testing
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
	return nil, nil
}