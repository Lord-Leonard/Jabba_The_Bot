package youtube

import (
	"Jabba_The_Bot/internal/music/domain"
	"context"
)

func (p *Provider) Resolve(ctx context.Context, querry string) (domain.Track, error) {
	tracks, err := p.Search(ctx, querry)
	if err != nil || len(tracks) == 0 {
		return domain.Track{}, err
	}

	return tracks[0], err
}
