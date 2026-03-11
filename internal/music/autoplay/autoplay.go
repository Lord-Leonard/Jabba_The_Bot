package autoplay

import (
	"Jabba_The_Bot/internal/music/domain"
	"context"
)

type RecommendationProvider interface {
	Recommend(ctx context.Context, seeds []domain.WeightedSeed) ([]domain.Track, error)
}
