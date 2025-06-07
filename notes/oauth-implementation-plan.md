# SoundCloud API Access Implementation Plan

## Problem Statement
SoundCloud TUI client needs access to SoundCloud content and data. However, SoundCloud's official API registration is permanently closed as of 2024, and using unofficial methods violates their Terms of Service. We need to find legitimate alternatives for accessing SoundCloud data.

## Solution Overview
Implement a dual-flow OAuth strategy similar to GitHub CLI:
- Browser-based OAuth flow with PKCE for security
- Local callback server to capture authorization codes
- Secure token storage using system keyring with encrypted file fallback
- Token refresh capability for long-term usage

## Implementation Plan

### Step 1: Project Setup and Dependencies
- [x] Initialize Go module and project structure
- [x] Add required dependencies (oauth2, keyring, browser utilities)
- [x] Create basic CLI structure with auth command
- [x] Test module initialization and dependency resolution

### Step 2: Configuration Management
- [ ] Implement config struct for OAuth settings
- [ ] Add support for user-provided client ID/secret
- [ ] Create config file loading/saving functionality  
- [ ] Test configuration persistence and loading

### Step 3: OAuth Core Implementation
- [ ] Implement OAuth config with PKCE support
- [ ] Create PKCE verifier generation and challenge
- [ ] Add authorization URL generation with proper scopes
- [ ] Test OAuth config creation and URL generation

### Step 4: Browser Flow and Callback Server
- [ ] Implement local HTTP server for OAuth callback
- [ ] Add browser opening functionality
- [ ] Create callback handler for authorization codes
- [ ] Implement timeout and error handling for callback
- [ ] Test complete browser flow end-to-end

### Step 5: Token Management
- [ ] Implement secure token storage with keyring
- [ ] Add encrypted file fallback for keyring failures
- [ ] Create token retrieval and validation
- [ ] Add token refresh capability
- [ ] Test token storage and retrieval across system restarts

### Step 6: CLI Integration
- [ ] Create auth command with proper flags
- [ ] Add token status checking
- [ ] Implement logout/token clearing
- [ ] Add help documentation and user guidance
- [ ] Test CLI commands and user experience

## Technical Details

### Dependencies
```go
require (
    golang.org/x/oauth2 v0.15.0
    github.com/99designs/keyring v1.2.2
    github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8
    github.com/spf13/cobra v1.8.0  // For CLI commands
)
```

### File Structure
```
internal/
├── api/
│   ├── auth.go          # OAuth implementation
│   ├── client.go        # API client with token handling
│   └── models.go        # Data structures
├── config/
│   ├── config.go        # Configuration management
│   └── keyring.go       # Secure token storage
cmd/sctui/
└── main.go              # CLI entry point
```

### OAuth Configuration
- **Authorization URL**: `https://soundcloud.com/connect`
- **Token URL**: `https://api.soundcloud.com/oauth2/token`
- **Scopes**: `non-expiring` (for persistent access)
- **Redirect URI**: `http://127.0.0.1:8888/callback`

### Security Considerations
- Use PKCE (RFC 7636) for all OAuth flows
- Store tokens in system keyring when available
- Fallback to encrypted config file with user warning
- Never log or expose tokens in plaintext
- Implement proper token validation and refresh

## Success Criteria
- User can authenticate via `sctui -auth` command
- OAuth flow opens browser and completes successfully
- Tokens are stored securely and persist across restarts
- Token refresh works automatically when needed
- Clear error messages for authentication failures
- Works on macOS, Linux, and Windows

## Notes/Considerations
- SoundCloud API registration is closed, so users must provide their own client credentials
- Consider adding device flow as alternative for headless systems
- May need to handle different callback URLs for development vs production
- Should provide clear documentation on obtaining SoundCloud API credentials
- Token storage location should be documented for troubleshooting