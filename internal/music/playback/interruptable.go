package playback

import (
	"fmt"
	"io"
	"sync"
	"time"
)

type interruptibleStream struct {
	source AudioStream
	stop   <-chan struct{}
	once   sync.Once
}

func newInterruptibleStream(source AudioStream, stop <-chan struct{}) AudioStream {
	return &interruptibleStream{
		source: source,
		stop:   stop,
	}
}

func (s *interruptibleStream) ProvideFrame() ([]byte, error) {
	select {
	case <-s.stop:
		s.once.Do(func() {
			if c, ok := s.source.(interface{ Close() error }); ok {
				_ = c.Close()
			}
		})
		return nil, io.EOF
	default:
	}
	return s.source.ProvideFrame()
}

func (s *interruptibleStream) Duration() time.Duration {
	if timed, ok := s.source.(TimedAudioStream); ok {
		return timed.Duration()
	}
	return 0
}

func (s *interruptibleStream) Position() time.Duration {
	if timed, ok := s.source.(TimedAudioStream); ok {
		return timed.Position()
	}
	return 0
}

func (s *interruptibleStream) Seek(target time.Duration) error {
	seekable, ok := s.source.(SeekableAudioStream)
	if !ok {
		return fmt.Errorf("underlying stream is not seekable")
	}
	return seekable.Seek(target)
}
