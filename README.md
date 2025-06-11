# SoundCloud TUI

A Terminal User Interface for SoundCloud written in Go, featuring real audio playback and interactive controls.

![SoundCloud TUI Demo](docs/demo.gif)
*Search, play, and control SoundCloud tracks directly from your terminal*

## ⚠️ Important Disclaimer

This application uses SoundCloud's undocumented internal API through a reverse-engineered Go library. This may violate SoundCloud's Terms of Service.

**By using this software, you acknowledge:**
- This is for educational/personal use only
- You assume full responsibility for ToS compliance  
- The functionality may break if SoundCloud changes their API
- Consider supporting artists through official channels

**Use at your own discretion and risk.**

## Features

✅ **Fully Implemented:**
- **Interactive TUI** with Bubble Tea framework
- **Real audio playback** using Beep library
- **Search and browse** SoundCloud tracks
- **Player controls** (play/pause, seek, volume)
- **Progress tracking** with smooth progress bars
- **Global hotkeys** (Space, ←→, +/-) work from any view
- **Track completion** handling with replay functionality
- **CLI mode** for search and track info

🎵 **TUI Navigation:**
- **Tab/Shift+Tab**: Switch between Search/Player/Queue views
- **Search View**: Enter to search, ↑↓ to navigate, Enter to select
- **Player View**: Space (play/pause), ←→ (seek 10s), +/- (volume)
- **Global Controls**: Audio controls work from any view

🚧 **Coming Soon:**
- Playlist management and queue functionality
- Favorites and user library integration
- Enhanced metadata display

## Installation

### Prerequisites
- Go 1.21 or later
- Audio system (ALSA/PulseAudio on Linux, Core Audio on macOS, DirectSound on Windows)

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd soundcloud-tui

# Install dependencies
make deps

# Build the application
make build
```

## Usage

### Interactive TUI Mode (Default)
```bash
./bin/sctui
```

Launches the full interactive Terminal UI with audio playback capabilities.

### CLI Mode Examples
```bash
# Search for tracks
./bin/sctui -search "lofi hip hop"

# Get track information
./bin/sctui -track "https://soundcloud.com/artist/track"

# Show help
./bin/sctui -help
```

### TUI Controls
- **Tab/Shift+Tab**: Navigate between views
- **Search View**: 
  - Type to search, Enter to execute
  - ↑↓ to navigate results, Enter to play
- **Global Audio Controls** (work from any view):
  - **Space**: Play/Pause
  - **←→**: Seek backward/forward (10 seconds)
  - **+/-**: Volume up/down
- **Ctrl+C**: Quit application

## Development

### Available Make Commands

```bash
make build       # Build the main application
make build-test  # Build test utilities
make test        # Run all tests
make clean       # Remove build artifacts
make run         # Build and run example search
make deps        # Install dependencies
make help        # Show available commands
```

### Project Structure

```
cmd/
├── sctui/          # Main TUI application entry point
└── test/           # Test utilities
internal/
├── audio/          # Audio playback and streaming (Beep integration)
├── soundcloud/     # SoundCloud API client wrapper
├── ui/             # TUI components (Bubble Tea)
│   ├── app/        # Main application model
│   ├── components/ # Player, Search, UI components
│   └── styles/     # Centralized styling
├── api/            # Legacy OAuth code (unused)
└── config/         # Configuration management
tests/
├── unit/           # Component unit tests
├── integration/    # API integration tests
└── e2e/            # End-to-end tests
notes/              # Planning and documentation
bin/                # Build artifacts (gitignored)
```

### Testing

We follow Test-Driven Development (TDD) principles:

```bash
# Run all tests
make test

# Run tests with coverage
go test -cover ./...
```

## Technical Architecture

### Audio Implementation
- **Beep Library**: High-performance audio playback with MP3/WAV support
- **HTTP Streaming**: Direct streaming from SoundCloud CDN (no downloads)
- **Real-time Position Tracking**: 250ms update intervals for smooth progress
- **Thread-safe Player**: Concurrent-safe with proper mutex locking

### TUI Framework
- **Bubble Tea**: Modern terminal UI framework with message passing
- **Component Architecture**: Modular player, search, and navigation components
- **Global State Management**: Centralized app state with component communication
- **Responsive Design**: Adapts to terminal size changes

### SoundCloud Integration  
- **Reverse-engineered API**: Uses `github.com/zackradisic/soundcloud-api`
- **No Official Credentials**: Works without API keys or authentication
- **Real Stream URLs**: Extracts actual CDN URLs for audio playback
- **CloudFront Authentication**: Handles signed URL parameters

## Roadmap

**Phase 1: Core TUI** ✅ 
- Interactive TUI with Bubble Tea
- Search and navigation
- Player controls and state management

**Phase 2: Real Audio** ✅
- Beep library integration
- HTTP audio streaming  
- Position/duration tracking
- Volume and seeking controls

**Phase 3: Enhanced Experience** 🚧
- Playlist management and queue
- Favorites and user library
- Advanced metadata display
- Improved error handling

## Contributing

This is an educational project demonstrating TUI development and audio programming in Go. Contributions welcome for:

- **Bug fixes and improvements**: Help make the player more robust
- **Test coverage**: Expand unit and integration test coverage  
- **Documentation**: Improve guides and API documentation
- **Performance optimizations**: Audio streaming and UI responsiveness improvements
- **New features**: Queue management, playlists, enhanced metadata

### Development Guidelines
- Follow TDD principles - write tests first
- Use the Makefile for all build operations
- Update CLAUDE.md for any new commands or workflows
- Ensure changes work across platforms (Linux/macOS/Windows)
- Include appropriate error handling and user feedback

### Getting Started
1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Write tests for your changes
4. Implement your feature
5. Ensure all tests pass: `make test`
6. Submit a pull request

Please ensure all changes include appropriate tests and documentation.

## Troubleshooting

### Audio Issues
- **Linux**: Ensure ALSA or PulseAudio is installed and running
  ```bash
  # Check audio system
  aplay -l  # List audio devices
  pulseaudio --check  # Check PulseAudio status
  ```
- **macOS**: Should work out of the box with Core Audio
- **Windows**: Requires DirectSound (usually pre-installed)

### Build Issues
- **Missing dependencies**: Run `make deps` to install Go modules
- **Permission errors**: Ensure Go workspace has write permissions
- **Network issues**: Some dependencies require internet access

### Runtime Issues
- **TUI not displaying**: Ensure terminal supports 256 colors
- **Track not playing**: Check internet connection and SoundCloud availability
- **Controls not responding**: Try different terminal emulator or update to latest version

For more help, check the [troubleshooting guide](notes/troubleshooting.md) or open an issue.

## Legal

This project is for educational purposes only. Users are responsible for compliance with SoundCloud's Terms of Service. The developers assume no liability for misuse of this software.

## License

[License TBD]