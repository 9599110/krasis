package search

import (
	"context"
	"errors"
)

var ErrEmptyQuery = errors.New("搜索关键词不能为空")

type Service struct {
	repo *SearchRepository
}

func NewService(repo *SearchRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Search(ctx context.Context, query, searchType string, page, size int) ([]*SearchResult, int64, error) {
	if query == "" {
		return nil, 0, ErrEmptyQuery
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return s.repo.Search(ctx, query, searchType, page, size)
}
