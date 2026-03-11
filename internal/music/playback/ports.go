package playback

import (
	"Jabba_The_Bot/internal/music/domain"
	"context"
)

type TrackProvider interface {
	Next(ctx context.Context) (*domain.Track, error) // blocks or returns ErrQueueEmpty
}

type AudioProvider interface {
	Provide(ctx context.Context, track *domain.Track) (AudioStream, error)
}

type AudioSink interface {
	Consume(stream AudioStream) error
}
