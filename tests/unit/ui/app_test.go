package ui_test

import (
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"soundcloud-tui/internal/ui/app"
)

func TestApp_NewApp(t *testing.T) {
	application := app.NewApp()
	
	require.NotNil(t, application)
	assert.Equal(t, app.ViewSearch, application.GetCurrentView())
	assert.False(t, application.IsQuitting())
}

func TestApp_Navigation(t *testing.T) {
	tests := []struct {
		name           string
		currentView    app.ViewType
		keyPressed     string
		expectedView   app.ViewType
		shouldChange   bool
	}{
		{
			name:         "tab from search to player",
			currentView:  app.ViewSearch,
			keyPressed:   "tab",
			expectedView: app.ViewPlayer,
			shouldChange: true,
		},
		{
			name:         "tab from player to queue",
			currentView:  app.ViewPlayer,
			keyPressed:   "tab",
			expectedView: app.ViewQueue,
			shouldChange: true,
		},
		{
			name:         "tab from queue to search",
			currentView:  app.ViewQueue,
			keyPressed:   "tab",
			expectedView: app.ViewSearch,
			shouldChange: true,
		},
		{
			name:         "shift+tab from search to queue",
			currentView:  app.ViewSearch,
			keyPressed:   "shift+tab",
			expectedView: app.ViewQueue,
			shouldChange: true,
		},
		{
			name:         "invalid key doesn't change view",
			currentView:  app.ViewSearch,
			keyPressed:   "x",
			expectedView: app.ViewSearch,
			shouldChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			application := app.NewApp()
			application.SetCurrentView(tt.currentView)
			
			initialView := application.GetCurrentView()
			assert.Equal(t, tt.currentView, initialView)
			
			// Simulate key press
			var msg tea.Msg
			switch tt.keyPressed {
			case "tab":
				msg = tea.KeyMsg{Type: tea.KeyTab}
			case "shift+tab":
				msg = tea.KeyMsg{Type: tea.KeyShiftTab}
			default:
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.keyPressed)}
			}
			
			updatedApp, _ := application.Update(msg)
			newApp := updatedApp.(*app.App)
			
			if tt.shouldChange {
				assert.Equal(t, tt.expectedView, newApp.GetCurrentView())
				assert.NotEqual(t, initialView, newApp.GetCurrentView())
			} else {
				assert.Equal(t, initialView, newApp.GetCurrentView())
			}
		})
	}
}

func TestApp_QuitHandling(t *testing.T) {
	application := app.NewApp()
	
	// Test Ctrl+C quits
	quitMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	updatedApp, cmd := application.Update(quitMsg)
	
	newApp := updatedApp.(*app.App)
	assert.True(t, newApp.IsQuitting())
	
	// Should return quit command
	assert.NotNil(t, cmd)
}

func TestApp_WindowSizeHandling(t *testing.T) {
	application := app.NewApp()
	
	// Test window size message
	sizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedApp, _ := application.Update(sizeMsg)
	
	newApp := updatedApp.(*app.App)
	width, height := newApp.GetSize()
	assert.Equal(t, 120, width)
	assert.Equal(t, 40, height)
}

func TestApp_ViewTypes(t *testing.T) {
	tests := []struct {
		viewType app.ViewType
		expected string
	}{
		{app.ViewSearch, "search"},
		{app.ViewPlayer, "player"},
		{app.ViewQueue, "queue"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.viewType.String())
		})
	}
}

func TestApp_BubbleTeaIntegration(t *testing.T) {
	application := app.NewApp()
	
	// Test that app implements tea.Model interface
	var _ tea.Model = application
	
	// Test Init returns expected command
	cmd := application.Init()
	assert.NotNil(t, cmd) // Should return some initialization command
	
	// Test View returns non-empty string
	view := application.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "SoundCloud TUI") // Should contain app title
}

func TestApp_StateManagement(t *testing.T) {
	application := app.NewApp()
	
	// Test initial state
	assert.Equal(t, app.ViewSearch, application.GetCurrentView())
	assert.False(t, application.IsQuitting())
	
	// Test state changes are persistent
	application.SetCurrentView(app.ViewPlayer)
	assert.Equal(t, app.ViewPlayer, application.GetCurrentView())
	
	// Test multiple view changes
	views := []app.ViewType{app.ViewQueue, app.ViewSearch, app.ViewPlayer}
	for _, view := range views {
		application.SetCurrentView(view)
		assert.Equal(t, view, application.GetCurrentView())
	}
}

func TestApp_ErrorHandling(t *testing.T) {
	application := app.NewApp()
	
	// Test with nil message
	updatedApp, cmd := application.Update(nil)
	assert.NotNil(t, updatedApp)
	assert.Nil(t, cmd) // Should handle gracefully
	
	// Test with unknown message type
	unknownMsg := "unknown message"
	updatedApp, cmd = application.Update(unknownMsg)
	assert.NotNil(t, updatedApp)
	// Should not crash and return the app
}