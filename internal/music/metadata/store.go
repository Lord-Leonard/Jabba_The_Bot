package metadata

import (
	"Jabba_The_Bot/internal/music/domain"
	"context"
)

type Store interface {
	Save(ctx context.Context, track domain.Track) error
	Get(ctx context.Context, videoID string) (domain.Track, error)
	SetCoverLocalPath(ctx context.Context, videoID string, path string) error
}
