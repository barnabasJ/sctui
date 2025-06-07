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

### Step 2: Audio Streaming Foundation
- [ ] Research streaming URL extraction from SoundCloud tracks
- [ ] Add Beep audio library dependency
- [ ] Implement basic audio playback from URLs
- [ ] Test streaming with actual SoundCloud tracks
- [ ] Add play/pause/stop controls via CLI flags

### Step 3: Core TUI Framework
- [ ] Add Bubble Tea and Lipgloss dependencies
- [ ] Design main application layout (search, player, queue)
- [ ] Implement search interface component
- [ ] Create track listing component with navigation
- [ ] Add basic vim-style keyboard shortcuts

### Step 4: Interactive Player
- [ ] Integrate audio streaming into TUI
- [ ] Add progress bar and time display
- [ ] Implement volume controls
- [ ] Add track metadata display panel
- [ ] Handle audio state management

### Step 5: Enhanced Features
- [ ] Track queueing system
- [ ] Playlist support and management
- [ ] Search result pagination
- [ ] Keyboard shortcuts reference
- [ ] Configuration file support

### Step 6: Polish and Testing
- [ ] Comprehensive error handling
- [ ] Performance optimization and caching
- [ ] User experience improvements
- [ ] Unit tests for core functionality
- [ ] Documentation and usage guide

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

### Phase 2: Audio Streaming & TUI
- [ ] Audio playback from streaming URLs
- [ ] Basic TUI interface with Bubble Tea
- [ ] Play/pause/stop controls
- [ ] Progress bar and time display
- [ ] Track metadata in TUI

### Phase 3: Enhanced Experience  
- [ ] Playlist creation and management
- [ ] Track queueing and autoplay
- [ ] Progress seeking
- [ ] Volume control
- [ ] Advanced keyboard shortcuts

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