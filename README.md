# SoundCloud TUI

A Terminal User Interface for SoundCloud written in Go.

## ⚠️ Important Disclaimer

This application uses SoundCloud's undocumented internal API through a reverse-engineered Go library. This may violate SoundCloud's Terms of Service.

**By using this software, you acknowledge:**
- This is for educational/personal use only
- You assume full responsibility for ToS compliance  
- The functionality may break if SoundCloud changes their API
- Consider supporting artists through official channels

**Use at your own discretion and risk.**

## Features

✅ **Currently Working:**
- Search tracks by keyword
- Display track metadata (title, artist, duration)
- Retrieve track information from URLs
- CLI interface with help system

🚧 **Coming Soon:**
- Audio playback and streaming
- Interactive TUI with Bubble Tea
- Playlist management
- Volume controls

## Installation

### Prerequisites
- Go 1.21 or later

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

### Search for Tracks
```bash
./bin/sctui -search "lofi hip hop"
```

### Get Track Information
```bash
./bin/sctui -track "https://soundcloud.com/artist/track"
```

### Show Help
```bash
./bin/sctui -help
```

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
├── sctui/          # Main CLI application
└── test/           # Test utilities
internal/
├── soundcloud/     # SoundCloud API client
├── api/           # Legacy OAuth code (unused)
└── config/        # Configuration management
notes/             # Planning and documentation
bin/               # Build artifacts (gitignored)
```

### Testing

We follow Test-Driven Development (TDD) principles:

```bash
# Run all tests
make test

# Run tests with coverage
go test -cover ./...
```

## Roadmap

See [implementation plan](notes/soundcloud-go-implementation.md) for detailed roadmap.

**Phase 1: MVP** ✅ 
- Basic SoundCloud integration
- CLI search and track info

**Phase 2: Audio Streaming (TDD)**
- Audio playback from streaming URLs
- Basic TUI interface
- Player controls

**Phase 3: Enhanced Experience**
- Playlist management
- Advanced TUI features
- Keyboard shortcuts

## Contributing

This is an educational project. Contributions welcome for:
- Bug fixes and improvements
- Test coverage
- Documentation
- Performance optimizations

Please ensure all changes include appropriate tests.

## Legal

This project is for educational purposes only. Users are responsible for compliance with SoundCloud's Terms of Service. The developers assume no liability for misuse of this software.

## License

[License TBD]