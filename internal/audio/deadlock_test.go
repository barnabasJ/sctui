package audio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
	"github.com/gopxl/beep/wav"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeStreamer is a minimal beep.StreamSeekCloser stand-in that lets tests
// exercise locking behavior without decoding real audio.
type fakeStreamer struct {
	length int
	pos    int
}

func (f *fakeStreamer) Stream(samples [][2]float64) (n int, ok bool) { return 0, false }
func (f *fakeStreamer) Err() error                                   { return nil }
func (f *fakeStreamer) Len() int                                     { return f.length }
func (f *fakeStreamer) Position() int                                { return f.pos }
func (f *fakeStreamer) Seek(p int) error                             { f.pos = p; return nil }
func (f *fakeStreamer) Close() error                                 { return nil }

// callWithTimeout runs fn on its own goroutine and fails the test if it
// doesn't return within the given timeout, which is how a self-deadlock
// (the same goroutine blocking on a lock it already holds) shows up.
func callWithTimeout(t *testing.T, timeout time.Duration, msg string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		fn()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal(msg)
	}
}

// TestBufferedStreamPlayer_Seek_DoesNotSelfDeadlock reproduces the reported
// deadlock: Seek acquires p.mu.Lock() (write) and then called GetDuration(),
// which acquires p.mu.RLock(). sync.RWMutex is not reentrant, so the same
// goroutine blocks on its own write lock forever. Once wedged, p.mu stays
// write-locked permanently, which in turn hangs every future GetState /
// GetPosition / GetDuration call -- including the ones the TUI's render loop
// (player.PlayerComponent.View) makes on every single frame. That is what
// turns one seek into a total, Ctrl+C-proof UI freeze while audio keeps
// playing in its own goroutine.
func TestBufferedStreamPlayer_Seek_DoesNotSelfDeadlock(t *testing.T) {
	p := NewBufferedStreamPlayer()
	p.streamer = &fakeStreamer{length: 44100 * 10}
	p.format = beep.Format{SampleRate: 44100, NumChannels: 1, Precision: 2}

	var seekErr error
	callWithTimeout(t, 2*time.Second, "Seek deadlocked (p.mu.Lock() then p.mu.RLock() on the same goroutine)", func() {
		seekErr = p.Seek(2 * time.Second)
	})
	assert.NoError(t, seekErr)

	// Simulate the UI's render loop, which calls these directly and, being
	// Bubble Tea's single event-loop goroutine, freezes completely (no more
	// input, no Ctrl+C) if any of them ever blocks.
	callWithTimeout(t, 2*time.Second, "GetState blocked after Seek returned", func() {
		p.GetState()
	})
	callWithTimeout(t, 2*time.Second, "GetPosition blocked after Seek returned", func() {
		p.GetPosition()
	})
	callWithTimeout(t, 2*time.Second, "GetDuration blocked after Seek returned", func() {
		p.GetDuration()
	})
}

// TestBeepPlayer_Seek_DoesNotSelfDeadlock is the same reproduction for the
// non-buffered BeepPlayer, which has the identical bug.
func TestBeepPlayer_Seek_DoesNotSelfDeadlock(t *testing.T) {
	p := NewBeepPlayer()
	p.streamer = &fakeStreamer{length: 44100 * 10}
	p.format = beep.Format{SampleRate: 44100, NumChannels: 1, Precision: 2}

	var seekErr error
	callWithTimeout(t, 2*time.Second, "Seek deadlocked (p.mu.Lock() then p.mu.RLock() on the same goroutine)", func() {
		seekErr = p.Seek(2 * time.Second)
	})
	assert.NoError(t, seekErr)

	callWithTimeout(t, 2*time.Second, "GetState blocked after Seek returned", func() {
		p.GetState()
	})
	callWithTimeout(t, 2*time.Second, "GetDuration blocked after Seek returned", func() {
		p.GetDuration()
	})
}

// generateSilentWAVBytes builds a tiny, valid, silent WAV file so playback
// tests can exercise the real decode/playback pipeline without any network
// or fixture assets.
func generateSilentWAVBytes(t *testing.T, numSamples int, sampleRate int) []byte {
	t.Helper()

	f, err := os.CreateTemp("", "silence-*.wav")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	defer f.Close()

	format := beep.Format{SampleRate: beep.SampleRate(sampleRate), NumChannels: 1, Precision: 2}
	require.NoError(t, wav.Encode(f, beep.Silence(numSamples), format))

	data, err := os.ReadFile(f.Name())
	require.NoError(t, err)
	return data
}

// TestBufferedStreamPlayer_CompletionCallback_NeverBlocksOnPMu deterministically
// reproduces the second bug: beep's speaker package invokes the
// playback-completion callback synchronously, on its own audio/oto callback
// thread, while holding its own internal mutex (speaker.Lock/Unlock). Pause,
// Resume, SetVolume and Seek all acquire p.mu first and then call
// speaker.Lock(). Before the fix, the completion callback acquired p.mu
// directly (speaker's lock, then p.mu) -- the exact inverse order -- so any
// overlap with one of those methods (p.mu, then speaker's lock) was a classic
// AB/BA deadlock: each side held the lock the other needed, freezing both
// the caller and the audio thread. Because the callback only ever fires once
// per track, racing it against real playback is inherently timing-dependent
// and flaky to reproduce; this test instead forces the exact interleaving
// directly, so it fails every time the ordering regresses.
func TestBufferedStreamPlayer_CompletionCallback_NeverBlocksOnPMu(t *testing.T) {
	p := NewBufferedStreamPlayer()
	p.positionTracker = &PositionTracker{}

	// Simulate a goroutine already inside Pause/Resume/SetVolume/Seek: it
	// holds p.mu and is blocked waiting for speaker's lock.
	speaker.Lock()
	muHeldBySomeoneWaitingOnSpeaker := make(chan struct{})
	go func() {
		p.mu.Lock()
		close(muHeldBySomeoneWaitingOnSpeaker)
		speaker.Lock() // released only when this test unlocks it below
		speaker.Unlock()
		p.mu.Unlock()
	}()
	<-muHeldBySomeoneWaitingOnSpeaker

	// registeredCallback is the exact function Play() hands to
	// beep.Callback; beep invokes it synchronously, from the audio thread,
	// while it holds speaker's lock (which this test is also holding).
	registeredCallback := p.completionCallback(make(chan bool, 1))

	callWithTimeout(t, 2*time.Second,
		"playback-completion callback blocked acquiring p.mu while speaker's lock was held -- AB/BA deadlock reintroduced",
		registeredCallback)

	speaker.Unlock()
}

// TestBufferedStreamPlayer_PlaybackCompletion_ConcurrentControlsDoNotDeadlock
// is a broader end-to-end smoke test: it drives a real (tiny, local, silent)
// playback through to natural completion while hammering the player's
// control methods from several goroutines, and fails if the whole thing
// doesn't finish promptly. It exercises the same code path as
// TestBufferedStreamPlayer_CompletionCallback_NeverBlocksOnPMu under
// realistic conditions, but the exact interleaving that triggers the AB/BA
// deadlock is a narrow, one-shot timing window, so this test alone is not a
// reliable reproduction of it.
func TestBufferedStreamPlayer_PlaybackCompletion_ConcurrentControlsDoNotDeadlock(t *testing.T) {
	wavData := generateSilentWAVBytes(t, 4000, 8000) // ~0.5s of silence at 8kHz

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write(wavData)
	}))
	defer server.Close()

	p := NewBufferedStreamPlayer()
	p.preloadSize = 256 // the fixture is far smaller than the 1MB default preload
	defer p.Close()

	// Stop hammering the moment the track completes naturally, rather than
	// racing a wall-clock timer against the hammering goroutines: under
	// heavy lock contention a busy wall-clock race is a source of test
	// flakiness unrelated to the thing being tested.
	trackCompleted := make(chan struct{})
	var closeOnce sync.Once
	p.SetStateChangeCallback(func(state PlayerState) {
		if state == StateStopped {
			closeOnce.Do(func() { close(trackCompleted) })
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, p.Play(ctx, server.URL))

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				select {
				case <-trackCompleted:
					return
				default:
				}
				_ = p.Pause()
				_ = p.SetVolume(0.5)
				_ = p.Resume()
				p.GetState()
				p.GetPosition()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("timed out: player controls deadlocked against the playback-completion callback")
	}
}
