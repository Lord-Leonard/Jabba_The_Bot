package service

import (
	"Jabba_The_Bot/internal/core/app/dto"
	"Jabba_The_Bot/internal/core/app/ports"
	"context"
	"strings"
)

type SearchService struct {
	searchProvider ports.MusicSearchProvider
	defaultLimit   int
	maxLimit       int
}

func NewSearchService(searchProvider ports.MusicSearchProvider) *SearchService {
	return &SearchService{
		searchProvider: searchProvider,
		defaultLimit:   5,
		maxLimit:       20,
	}
}

func (s *SearchService) Search(ctx context.Context, query string, limit int) ([]dto.SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []dto.SearchResult{}, nil
	}

	if limit <= 0 {
		limit = s.defaultLimit
	}
	if limit > s.maxLimit {
		limit = s.maxLimit
	}

	return s.searchProvider.Search(ctx, query, limit)
}
