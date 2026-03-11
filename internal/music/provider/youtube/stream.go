package youtube

import (
	"Jabba_The_Bot/pkg/audio/webm"
	"io"
	"log/slog"
	"sync"
	"time"
)

type stream struct {
	demuxer *webm.Demuxer
	source  io.ReadCloser

	once sync.Once
}

func newStream(source io.ReadCloser, demuxLogger *slog.Logger) (*stream, error) {
	demuxer, err := webm.NewDemuxer(source, webm.WithDebugLogger(demuxLogger))
	if err != nil {
		return nil, err
	}

	return &stream{
		demuxer: demuxer,
		source:  source,
	}, nil
}

func (s *stream) ProvideFrame() ([]byte, error) {
	return s.demuxer.ProvideFrame()
}

func (s *stream) Duration() time.Duration {
	return s.demuxer.Duration()
}

func (s *stream) Position() time.Duration {
	return s.demuxer.Position()
}

func (s *stream) Seek(target time.Duration) error {
	return s.demuxer.Seek(target)
}

func (s *stream) Close() error {
	var err error
	s.once.Do(func() {
		err = s.source.Close()
	})
	return err
}
