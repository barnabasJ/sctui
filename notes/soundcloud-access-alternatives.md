# SoundCloud Access Alternatives - Updated Implementation Plan

## Problem Statement
SoundCloud's official API registration is permanently closed as of 2024. Using unofficial methods or extracted client IDs violates their Terms of Service and could result in legal consequences. We need legitimate alternatives for building a SoundCloud-like TUI experience.

## Legitimate Alternative Solutions

### Option 1: Web Scraping with Disclaimers (High Risk)
- Extract data from SoundCloud's public web pages
- **Risks**: Violates ToS, prone to breakage, legal issues
- **Status**: Not recommended for open-source project

### Option 2: User-Provided API Credentials (Medium Risk)
- Require users to provide their own valid SoundCloud API credentials
- Document how users with existing API access can use the tool
- **Risks**: Limited user base, still depends on SoundCloud API
- **Status**: Possible but limited utility

### Option 3: Alternative Music Platforms (Recommended)
Focus on platforms with open/accessible APIs:

#### 3a. Jamendo API
- **Status**: Open API available
- **Content**: Creative Commons licensed music
- **Access**: Free API key registration
- **URL**: https://developer.jamendo.com/

#### 3b. Freesound API
- **Status**: Open API with registration
- **Content**: Sound effects and audio clips
- **Access**: Free API key after registration
- **URL**: https://freesound.org/docs/api/

#### 3c. Last.fm API
- **Status**: Open API available
- **Content**: Music metadata, recommendations, scrobbling
- **Access**: Free API key registration
- **URL**: https://www.last.fm/api

#### 3d. Spotify Web API
- **Status**: Open but with limitations
- **Content**: Music metadata, playlists (30-second previews only)
- **Access**: Free API registration required
- **URL**: https://developer.spotify.com/

### Option 4: Multi-Platform Music TUI (Recommended)
Build a generic music TUI that supports multiple platforms:
- Start with open APIs (Jamendo, Last.fm)
- Add plugin architecture for future platforms
- Allow users to add SoundCloud support if they have credentials

## Recommended Implementation Path

### Phase 1: Generic Music TUI Foundation
1. **Core TUI Framework**
   - Bubble Tea-based interface
   - Audio playback with Beep
   - Generic plugin architecture

2. **Plugin System**
   - Interface for music providers
   - Standardized track/artist data models
   - Configuration per provider

### Phase 2: Jamendo Integration (Proof of Concept)
1. **Jamendo API Client**
   - OAuth or API key authentication
   - Search functionality
   - Track streaming
   - Legal, CC-licensed content

2. **Basic Features**
   - Search tracks and artists
   - Play/pause/skip controls
   - Volume control
   - Simple playlists

### Phase 3: Additional Platform Support
1. **Last.fm Integration**
   - Music discovery and recommendations
   - Scrobbling support
   - Artist/track metadata

2. **Local File Support**
   - MP3/FLAC/OGG playback
   - File browser
   - Metadata parsing

### Phase 4: Advanced Features
1. **Playlist Management**
   - Cross-platform playlists
   - Import/export functionality
   - Collaborative features

2. **Enhanced Discovery**
   - Recommendation algorithms
   - Genre-based browsing
   - Social features where supported

## Technical Architecture

### Plugin Interface
```go
type MusicProvider interface {
    Name() string
    Authenticate(ctx context.Context) error
    Search(query string) ([]Track, error)
    GetTrackURL(trackID string) (string, error)
    GetArtist(artistID string) (*Artist, error)
    GetAlbum(albumID string) (*Album, error)
}
```

### Configuration
```yaml
providers:
  jamendo:
    enabled: true
    api_key: "${JAMENDO_API_KEY}"
  lastfm:
    enabled: true
    api_key: "${LASTFM_API_KEY}"
    api_secret: "${LASTFM_API_SECRET}"
  soundcloud:
    enabled: false  # User must provide credentials
    client_id: "${SOUNDCLOUD_CLIENT_ID}"
    client_secret: "${SOUNDCLOUD_CLIENT_SECRET}"
```

## Benefits of This Approach

1. **Legal Compliance**: Uses only legitimate, authorized APIs
2. **Extensibility**: Easy to add new platforms as APIs become available
3. **User Choice**: Users can enable platforms they have access to
4. **Educational Value**: Demonstrates proper API integration patterns
5. **Community Friendly**: Open source without ToS violations

## Success Criteria

1. **Core TUI**: Responsive, keyboard-driven music interface
2. **Jamendo Integration**: Full search, playback, and discovery
3. **Plugin Architecture**: Easy addition of new music providers
4. **Documentation**: Clear setup instructions for each platform
5. **Legal Clarity**: No Terms of Service violations

## Next Steps

1. Pivot current OAuth implementation to generic provider authentication
2. Implement Jamendo API client as first provider
3. Build core TUI with music playback capabilities
4. Add plugin system for multiple providers
5. Document setup process for legitimate API access

This approach provides a legitimate, extensible foundation while avoiding the legal and technical risks of unauthorized SoundCloud API access.