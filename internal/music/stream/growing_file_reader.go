package stream

import (
	"errors"
	"io"
	"os"
	"sync"
	"time"
)

// GrowingFileReader reads from a file that is still being appended to.
// It blocks on EOF while the writer is still active.
type GrowingFileReader struct {
	file *os.File
	done <-chan error

	pollInterval time.Duration

	mu         sync.Mutex
	writerDone bool
	writerErr  error
	closeOnce  sync.Once
}

func NewGrowingFileReader(file *os.File, done <-chan error) *GrowingFileReader {
	return &GrowingFileReader{
		file:         file,
		done:         done,
		pollInterval: 10 * time.Millisecond,
	}
}

func (r *GrowingFileReader) Read(p []byte) (int, error) {
	for {
		r.mu.Lock()
		n, err := r.file.Read(p)
		writerDone := r.writerDone
		writerErr := r.writerErr
		r.mu.Unlock()

		if n > 0 {
			return n, nil
		}
		if err == nil {
			continue
		}
		if !errors.Is(err, io.EOF) {
			return 0, err
		}

		if writerDone {
			if writerErr != nil {
				return 0, writerErr
			}
			return 0, io.EOF
		}

		select {
		case err, ok := <-r.done:
			r.mu.Lock()
			if ok {
				r.writerErr = err
			}
			r.writerDone = true
			r.mu.Unlock()
		case <-time.After(r.pollInterval):
		}
	}
}

func (r *GrowingFileReader) Seek(offset int64, whence int) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.file.Seek(offset, whence)
}

func (r *GrowingFileReader) Close() error {
	var closeErr error
	r.closeOnce.Do(func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		closeErr = r.file.Close()
	})
	return closeErr
}

