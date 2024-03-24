package audio

import (
	"io"
	"sync"
)

// AsyncBuffer is an unbounded in-memory buffer that implements io.ReadWriteCloser.
// It allows a producer to write data as fast as possible without being throttled by the consumer.
type AsyncBuffer struct {
	mu     sync.Mutex
	cond   *sync.Cond
	chunks [][]byte
	closed bool
	err    error
}

// NewAsyncBuffer creates a new AsyncBuffer.
func NewAsyncBuffer() *AsyncBuffer {
	b := &AsyncBuffer{}
	b.cond = sync.NewCond(&b.mu)
	return b
}

// Write appends data to the buffer. It never blocks unless the buffer is closed.
func (b *AsyncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return 0, io.ErrClosedPipe
	}

	if len(p) == 0 {
		return 0, nil
	}

	// Copy the data to a new chunk to avoid issues if the caller reuses the slice.
	chunk := make([]byte, len(p))
	copy(chunk, p)
	b.chunks = append(b.chunks, chunk)

	b.cond.Signal()
	return len(p), nil
}

// Read reads data from the buffer. It blocks if the buffer is empty until more data is written or the buffer is closed.
func (b *AsyncBuffer) Read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for len(b.chunks) == 0 && !b.closed {
		b.cond.Wait()
	}

	if len(b.chunks) == 0 {
		if b.err != nil {
			return 0, b.err
		}
		return 0, io.EOF
	}

	chunk := b.chunks[0]
	n := copy(p, chunk)

	if n == len(chunk) {
		b.chunks = b.chunks[1:]
	} else {
		b.chunks[0] = chunk[n:]
	}

	return n, nil
}

// Close closes the buffer for writing. Subsequent reads will return io.EOF after the buffer is drained.
func (b *AsyncBuffer) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.closed {
		b.closed = true
		b.cond.Broadcast()
	}
	return nil
}

// CloseWithError closes the buffer and sets an error that will be returned by subsequent reads.
func (b *AsyncBuffer) CloseWithError(err error) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.closed {
		b.err = err
		b.closed = true
		b.cond.Broadcast()
	}
	return nil
}
