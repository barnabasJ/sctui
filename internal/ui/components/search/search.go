package search

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"soundcloud-tui/internal/soundcloud"
	"soundcloud-tui/internal/ui/styles"
)

// State represents the current state of the search component
type State int

const (
	StateInput State = iota
	StateSearching
	StateResults
	StateError
	StateTrackSelected // New state for when a track is selected
)

// String returns the string representation of State
func (s State) String() string {
	switch s {
	case StateInput:
		return "input"
	case StateSearching:
		return "searching"
	case StateResults:
		return "results"
	case StateError:
		return "error"
	case StateTrackSelected:
		return "track_selected"
	default:
		return "unknown"
	}
}

// SearchResultsMsg represents search results message
type SearchResultsMsg struct {
	Results []soundcloud.Track
	Error   error
}

// SearchComponent represents the search view component
type SearchComponent struct {
	// Size
	width  int
	height int
	
	// State
	state         State
	query         string
	results       []soundcloud.Track
	selectedIndex int
	selectedTrack *soundcloud.Track
	error         error
	
	// Dependencies
	client soundcloud.ClientInterface
}

// NewSearchComponent creates a new search component
func NewSearchComponent(client soundcloud.ClientInterface) *SearchComponent {
	return &SearchComponent{
		width:         80,
		height:        20,
		state:         StateInput,
		query:         "",
		results:       []soundcloud.Track{},
		selectedIndex: 0,
		selectedTrack: nil,
		error:         nil,
		client:        client,
	}
}

// Init initializes the search component
func (s *SearchComponent) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the search component
func (s *SearchComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch s.state {
		case StateInput:
			return s.handleInputState(msg)
		case StateResults:
			return s.handleResultsState(msg)
		case StateSearching:
			// Ignore input while searching
			return s, nil
		case StateError:
			// Allow escape to go back to input
			if msg.Type == tea.KeyEsc {
				s.state = StateInput
				s.error = nil
			}
			return s, nil
		}
		
	case SearchResultsMsg:
		return s.handleSearchResults(msg)
		
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
	}
	
	return s, nil
}

// handleInputState handles key messages in input state
func (s *SearchComponent) handleInputState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		if strings.TrimSpace(s.query) != "" {
			s.state = StateSearching
			return s, s.performSearch()
		}
		return s, nil
		
	case tea.KeyBackspace:
		if len(s.query) > 0 {
			s.query = s.query[:len(s.query)-1]
		}
		return s, nil
		
	case tea.KeyEsc:
		s.query = ""
		s.results = []soundcloud.Track{}
		s.selectedIndex = 0
		s.selectedTrack = nil
		s.error = nil
		return s, nil
		
	case tea.KeyRunes:
		s.query += string(msg.Runes)
		return s, nil
	}
	
	return s, nil
}

// handleResultsState handles key messages in results state
func (s *SearchComponent) handleResultsState(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if s.selectedIndex > 0 {
			s.selectedIndex--
		}
		return s, nil
		
	case tea.KeyDown:
		if s.selectedIndex < len(s.results)-1 {
			s.selectedIndex++
		}
		return s, nil
		
	case tea.KeyEnter:
		if s.selectedIndex < len(s.results) {
			s.selectedTrack = &s.results[s.selectedIndex]
			s.state = StateTrackSelected // Show loading feedback
			return s, nil
		}
		return s, nil
		
	case tea.KeyEsc:
		s.state = StateInput
		s.selectedIndex = 0
		s.selectedTrack = nil
		s.results = []soundcloud.Track{}
		return s, nil
	}
	
	return s, nil
}

// handleSearchResults handles search results message
func (s *SearchComponent) handleSearchResults(msg SearchResultsMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		s.state = StateError
		s.error = msg.Error
		s.results = []soundcloud.Track{}
	} else {
		s.state = StateResults
		s.results = msg.Results
		s.selectedIndex = 0
		s.error = nil
	}
	
	return s, nil
}

// performSearch performs the actual search
func (s *SearchComponent) performSearch() tea.Cmd {
	if s.client == nil {
		return func() tea.Msg {
			return SearchResultsMsg{
				Results: nil,
				Error:   fmt.Errorf("no SoundCloud client available"),
			}
		}
	}
	
	query := s.query
	return func() tea.Msg {
		results, err := s.client.Search(query)
		return SearchResultsMsg{
			Results: results,
			Error:   err,
		}
	}
}

// View renders the search component
func (s *SearchComponent) View() string {
	switch s.state {
	case StateInput:
		return s.renderInputView()
	case StateSearching:
		return s.renderSearchingView()
	case StateResults:
		return s.renderResultsView()
	case StateError:
		return s.renderErrorView()
	case StateTrackSelected:
		return s.renderTrackSelectedView()
	default:
		return "Unknown state"
	}
}

// renderInputView renders the input view
func (s *SearchComponent) renderInputView() string {
	// Search box
	prompt := "Search SoundCloud:"
	input := s.query + "█" // Cursor
	
	searchBox := styles.SearchBoxStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			prompt,
			styles.InputFocusedStyle.Width(s.width-6).Render(input),
		),
	)
	
	// Help text
	help := styles.HelpStyle.Render("Type to search, Enter to execute, Esc to clear")
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		searchBox,
		help,
	)
}

// renderSearchingView renders the searching view
func (s *SearchComponent) renderSearchingView() string {
	searchBox := styles.SearchBoxStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			"Searching: "+s.query,
			styles.LoadingStatusStyle.Render("🔍 Searching..."),
		),
	)
	
	return searchBox
}

// renderResultsView renders the results view
func (s *SearchComponent) renderResultsView() string {
	if len(s.results) == 0 {
		return styles.SearchResultsStyle.Render(
			styles.StatusStyle.Render("No results found for: " + s.query),
		)
	}
	
	// Header
	header := fmt.Sprintf("Search Results (%d found):", len(s.results))
	
	// Results list
	var resultItems []string
	visibleStart := 0
	visibleEnd := len(s.results)
	maxVisible := s.height - 8 // Reserve space for header, input, and help
	
	if len(s.results) > maxVisible {
		if s.selectedIndex >= maxVisible/2 {
			visibleStart = s.selectedIndex - maxVisible/2
			visibleEnd = visibleStart + maxVisible
			if visibleEnd > len(s.results) {
				visibleEnd = len(s.results)
				visibleStart = visibleEnd - maxVisible
			}
		} else {
			visibleEnd = maxVisible
		}
	}
	
	for i := visibleStart; i < visibleEnd; i++ {
		track := s.results[i]
		item := fmt.Sprintf("%-50s %s (%s)",
			styles.TruncateText(track.Title, 50),
			track.Artist(),
			track.DurationString(),
		)
		
		if i == s.selectedIndex {
			resultItems = append(resultItems, styles.SelectedListItemStyle.Render("▶ "+item))
		} else {
			resultItems = append(resultItems, styles.ListItemStyle.Render("  "+item))
		}
	}
	
	// Scroll indicator
	var scrollIndicator string
	if len(s.results) > maxVisible {
		scrollIndicator = fmt.Sprintf(" [%d-%d of %d]", visibleStart+1, visibleEnd, len(s.results))
	}
	
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		styles.TrackTitleStyle.Render(header+scrollIndicator),
		"",
		lipgloss.JoinVertical(lipgloss.Left, resultItems...),
	)
	
	help := styles.HelpStyle.Render("↑↓: Navigate • Enter: Select • Esc: Back to search")
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		styles.SearchResultsStyle.Render(content),
		help,
	)
}

// renderErrorView renders the error view
func (s *SearchComponent) renderErrorView() string {
	errorBox := styles.SearchBoxStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			styles.ErrorStatusStyle.Render("❌ Search Error"),
			"",
			styles.ErrorStatusStyle.Render(s.error.Error()),
		),
	)
	
	help := styles.HelpStyle.Render("Esc: Back to search")
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		errorBox,
		help,
	)
}

// renderTrackSelectedView renders the track selected/loading view
func (s *SearchComponent) renderTrackSelectedView() string {
	if s.selectedTrack == nil {
		return s.renderResultsView() // Fallback to results if no track selected
	}
	
	loadingBox := styles.SearchBoxStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			styles.TrackTitleStyle.Render("Loading Track..."),
			"",
			styles.LoadingStatusStyle.Render("🎵 "+s.selectedTrack.Title),
			styles.TrackArtistStyle.Render("by "+s.selectedTrack.Artist()),
			"",
			styles.LoadingStatusStyle.Render("⏳ Fetching stream URL..."),
		),
	)
	
	return loadingBox
}

// Getter methods for testing and integration
func (s *SearchComponent) GetQuery() string {
	return s.query
}

func (s *SearchComponent) GetResults() []soundcloud.Track {
	return s.results
}

func (s *SearchComponent) IsSearching() bool {
	return s.state == StateSearching
}

func (s *SearchComponent) GetState() State {
	return s.state
}

func (s *SearchComponent) GetSelectedIndex() int {
	return s.selectedIndex
}

func (s *SearchComponent) GetSelectedTrack() *soundcloud.Track {
	return s.selectedTrack
}

func (s *SearchComponent) GetError() error {
	return s.error
}

func (s *SearchComponent) ClearSelection() {
	s.selectedTrack = nil
}

func (s *SearchComponent) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// ResetToResults resets the component back to showing results after track selection
func (s *SearchComponent) ResetToResults() {
	if s.state == StateTrackSelected {
		s.state = StateResults
		s.selectedTrack = nil
	}
}