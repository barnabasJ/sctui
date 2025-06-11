package app

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"soundcloud-tui/internal/audio"
	"soundcloud-tui/internal/soundcloud"
	"soundcloud-tui/internal/ui/components/player"
	"soundcloud-tui/internal/ui/components/search"
	"soundcloud-tui/internal/ui/styles"
)

// ViewType represents the different views in the application
type ViewType int

const (
	ViewSearch ViewType = iota
	ViewPlayer
	ViewQueue
)

// String returns the string representation of ViewType
func (v ViewType) String() string {
	switch v {
	case ViewSearch:
		return "search"
	case ViewPlayer:
		return "player"
	case ViewQueue:
		return "queue"
	default:
		return "unknown"
	}
}

// App represents the main application model
type App struct {
	// Window size
	width  int
	height int
	
	// Current view
	currentView ViewType
	quitting    bool
	
	// Components
	searchComponent *search.SearchComponent
	playerComponent *player.PlayerComponent
	
	// Dependencies
	soundCloudClient soundcloud.ClientInterface
	audioPlayer      audio.Player
	streamExtractor  audio.StreamExtractor
}


// NewApp creates a new application instance
func NewApp() *App {
	// Initialize SoundCloud client
	client, _ := soundcloud.NewClient()
	
	// Initialize audio player
	audioPlayer := audio.NewBeepPlayer()
	
	// Initialize real stream extractor with the SoundCloud client
	streamExtractor := audio.NewRealSoundCloudStreamExtractor(client)
	
	// Initialize components
	searchComponent := search.NewSearchComponent(client)
	playerComponent := player.NewPlayerComponent(audioPlayer, streamExtractor)
	
	return &App{
		width:            80,
		height:           24,
		currentView:      ViewSearch,
		quitting:         false,
		searchComponent:  searchComponent,
		playerComponent:  playerComponent,
		soundCloudClient: client,
		audioPlayer:      audioPlayer,
		streamExtractor:  streamExtractor,
	}
}

// Init initializes the application
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.searchComponent.Init(),
		a.playerComponent.Init(),
	)
}

// Update handles messages and updates the application state
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global key handling
		switch msg.Type {
		case tea.KeyCtrlC:
			a.quitting = true
			return a, tea.Quit
			
		case tea.KeyTab:
			a.nextView()
			return a, nil
			
		case tea.KeyShiftTab:
			a.previousView()
			return a, nil
			
		case tea.KeySpace:
			// Always pass space key to player component for play/pause
			updatedPlayer, playerCmd := a.playerComponent.Update(msg)
			a.playerComponent = updatedPlayer.(*player.PlayerComponent)
			if playerCmd != nil {
				cmds = append(cmds, playerCmd)
			}
			return a, tea.Batch(cmds...)
			
		case tea.KeyLeft, tea.KeyRight:
			// Always pass seek keys to player component
			updatedPlayer, playerCmd := a.playerComponent.Update(msg)
			a.playerComponent = updatedPlayer.(*player.PlayerComponent)
			if playerCmd != nil {
				cmds = append(cmds, playerCmd)
			}
			return a, tea.Batch(cmds...)
			
		case tea.KeyRunes:
			// Handle volume controls globally
			if len(msg.Runes) > 0 {
				switch string(msg.Runes) {
				case "+", "=", "-":
					// Always pass volume keys to player component
					updatedPlayer, playerCmd := a.playerComponent.Update(msg)
					a.playerComponent = updatedPlayer.(*player.PlayerComponent)
					if playerCmd != nil {
						cmds = append(cmds, playerCmd)
					}
					return a, tea.Batch(cmds...)
				}
			}
		}
		
		// Pass key messages to current view
		switch a.currentView {
		case ViewSearch:
			updatedSearch, cmd := a.searchComponent.Update(msg)
			a.searchComponent = updatedSearch.(*search.SearchComponent)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			
			// Handle track selection from search
			if selectedTrack := a.searchComponent.GetSelectedTrack(); selectedTrack != nil {
				// Don't clear selection immediately - wait for playback result
				playCmd := player.PlayTrackMsg{Track: selectedTrack}
				updatedPlayer, playerCmd := a.playerComponent.Update(playCmd)
				a.playerComponent = updatedPlayer.(*player.PlayerComponent)
				if playerCmd != nil {
					cmds = append(cmds, playerCmd)
				}
			}
			
		case ViewPlayer:
			updatedPlayer, cmd := a.playerComponent.Update(msg)
			a.playerComponent = updatedPlayer.(*player.PlayerComponent)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		
		// Update component sizes
		a.searchComponent.SetSize(msg.Width, msg.Height-4) // Reserve space for header/footer
		a.playerComponent.SetSize(msg.Width, msg.Height-4)
		
	case player.PlaybackStartedMsg:
		// Playback started successfully - reset search state
		a.searchComponent.ClearSelection()
		a.searchComponent.ResetToResults()
		// Switch to player view to show playback
		a.currentView = ViewPlayer
		return a, nil
		
	case player.PlaybackFailedMsg:
		// Playback failed - reset search state and show error
		a.searchComponent.ClearSelection()
		a.searchComponent.ResetToResults()
		// Stay in search view to let user try another track
		// The error will be shown in the player component
		return a, nil
		
	default:
		// Pass other messages to components
		updatedSearch, searchCmd := a.searchComponent.Update(msg)
		a.searchComponent = updatedSearch.(*search.SearchComponent)
		if searchCmd != nil {
			cmds = append(cmds, searchCmd)
		}
		
		updatedPlayer, playerCmd := a.playerComponent.Update(msg)
		a.playerComponent = updatedPlayer.(*player.PlayerComponent)
		if playerCmd != nil {
			cmds = append(cmds, playerCmd)
		}
	}
	
	return a, tea.Batch(cmds...)
}

// View renders the application
func (a *App) View() string {
	if a.quitting {
		return "Goodbye!\n"
	}
	
	// Build the view
	var view string
	
	// Header
	header := a.renderHeader()
	
	// Main content based on current view
	var content string
	switch a.currentView {
	case ViewSearch:
		content = a.searchComponent.View()
	case ViewPlayer:
		content = a.playerComponent.View()
	case ViewQueue:
		content = "Queue view - Coming soon!"
	}
	
	// Footer
	footer := a.renderFooter()
	
	// Combine all parts
	view = lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)
	
	return view
}

// renderHeader renders the application header
func (a *App) renderHeader() string {
	title := styles.TitleStyle.Render("SoundCloud TUI")
	
	// Navigation tabs
	tabs := []string{}
	for i, viewName := range []string{"Search", "Player", "Queue"} {
		if ViewType(i) == a.currentView {
			tabs = append(tabs, styles.ActiveTabStyle.Render(viewName))
		} else {
			tabs = append(tabs, styles.InactiveTabStyle.Render(viewName))
		}
	}
	
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	
	// Combine title and tabs
	header := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		tabBar,
	)
	
	return styles.HeaderStyle.Render(header)
}

// renderFooter renders the application footer
func (a *App) renderFooter() string {
	helpText := "Tab: Next View • Shift+Tab: Previous View • Ctrl+C: Quit"
	
	// Add global audio controls (work from any view)
	if a.playerComponent.GetCurrentTrack() != nil {
		helpText += " • Space: Play/Pause • ←→: Seek • +/-: Volume"
	}
	
	// Add view-specific help
	switch a.currentView {
	case ViewSearch:
		helpText += " • Enter: Search • ↑↓: Navigate • Enter: Select"
	case ViewPlayer:
		// Player-specific controls already shown above
		helpText += ""
	}
	
	return styles.FooterStyle.Render(helpText)
}

// nextView switches to the next view in the cycle
func (a *App) nextView() {
	switch a.currentView {
	case ViewSearch:
		a.currentView = ViewPlayer
	case ViewPlayer:
		a.currentView = ViewQueue
	case ViewQueue:
		a.currentView = ViewSearch
	}
}

// previousView switches to the previous view in the cycle
func (a *App) previousView() {
	switch a.currentView {
	case ViewSearch:
		a.currentView = ViewQueue
	case ViewPlayer:
		a.currentView = ViewSearch
	case ViewQueue:
		a.currentView = ViewPlayer
	}
}

// Getter methods for testing
func (a *App) GetCurrentView() ViewType {
	return a.currentView
}

func (a *App) SetCurrentView(view ViewType) {
	a.currentView = view
}

func (a *App) IsQuitting() bool {
	return a.quitting
}

func (a *App) GetSize() (int, int) {
	return a.width, a.height
}