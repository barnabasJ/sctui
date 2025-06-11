package audio

import (
	"errors"
	"io"
	"sync"
	"time"
)

// BufferReader implements io.Reader for the StreamBuffer
type BufferReader struct {
	buffer   *StreamBuffer
	position int64
	mu       sync.RWMutex
}

// NewBufferReader creates a new BufferReader for the given StreamBuffer
func NewBufferReader(buffer *StreamBuffer) *BufferReader {
	return &BufferReader{
		buffer:   buffer,
		position: 0,
	}
}

// Read implements io.Reader
func (br *BufferReader) Read(p []byte) (n int, err error) {
	br.mu.Lock()
	defer br.mu.Unlock()
	
	// Check if we have data available
	br.buffer.mu.RLock()
	available := br.buffer.writePos - br.position
	completed := br.buffer.completed
	br.buffer.mu.RUnlock()
	
	if available == 0 {
		if completed {
			return 0, io.EOF
		}
		// Block and wait for more data with timeout
		maxWait := 50 // 50 * 100ms = 5 seconds max
		for i := 0; i < maxWait && available == 0; i++ {
			time.Sleep(100 * time.Millisecond)
			br.buffer.mu.RLock()
			available = br.buffer.writePos - br.position
			completed = br.buffer.completed
			br.buffer.mu.RUnlock()
			
			if completed && available == 0 {
				return 0, io.EOF
			}
		}
		
		// If still no data after waiting, return EOF to prevent hanging
		if available == 0 {
			return 0, io.EOF
		}
	}
	
	// Copy available data
	br.buffer.mu.RLock()
	toCopy := int64(len(p))
	if toCopy > available {
		toCopy = available
	}
	
	copy(p, br.buffer.data[br.position:br.position+toCopy])
	br.buffer.mu.RUnlock()
	
	br.position += toCopy
	return int(toCopy), nil
}

// Reset resets the reader position to the beginning
func (br *BufferReader) Reset() {
	br.mu.Lock()
	defer br.mu.Unlock()
	br.position = 0
}

// Seek implements seeking within the buffer
func (br *BufferReader) Seek(offset int64, whence int) (int64, error) {
	br.mu.Lock()
	defer br.mu.Unlock()
	
	var newPos int64
	
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = br.position + offset
	case io.SeekEnd:
		br.buffer.mu.RLock()
		newPos = br.buffer.writePos + offset
		br.buffer.mu.RUnlock()
	default:
		return br.position, errors.New("invalid seek")
	}
	
	if newPos < 0 {
		return br.position, errors.New("invalid seek")
	}
	
	br.buffer.mu.RLock()
	if newPos > br.buffer.writePos {
		br.buffer.mu.RUnlock()
		return br.position, errors.New("invalid seek")
	}
	br.buffer.mu.RUnlock()
	
	br.position = newPos
	return br.position, nil
}

// Close implements io.Closer (BufferReader doesn't need to close anything)
func (br *BufferReader) Close() error {
	return nil
}