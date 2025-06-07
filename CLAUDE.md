# CLAUDE.md

This file provides guidance to Claude Code when working with the SoundCloud TUI project.

## Project Overview

This is a SoundCloud Terminal User Interface (TUI) built in Go using:
- `github.com/zackradisic/soundcloud-api` for SoundCloud integration (no official API key needed)
- Bubble Tea for the TUI framework
- Beep library for audio playback
- Test-driven development methodology

## Key Commands

### Build and Run (Use Makefile!)
- `make build` - Build the TUI application to `bin/sctui`
- `make test` - Run all tests
- `make clean` - Clean build artifacts
- `make run` - Build and run example search
- `make deps` - Install and tidy dependencies
- `make help` - Show available commands

### Running the Application
```bash
# Interactive TUI mode
./bin/sctui

# CLI mode examples  
./bin/sctui -search "lofi hip hop"
./bin/sctui -track "https://soundcloud.com/artist/track"
./bin/sctui -help
```

### TUI Navigation
- **Tab/Shift+Tab**: Navigate between Search/Player/Queue views
- **Search View**: Enter to search, ↑↓ to navigate results, Enter to select track
- **Player View**: Space (play/pause), ←→ (seek), +/- (volume)
- **Ctrl+C**: Quit application

## Architecture & Structure

### Project Structure
```
cmd/sctui/main.go           # Application entry point
internal/
├── audio/
│   ├── player.go           # Audio player interface and BeepPlayer implementation
│   └── stream.go           # Stream URL extraction from SoundCloud
├── soundcloud/
│   └── client.go           # SoundCloud API wrapper
└── ui/
    ├── app/app.go          # Main TUI application model
    ├── components/
    │   ├── player/player.go    # Player component with controls
    │   └── search/search.go    # Search component
    └── styles/styles.go    # Centralized UI styling
tests/
├── unit/                   # Component unit tests
├── integration/            # API integration tests  
└── e2e/                   # End-to-end workflow tests
```

### Component Architecture

#### TUI Components (Bubble Tea)
- **App**: Main application model handling view switching and global state
- **SearchComponent**: Track search interface with result navigation
- **PlayerComponent**: Audio player with progress, volume, metadata display
- All components implement `tea.Model` interface

#### Audio System
- **Player interface**: Abstract audio playback (Play, Pause, Stop, Seek, Volume)
- **BeepPlayer**: Concrete implementation using Beep library
- **StreamExtractor**: Extracts playable URLs from SoundCloud track IDs

#### State Management
- **Player States**: Idle → Loading → Playing ↔ Paused → Error
- **State Sync**: UI components sync with audio player state via `syncStateWithAudioPlayer()`
- **Error Handling**: Graceful error states with recovery paths

## Development Patterns

### Test-Driven Development
- **Write tests first** for all new functionality
- **Use TodoWrite/TodoRead** to track TDD cycles and progress
- **Test structure**: unit/component/integration tests in parallel directories
- **Mock implementations**: Centralized mocks in `tests/unit/*/mocks.go`

### Styling and UI
- **Centralized styles** in `internal/ui/styles/styles.go`
- **Reusable functions**: `RenderProgressBar()`, `RenderMetadataPanel()`, etc.
- **Visual feedback**: Volume icons (🔇🔉🔊), progress bars with Unicode blocks
- **Color scheme**: SoundCloud orange primary, consistent secondary colors

### Error Handling
- **Graceful degradation**: UI works even with mock/missing audio implementation
- **Clear error messages**: Specific error states with recovery suggestions
- **State recovery**: Can play new tracks after errors

## Current Status (December 2025)

### ✅ Working
- Complete TUI interface with Search/Player/Queue views
- SoundCloud search and track metadata retrieval
- Player UI with progress bar, volume controls, metadata display
- Navigation and keyboard shortcuts
- State management and error handling

### 🚧 Current Limitation
- **Audio player is mock implementation** - shows UI but doesn't play actual audio
- **Stream URLs are fake** - generates placeholder URLs instead of real ones

### 🎯 Next Steps (Step 5: Real Audio Implementation)
1. **Real stream URL extraction** from SoundCloud's transcoding API
2. **Beep library integration** for actual audio playback
3. **Streaming audio support** (no full download required)
4. **Real-time position tracking** during playback
5. **Volume control integration** with audio output

## Technical Notes

### SoundCloud API Usage
- Uses **undocumented internal API** via reverse-engineered library
- **No official credentials required** but may violate ToS
- **Rate limiting considerations** - implement delays if needed
- **Disclaimer required** for users about legal considerations

### Audio Implementation Challenges
- **Format support**: SoundCloud uses MP3/M4A formats
- **Streaming vs download**: Prefer streaming for memory efficiency
- **Cross-platform audio**: Beep should work on Linux/macOS/Windows
- **Seeking support**: Real-time position updates and seek capabilities

### Testing Strategy
- **Package naming**: Use `package ui_test` for UI tests to avoid conflicts
- **Mock audio player**: Test UI behavior without actual audio
- **Integration testing**: Test SoundCloud API calls with real data
- **TUI testing**: Component behavior with Bubble Tea message passing

## Dependencies

### Core Dependencies
```go
github.com/zackradisic/soundcloud-api  // SoundCloud API access
github.com/charmbracelet/bubbletea     // TUI framework
github.com/charmbracelet/lipgloss      // TUI styling
github.com/gopxl/beep                  // Audio playback (planned)
```

### Development Dependencies
```go
github.com/stretchr/testify            // Testing framework
```

## Legal Considerations

⚠️ **Important**: This project uses SoundCloud's undocumented API which may violate their Terms of Service. Users must be informed and accept responsibility. The application includes a comprehensive disclaimer on startup.

## Development Tips

- **Always use the Makefile** instead of direct `go build` commands
- **Update planning documents** as you discover new requirements
- **Test the UI frequently** by running `./bin/sctui` to verify UX
- **Mock first, implement later** - validate architecture before complex implementation
- **Keep state management explicit** - document state transitions clearly