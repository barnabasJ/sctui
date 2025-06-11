# Implementation Plan for Open-Source SoundCloud TUI Client in Go

## Current Status (December 2024)

### ✅ COMPLETED PHASES
- **Phase 1-4**: Complete TUI implementation with working audio playback
- **Phase 5**: Real audio implementation with HTTP streaming
- **Recent Fix**: Audio playback reliability and user feedback system
- **Phase 6**: Direct URL playback feature with auto-restart functionality
- **Phase 7**: Audio streaming responsiveness improvements

#### Latest Improvements (Audio Streaming & UI Responsiveness)
- ✅ **Direct Play Feature**: Added `--play <url>` flag for immediate track playback
- ✅ **Debug Tools**: Implemented `--test-audio` and `--test-tui` for troubleshooting
- ✅ **Auto-Restart**: Position-preserving restart for premature playback stops
- ✅ **Error Investigation**: Deep analysis of Beep library premature stopping issues
- ✅ **State Management**: Enhanced premature stop detection and recovery
- ✅ **Buffered Streaming**: Implemented BufferedStreamPlayer with progressive download
- ✅ **Audio/UI Coordination**: Fixed blocking issues between audio loading and UI responsiveness
- ✅ **Timeout Management**: Added proper timeout protection to prevent hanging
- ✅ **Error Handling**: Enhanced error recovery and state transitions for audio playback

#### Technical Architecture Achieved
- Complete Bubble Tea TUI with Search/Player/Queue views
- Real SoundCloud API integration via github.com/zackradisic/soundcloud-api
- Beep audio library with streaming MP3/WAV support
- Comprehensive error handling and state management
- Test-driven development with unit and integration tests

### 🎯 NEXT PHASE: Production Readiness

## OAuth 2.0 Browser-Based Authentication

### SoundCloud API registration challenges mean you cannot get official API access

The SoundCloud API v1 is **currently closed for new registrations** as of 2024,
presenting a fundamental challenge for new open-source projects. The unofficial
API v2 violates Terms of Service and is unreliable. For a legitimate open-source
implementation, you'll need to implement one of these strategies:

1. **User-provided API credentials** - Transfer responsibility to users who have
   existing access
2. **Contact SoundCloud directly** - Request special developer access for your
   open-source project
3. **Implement OAuth with placeholder credentials** - Allow users to configure
   their own client ID/secret

### Secure OAuth implementation pattern

Using the GitHub CLI approach as a reference, implement a dual-flow OAuth
strategy:

```go
// Core OAuth flow structure
type OAuthFlow struct {
    config      *oauth2.Config
    keyring     keyring.Keyring
    deviceFlow  bool
    verifier    string // For PKCE
}

// Implement browser-based flow with PKCE
func (f *OAuthFlow) AuthorizeBrowser(ctx context.Context) (*oauth2.Token, error) {
    // Generate PKCE verifier
    f.verifier = oauth2.GenerateVerifier()

    // Start local callback server
    redirectURL := "http://127.0.0.1:8888/callback"
    f.config.RedirectURL = redirectURL

    // Generate authorization URL with PKCE challenge
    authURL := f.config.AuthCodeURL("state",
        oauth2.AccessTypeOffline,
        oauth2.S256ChallengeOption(f.verifier))

    // Open browser
    if err := browser.OpenURL(authURL); err != nil {
        return nil, fmt.Errorf("failed to open browser: %w", err)
    }

    // Start callback server and wait for code
    code := f.waitForCallback()

    // Exchange code for token with PKCE verifier
    return f.config.Exchange(ctx, code, oauth2.VerifierOption(f.verifier))
}
```

### Token storage security

Use **99designs/keyring** for cross-platform secure storage with fallback to
encrypted config file:

```go
type TokenManager struct {
    keyring keyring.Keyring
    appName string
}

func (tm *TokenManager) StoreToken(token *oauth2.Token) error {
    // Try keyring first
    data, _ := json.Marshal(token)
    err := tm.keyring.Set(keyring.Item{
        Key:  "soundcloud_token",
        Data: data,
        Label: "SoundCloud OAuth Token",
    })

    if err != nil {
        // Fallback to encrypted file with user warning
        log.Warn("Failed to store token securely, using encrypted file")
        return tm.storeEncryptedFile(token)
    }
    return nil
}
```

## API Integration Architecture

### Rate limiting implementation

SoundCloud enforces **15,000 requests per day** - implement aggressive rate
limiting:

```go
type RateLimitedClient struct {
    httpClient *http.Client
    limiter    *rate.Limiter
    cache      *ttlcache.Cache
}

func NewSoundCloudClient() *RateLimitedClient {
    return &RateLimitedClient{
        httpClient: &http.Client{Timeout: 30 * time.Second},
        limiter:    rate.NewLimiter(rate.Every(24*time.Hour/15000), 10), // burst of 10
        cache:      ttlcache.NewCache(),
    }
}

func (c *RateLimitedClient) Get(ctx context.Context, url string) (*http.Response, error) {
    // Check cache first
    if cached, exists := c.cache.Get(url); exists {
        return cached.(*http.Response), nil
    }

    // Rate limit
    if err := c.limiter.Wait(ctx); err != nil {
        return nil, err
    }

    // Make request with exponential backoff
    return c.doWithRetry(ctx, url)
}
```

### Error handling with retry logic

```go
func (c *RateLimitedClient) doWithRetry(ctx context.Context, url string) (*http.Response, error) {
    backoff := []time.Duration{1 * time.Second, 3 * time.Second, 10 * time.Second}

    var lastErr error
    for attempt, delay := range backoff {
        resp, err := c.httpClient.Get(url)

        if err == nil && resp.StatusCode < 500 {
            if resp.StatusCode == 429 { // Rate limited
                retryAfter := resp.Header.Get("Retry-After")
                time.Sleep(parseRetryAfter(retryAfter))
                continue
            }
            return resp, nil
        }

        lastErr = err
        if attempt < len(backoff)-1 {
            time.Sleep(delay)
        }
    }

    return nil, fmt.Errorf("request failed after retries: %w", lastErr)
}
```

## Audio Streaming Architecture

### Streaming buffer management

Implement progressive download with intelligent buffering:

```go
type AudioStreamer struct {
    url         string
    buffer      *ring.Ring
    bufferSize  int
    preloadSize int
    client      *http.Client
    mu          sync.RWMutex
}

func (s *AudioStreamer) Stream(ctx context.Context) (beep.StreamCloser, beep.Format, error) {
    // Start progressive download in background
    go s.downloadInBackground(ctx)

    // Wait for initial buffer fill
    s.waitForPreload()

    // Create custom streamer that reads from ring buffer
    return &BufferedStreamer{
        buffer: s.buffer,
        format: s.detectFormat(),
    }, s.format, nil
}

func (s *AudioStreamer) downloadInBackground(ctx context.Context) {
    req, _ := http.NewRequestWithContext(ctx, "GET", s.url, nil)
    req.Header.Set("Range", fmt.Sprintf("bytes=%d-", s.currentPos))

    resp, err := s.client.Do(req)
    if err != nil {
        return
    }
    defer resp.Body.Close()

    buffer := make([]byte, 32*1024) // 32KB chunks
    for {
        n, err := resp.Body.Read(buffer)
        if n > 0 {
            s.mu.Lock()
            s.buffer.Value = buffer[:n]
            s.buffer = s.buffer.Next()
            s.mu.Unlock()
        }

        if err == io.EOF {
            break
        }
    }
}
```

### Beep audio integration

```go
type Player struct {
    ctrl     *beep.Ctrl
    volume   *effects.Volume
    format   beep.Format
    speaker  *sync.Once
}

func (p *Player) Play(streamer beep.StreamCloser, format beep.Format) error {
    // Initialize speaker once
    p.speaker.Do(func() {
        speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
    })

    // Wrap with control and volume
    p.ctrl = &beep.Ctrl{Streamer: streamer}
    p.volume = &effects.Volume{
        Streamer: p.ctrl,
        Base:     2,
        Volume:   0,
    }

    speaker.Play(p.volume)
    return nil
}
```

## TUI Architecture with Bubble Tea

### Project structure

```
soundcloud-tui/
├── cmd/
│   └── sctui/
│       └── main.go
├── internal/
│   ├── ui/
│   │   ├── app/
│   │   │   └── app.go         # Main app model
│   │   ├── components/
│   │   │   ├── player/        # Player controls
│   │   │   ├── search/        # Search interface
│   │   │   └── playlist/      # Playlist view
│   │   └── styles/
│   │       └── theme.go        # Lipgloss styles
│   ├── audio/
│   │   ├── player.go          # Audio player interface
│   │   ├── beep.go            # Beep implementation
│   │   └── stream.go          # Streaming logic
│   ├── api/
│   │   ├── client.go          # SoundCloud API client
│   │   ├── auth.go            # OAuth implementation
│   │   └── models.go          # API data models
│   └── config/
│       ├── config.go          # Configuration management
│       └── keyring.go         # Secure storage
└── pkg/
    └── soundcloud/            # Public API wrapper
```

### Main application model

```go
type App struct {
    // Sub-components
    player   player.Model
    search   search.Model
    playlist playlist.Model

    // Audio state
    audioPlayer audio.Player
    nowPlaying  *api.Track

    // UI state
    activeView  View
    windowSize  tea.WindowSizeMsg

    // Business logic
    client *api.Client
    config *config.Config
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return a.handleKeypress(msg)

    case tea.WindowSizeMsg:
        a.windowSize = msg
        return a, a.propagateResize()

    case player.PlaybackMsg:
        return a.handlePlayback(msg)

    case api.SearchResultMsg:
        a.search.SetResults(msg.Tracks)
        return a, nil
    }

    // Delegate to active component
    return a.delegateUpdate(msg)
}
```

### Component communication pattern

Use commands for all async operations and cross-component communication:

```go
// Audio state updates
func audioProgressTick() tea.Cmd {
    return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
        return progressUpdateMsg{time: t}
    })
}

// API requests
func searchTracks(query string, client *api.Client) tea.Cmd {
    return func() tea.Msg {
        tracks, err := client.Search(query)
        if err != nil {
            return errMsg{err}
        }
        return searchResultMsg{tracks}
    }
}
```

## Testing Strategy

### Unit testing with mocks

```go
// Mock audio player for testing
type MockPlayer struct {
    mock.Mock
}

func (m *MockPlayer) Play(url string) error {
    args := m.Called(url)
    return args.Error(0)
}

// Test model updates
func TestPlayerModel_PlayTrack(t *testing.T) {
    mockPlayer := new(MockPlayer)
    mockPlayer.On("Play", "http://example.com/track.mp3").Return(nil)

    model := player.New(mockPlayer)
    _, cmd := model.Update(player.PlayMsg{URL: "http://example.com/track.mp3"})

    require.NotNil(t, cmd)
    mockPlayer.AssertExpectations(t)
}
```

### Integration testing with teatest

```go
func TestAppIntegration(t *testing.T) {
    // Create test model with mocked dependencies
    app := createTestApp()

    tm := teatest.NewTestModel(t, app,
        teatest.WithInitialTermSize(80, 24))

    // Simulate user interaction
    tm.Send(tea.KeyMsg{Type: tea.KeyCtrlS}) // Open search
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("test song")})
    tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

    // Wait for search to complete
    teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
        return strings.Contains(string(out), "Search Results")
    })

    // Verify output
    golden := filepath.Join("testdata", "search_results.golden")
    teatest.RequireEqualOutput(t, tm.FinalOutput(t), golden)
}
```

## CI/CD Configuration

### GitHub Actions workflow

```yaml
name: Build and Release

on:
  push:
    branches: [main]
    tags: ["v*"]
  pull_request:

jobs:
  test:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go: ["1.21", "1.22"]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go }}

      - name: Install dependencies
        run: |
          if [ "$RUNNER_OS" == "Linux" ]; then
            sudo apt-get update
            sudo apt-get install -y libasound2-dev
          fi
        shell: bash

      - name: Run tests
        run: go test -v -race ./...

      - name: Run linter
        uses: golangci/golangci-lint-action@v3

  release:
    if: startsWith(github.ref, 'refs/tags/')
    needs: test
    runs-on: ubuntu-latest
    strategy:
      matrix:
        include:
          - goos: linux
            goarch: amd64
          - goos: linux
            goarch: arm64
          - goos: darwin
            goarch: amd64
          - goos: darwin
            goarch: arm64
          - goos: windows
            goarch: amd64

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.21"

      - name: Build binary
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
          CGO_ENABLED: 0
        run: |
          go build -ldflags="-s -w" -o sctui-${{ matrix.goos }}-${{ matrix.goarch }} ./cmd/sctui

      - name: Upload release assets
        uses: softprops/action-gh-release@v1
        with:
          files: sctui-*
```

## Key Implementation Considerations

### Security best practices

1. **Never embed SoundCloud API secrets** in open source code
2. **Use PKCE for all OAuth flows** to prevent authorization code interception
3. **Store tokens in system keyring** with encrypted file fallback
4. **Validate all API responses** and handle rate limiting gracefully

### Performance optimization

1. **Aggressive caching** to minimize API calls (15k/day limit)
2. **Progressive audio download** with ring buffer for smooth playback
3. **Concurrent component updates** using Bubble Tea commands
4. **Lazy loading** for search results and playlists

### User experience

1. **Vim-style keybindings** for navigation
2. **Real-time search** with debouncing
3. **Progress bars** for playback and downloads
4. **Responsive layout** that adapts to terminal size

### Distribution strategy

1. **go install** support:
   `go install github.com/yourusername/soundcloud-tui@latest`
2. **Homebrew formula** for macOS users
3. **Snap package** for Linux with audio permissions
4. **GitHub releases** with pre-built binaries

This implementation plan provides a solid foundation for building a
production-ready SoundCloud TUI client while navigating the API access
limitations and ensuring security, performance, and excellent user experience.
