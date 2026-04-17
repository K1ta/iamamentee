package service

import (
	"context"
	"fmt"
	"products/internal/domain"
)

type productRepository interface {
	Create(ctx context.Context, product *domain.Product) error
	ListByIDs(ctx context.Context, ids []int64) ([]domain.Product, error)
}

type productIndex interface {
	Search(ctx context.Context, query domain.SearchQuery) ([]int64, error)
	Index(ctx context.Context, product *domain.Product) error
}

type ProductService struct {
	repo  productRepository
	index productIndex
}

func NewProductService(repo productRepository, index productIndex) *ProductService {
	return &ProductService{repo: repo, index: index}
}

func (s *ProductService) Search(ctx context.Context, query domain.SearchQuery) ([]domain.Product, error) {
	ids, err := s.index.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("search in index: %w", err)
	}
	if len(ids) == 0 {
		return []domain.Product{}, nil
	}
	products, err := s.repo.ListByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list by ids: %w", err)
	}
	return products, nil
}

func (s *ProductService) CreateProduct(ctx context.Context, product *domain.Product) error {
	if err := s.repo.Create(ctx, product); err != nil {
		return fmt.Errorf("create product %d: %w", product.ID, err)
	}
	if err := s.index.Index(ctx, product); err != nil {
		return fmt.Errorf("index product %d: %w", product.ID, err)
	}
	return nil
}
