package ports

import (
	"Jabba_The_Bot/internal/core/app/dto"
	"context"
)

type MusicSearchProvider interface {
	Search(ctx context.Context, query string, limit int) ([]dto.SearchResult, error)
}
