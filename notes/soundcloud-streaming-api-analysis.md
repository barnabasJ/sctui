# SoundCloud Streaming API Analysis

## Overview

Based on analysis of the `github.com/zackradisic/soundcloud-api` library, here's how to extract real streaming URLs from SoundCloud's internal API v2.

## Track Structure

Each SoundCloud track contains a `Media` field with transcoding information:

```go
type Track struct {
    ID          int64
    Title       string
    Duration    int64  // DurationMS in the API
    Media       Media
    // ... other fields
}

type Media struct {
    Transcodings []Transcoding
}

type Transcoding struct {
    URL      string             // API endpoint to get actual stream URL
    Preset   string             // Quality/format preset
    Snipped  bool              // Whether this is a preview/snippet
    Format   TranscodingFormat  // Protocol and MIME type info
}

type TranscodingFormat struct {
    Protocol string  // "progressive" or "hls"
    MimeType string  // "audio/mpeg", "audio/mpegurl", "audio/ogg; codecs=\"opus\""
}
```

## Available Transcodings

A typical track has 4 transcoding options:

### 1. HLS Adaptive Bitrate (abr_sq)
- **URL**: `https://api-v2.soundcloud.com/media/soundcloud:tracks:{ID}/{hash}/stream/hls`
- **Protocol**: `hls`
- **MIME Type**: `audio/mpegurl`
- **Use Case**: Adaptive streaming, best for bandwidth adaptation

### 2. HLS MP3 (mp3_1_0)
- **URL**: `https://api-v2.soundcloud.com/media/soundcloud:tracks:{ID}/{hash}/stream/hls`
- **Protocol**: `hls`
- **MIME Type**: `audio/mpeg`
- **Use Case**: MP3 format with HLS streaming

### 3. Progressive MP3 (mp3_1_0)
- **URL**: `https://api-v2.soundcloud.com/media/soundcloud:tracks:{ID}/{hash}/stream/progressive`
- **Protocol**: `progressive`
- **MIME Type**: `audio/mpeg`
- **Use Case**: Direct MP3 download/streaming (preferred for audio players)

### 4. HLS Opus (opus_0_0)
- **URL**: `https://api-v2.soundcloud.com/media/soundcloud:tracks:{ID}/{hash}/stream/hls`
- **Protocol**: `hls`
- **MIME Type**: `audio/ogg; codecs="opus"`
- **Use Case**: High efficiency codec, smaller file sizes

## Quality Mapping

- **abr_sq**: Adaptive Standard Quality (HLS)
- **mp3_1_0**: Standard Quality MP3 (~128 kbps)
- **opus_0_0**: Opus compression (high efficiency)

## URL Extraction Workflow

### Method 1: Using GetDownloadURL (Recommended)

```go
api, err := soundcloudapi.New(soundcloudapi.APIOptions{})
if err != nil {
    return nil, err
}

// Get direct streaming URL
streamURL, err := api.GetDownloadURL(track.PermalinkURL, "progressive")
if err != nil {
    return nil, fmt.Errorf("failed to get stream URL: %w", err)
}

// Result: https://cf-media.sndcdn.com/{file}.128.mp3?Policy=...&Signature=...&Key-Pair-Id=...
```

### Method 2: Manual Transcoding Selection

```go
// Get track info first
tracks, err := api.GetTrackInfo(soundcloudapi.GetTrackInfoOptions{
    ID: []int64{trackID},
})

track := tracks[0]

// Find preferred transcoding
var selectedTranscoding *soundcloudapi.Transcoding
for _, t := range track.Media.Transcodings {
    if t.Format.Protocol == "progressive" && t.Format.MimeType == "audio/mpeg" {
        selectedTranscoding = &t
        break
    }
}

// Use the transcoding URL to get actual media URL
// (This requires additional API call to the transcoding URL)
```

## Stream URL Format

The final streaming URL follows this pattern:
```
https://cf-media.sndcdn.com/{encodedFilename}.128.mp3?Policy={base64Policy}&Signature={signature}&Key-Pair-Id={keyId}
```

This is a signed URL with:
- **Policy**: Base64-encoded access policy with expiration
- **Signature**: HMAC signature for authentication
- **Key-Pair-Id**: CloudFront key pair identifier

## Implementation Strategy

### For Audio Players (Beep Library)

1. **Prefer Progressive MP3**: Use `protocol: "progressive"` and `mime_type: "audio/mpeg"`
2. **Use GetDownloadURL**: Simplest method to get playable URL
3. **Handle Expiration**: URLs expire, cache track info and re-fetch URLs as needed
4. **Stream vs Download**: Progressive URLs support both streaming and seeking

### Quality Selection Algorithm

```go
func selectBestTranscoding(transcodings []Transcoding, preferredQuality string) *Transcoding {
    // Priority order:
    // 1. Progressive MP3 (best for audio players)
    // 2. HLS MP3 (fallback for streaming)
    // 3. HLS Adaptive (bandwidth adaptation)
    // 4. Opus (high efficiency)
    
    priorities := []struct {
        protocol string
        mimeType string
        preset   string
    }{
        {"progressive", "audio/mpeg", "mp3_1_0"},
        {"hls", "audio/mpeg", "mp3_1_0"},
        {"hls", "audio/mpegurl", "abr_sq"},
        {"hls", "audio/ogg", "opus_0_0"},
    }
    
    for _, priority := range priorities {
        for _, t := range transcodings {
            if t.Format.Protocol == priority.protocol && 
               strings.Contains(t.Format.MimeType, priority.mimeType) {
                return &t
            }
        }
    }
    
    return nil // No suitable transcoding found
}
```

## Real Implementation for stream.go

The current mock implementation should be replaced with:

```go
func (e *SoundCloudStreamExtractor) ExtractStreamURL(ctx context.Context, trackID int64) (*StreamInfo, error) {
    // Get track information 
    tracks, err := e.api.GetTrackInfo(soundcloudapi.GetTrackInfoOptions{
        ID: []int64{trackID},
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get track info: %w", err)
    }
    
    track := tracks[0]
    
    // Get actual streaming URL using the library's method
    streamURL, err := e.api.GetDownloadURL(track.PermalinkURL, "progressive")
    if err != nil {
        return nil, fmt.Errorf("failed to get download URL: %w", err)
    }
    
    // Determine format and quality from transcodings
    format := "mp3"
    quality := "sq"
    
    for _, t := range track.Media.Transcodings {
        if t.Format.Protocol == "progressive" && strings.Contains(t.Format.MimeType, "audio/mpeg") {
            format = "mp3"
            quality = "sq" // Standard quality for mp3_1_0 preset
            break
        }
    }
    
    return &StreamInfo{
        URL:      streamURL,
        Format:   format,
        Quality:  quality,
        Duration: track.DurationMS,
    }, nil
}
```

## Error Handling

Common issues and solutions:

1. **No Transcodings Available**: Track may be private or deleted
2. **URL Expiration**: Re-fetch URLs when they expire (typically 24 hours)
3. **Geographic Restrictions**: Some tracks unavailable in certain regions
4. **Rate Limiting**: Implement delays between API calls if needed

## Legal Considerations

⚠️ **Important**: This uses SoundCloud's undocumented internal API v2 which may violate their Terms of Service. The URLs returned are legitimate streaming URLs with proper authentication, but accessing them programmatically may be against ToS.

## Testing Strategy

1. **Mock Tests**: Use fake transcoding data for unit tests
2. **Integration Tests**: Test with real API but use public domain tracks
3. **URL Validation**: Verify returned URLs are playable
4. **Expiration Handling**: Test URL refresh behavior

This analysis provides the foundation for implementing real audio streaming in the SoundCloud TUI project.