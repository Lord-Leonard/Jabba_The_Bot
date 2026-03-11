package youtube

import (
	"Jabba_The_Bot/internal/music/domain"
	"context"
	"strings"
)

func (p *Provider) Search(ctx context.Context, query string) ([]domain.Track, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []domain.Track{}, nil
	}

	results, err := p.client.Search(ctx, query, "EgWKAQIIAWoSEAQQAxAFEAkQEBAKEA4QFRAR") // TODO: thats for sure not the right way or place or place to do it ...
	if err != nil {
		return nil, err
	}

	tracks := make([]domain.Track, 0, len(results))
	for _, res := range results {
		if strings.TrimSpace(res.VideoID) == "" {
			continue
		}
		tracks = append(tracks, domain.Track{
			Title:    res.Title,
			Artist:   res.Artist,
			Album:    res.Album,
			VideoID:  res.VideoID,
			CoverUrl: res.CoverArtUrl,
		})
	}

	return tracks, nil
}
