# SoundCloud TUI with Go SoundCloud API - Implementation Plan

## Problem Statement
Build a SoundCloud TUI client using the `github.com/zackradisic/soundcloud-api` Go library, which provides access to SoundCloud's undocumented API v2 without requiring official API credentials.

## Solution Overview
Use the reverse-engineered SoundCloud API library to build a legitimate TUI client with:
- Search functionality for tracks, artists, and playlists
- Track metadata retrieval and display
- Audio streaming with progress controls
- Playlist management
- Modern TUI interface with Bubble Tea

## Legal Considerations
- ⚠️ Uses SoundCloud's undocumented internal API
- ⚠️ May violate SoundCloud's Terms of Service
- ✅ No official API credentials required
- ✅ Open source educational implementation
- 📝 Users assume responsibility for ToS compliance

## Implementation Plan

### Step 1: Setup and Basic Integration ✅ COMPLETED
- [x] Add soundcloud-api dependency
- [x] Create SoundCloud client wrapper
- [x] Test basic track info retrieval
- [x] Verify search functionality works
- [x] Build working CLI with search and track info commands
- [x] Add comprehensive ToS disclaimer
- [x] Test real SoundCloud API integration

**Status**: ✅ Working CLI can search tracks and display metadata

**Current Functionality:**
```bash
# Search for tracks
./sctui -search "lofi hip hop"

# Get track information  
./sctui -track "https://soundcloud.com/artist/track"

# Show help
./sctui -help
```

**What Works:**
- SoundCloud API integration without official credentials
- Track search with formatted results (title, artist, duration, URL)
- Track metadata retrieval from URLs
- Comprehensive ToS disclaimer for users
- Error handling and user-friendly output

### Step 2: Audio Streaming Foundation (TDD Approach)
- [ ] Write tests for streaming URL extraction interface
- [ ] Research streaming URL extraction from SoundCloud tracks
- [ ] Implement URL extraction to pass tests
- [ ] Write tests for audio player interface
- [ ] Add Beep audio library dependency
- [ ] Implement basic audio playback to pass tests
- [ ] Write integration tests for streaming workflow
- [ ] Test with actual SoundCloud tracks and refine
- [ ] Write tests for CLI play/pause/stop commands
- [ ] Add play/pause/stop controls via CLI flags

### Step 3: Core TUI Framework (TDD Approach)
- [ ] Write tests for TUI component interfaces
- [ ] Add Bubble Tea and Lipgloss dependencies
- [ ] Write tests for main application model updates
- [ ] Design and implement main application layout
- [ ] Write tests for search interface component
- [ ] Implement search interface component to pass tests
- [ ] Write tests for track listing component with navigation
- [ ] Create track listing component to pass tests
- [ ] Write tests for keyboard shortcut handling
- [ ] Add basic vim-style keyboard shortcuts to pass tests

### Step 4: Interactive Player (TDD Approach)
- [ ] Write tests for audio integration with TUI
- [ ] Integrate audio streaming into TUI to pass tests
- [ ] Write tests for progress bar and time display
- [ ] Implement progress bar and time display to pass tests
- [ ] Write tests for volume controls
- [ ] Implement volume controls to pass tests
- [ ] Write tests for track metadata display panel
- [ ] Add track metadata display panel to pass tests
- [ ] Write tests for audio state management
- [ ] Handle audio state management to pass tests

### Step 5: Enhanced Features (TDD Approach)
- [ ] Write tests for track queueing system
- [ ] Implement track queueing system to pass tests
- [ ] Write tests for playlist support and management
- [ ] Implement playlist support to pass tests
- [ ] Write tests for search result pagination
- [ ] Implement search result pagination to pass tests
- [ ] Write tests for keyboard shortcuts reference
- [ ] Add keyboard shortcuts reference to pass tests
- [ ] Write tests for configuration file support
- [ ] Implement configuration file support to pass tests

### Step 6: Polish and Integration Testing
- [ ] Write comprehensive integration tests
- [ ] Add end-to-end testing scenarios
- [ ] Write performance tests for audio streaming
- [ ] Performance optimization based on test results
- [ ] Write tests for error handling edge cases
- [ ] Improve error handling to pass tests
- [ ] User experience testing and improvements
- [ ] Documentation and usage guide with examples

## Technical Architecture

### Dependencies
```go
require (
    github.com/zackradisic/soundcloud-api v0.1.0
    github.com/charmbracelet/bubbletea v1.0.0
    github.com/charmbracelet/lipgloss v0.12.0
    github.com/gopxl/beep v1.4.0
    github.com/spf13/cobra v1.8.0
)
```

### Project Structure
```
cmd/sctui/
├── main.go                 # CLI entry point
internal/
├── soundcloud/
│   ├── client.go          # SoundCloud API wrapper
│   ├── models.go          # Data structures
│   └── stream.go          # Audio streaming
├── ui/
│   ├── app.go             # Main app model
│   ├── search.go          # Search component
│   ├── player.go          # Player controls
│   └── styles.go          # UI styling
└── audio/
    ├── player.go          # Audio playback
    └── queue.go           # Track queue
```

### Core Components

#### SoundCloud Client Wrapper
```go
type Client struct {
    api *soundcloudapi.API
}

func (c *Client) Search(query string) ([]Track, error)
func (c *Client) GetTrack(url string) (*Track, error)
func (c *Client) GetStreamURL(track *Track) (string, error)
```

#### Main TUI Application
```go
type App struct {
    client    *soundcloud.Client
    player    *audio.Player
    search    search.Model
    playlist  playlist.Model
    currentTrack *soundcloud.Track
}
```

#### Audio Player Integration
```go
type Player struct {
    speaker   *beep.Buffer
    ctrl      *beep.Ctrl
    volume    *effects.Volume
    streamer  beep.StreamCloser
}
```

## Features Roadmap

### Phase 1: MVP ✅ COMPLETED
- [x] Basic SoundCloud API integration
- [x] Search tracks by keyword
- [x] Display track information
- [x] CLI interface with help and disclaimers
- [ ] Play/pause audio controls (Next: Phase 2)

### Phase 2: Audio Streaming & TUI (TDD)
- [ ] Write tests for audio streaming interfaces
- [ ] Audio playback from streaming URLs (test-driven)
- [ ] Write tests for TUI components
- [ ] Basic TUI interface with Bubble Tea (test-driven)
- [ ] Write tests for player controls
- [ ] Play/pause/stop controls (test-driven)
- [ ] Progress bar and time display with tests
- [ ] Track metadata in TUI with component tests

### Phase 3: Enhanced Experience (TDD)
- [ ] Test-driven playlist creation and management
- [ ] Test-driven track queueing and autoplay
- [ ] Progress seeking with comprehensive tests
- [ ] Volume control with interface tests
- [ ] Advanced keyboard shortcuts with behavior tests

### Phase 4: Advanced Features
- [ ] User profiles and likes
- [ ] Download functionality (where permitted)
- [ ] Recommendations
- [ ] Scrobbling integration
- [ ] Configuration management

## Success Criteria
- Can search and play SoundCloud tracks
- Responsive TUI with intuitive controls
- Stable audio playback without interruptions
- Handles network errors gracefully
- Clear documentation for users

## Risk Mitigation
1. **API Stability**: SoundCloud may change internal APIs
   - *Mitigation*: Monitor library updates, have fallback plans
   
2. **Terms of Service**: Using undocumented APIs
   - *Mitigation*: Clear user disclaimers, educational purpose
   
3. **Rate Limiting**: Potential API throttling
   - *Mitigation*: Implement request caching and delays

## User Disclaimer
```
⚠️  IMPORTANT DISCLAIMER ⚠️

This application uses SoundCloud's undocumented internal API 
through a reverse-engineered Go library. This may violate 
SoundCloud's Terms of Service.

By using this software, you acknowledge:
- This is for educational/personal use only
- You assume full responsibility for ToS compliance
- The functionality may break if SoundCloud changes their API
- Consider supporting artists through official channels

Use at your own discretion and risk.
```

This approach allows us to build a functional SoundCloud TUI while being transparent about the legal considerations and technical limitations.

## Test-Driven Development Methodology

This project follows TDD principles for better code quality and design:

### TDD Cycle for Each Feature
1. **Write failing tests** - Define expected behavior first
2. **Write minimal code** - Make tests pass with simplest implementation  
3. **Refactor** - Improve code while keeping tests green
4. **Repeat** - Continue for next feature or improvement

### Testing Strategy
- **Unit Tests**: Individual functions and methods
- **Component Tests**: TUI components and their interactions
- **Integration Tests**: SoundCloud API integration and audio streaming
- **End-to-End Tests**: Complete user workflows

### Test Structure
```
tests/
├── unit/
│   ├── soundcloud/          # SoundCloud client tests
│   ├── audio/               # Audio player tests
│   └── ui/                  # TUI component tests
├── integration/
│   ├── api_test.go          # API integration tests
│   └── streaming_test.go    # Audio streaming tests
└── e2e/
    └── workflows_test.go    # Complete user scenarios
```

### Benefits of TDD Approach
- ✅ **Better Design**: Tests force thinking about interfaces first
- ✅ **Higher Quality**: Catches bugs early in development cycle
- ✅ **Documentation**: Tests serve as living documentation
- ✅ **Confidence**: Safe refactoring with comprehensive test coverage
- ✅ **Debugging**: Tests help isolate and fix issues quickly