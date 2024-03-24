package music

import (
	"Jabba_The_Bot/internal/core/app/dto"
	"Jabba_The_Bot/internal/core/app/ports"
	youtubemusic "Jabba_The_Bot/pkg/youtube/music"
	"context"
	"strings"
)

var _ ports.MusicSearchProvider = (*YouTubeSearchProvider)(nil)

type YouTubeSearchProvider struct {
	client *youtubemusic.Client
}

func NewYouTubeSearchProvider(client *youtubemusic.Client) *YouTubeSearchProvider {
	if client == nil {
		client = youtubemusic.NewClient()
	}
	return &YouTubeSearchProvider{client: client}
}

func (p *YouTubeSearchProvider) Search(ctx context.Context, query string, limit int) ([]dto.SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []dto.SearchResult{}, nil
	}

	results, err := p.client.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	tracks := make([]dto.SearchResult, 0, len(results))
	for _, res := range results {
		if strings.TrimSpace(res.VideoID) == "" {
			continue
		}
		tracks = append(tracks, dto.SearchResult{
			Title:       res.Title,
			Artist:      res.Artist,
			VideoID:     res.VideoID,
			CoverArtURL: res.CoverArtUrl,
		})
	}

	return tracks, nil
}
