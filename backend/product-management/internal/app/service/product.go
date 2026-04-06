package service

import (
	"context"
	"fmt"
	"product-management/internal/app/models"
	"product-management/internal/pkg/snowflake"
)

type ProductService struct {
	repo      ProductRepository
	producer  MessageProducer
	snowflake *snowflake.Snowflake
}

type (
	ProductRepository interface {
		Create(ctx context.Context, product *models.Product) error
		GetByID(ctx context.Context, id, userID int64) (*models.Product, error)
		List(ctx context.Context, userID int64) ([]models.Product, error)
	}

	MessageProducer interface {
		ProduceEvent(ctx context.Context, eventType string, product *models.Product) error
	}
)

func NewProductService(repo ProductRepository, producer MessageProducer, snowflake *snowflake.Snowflake) *ProductService {
	return &ProductService{
		repo:      repo,
		producer:  producer,
		snowflake: snowflake,
	}
}

func (s *ProductService) Create(ctx context.Context, userID int64, name string, price int64) (*models.Product, error) {
	product, err := models.NewProduct(s.snowflake.NextID(), userID, name, price)
	if err != nil {
		return nil, fmt.Errorf("new product: %w", err)
	}
	if err = s.repo.Create(ctx, product); err != nil {
		return nil, fmt.Errorf("create product in repo: %w", err)
	}
	if err = s.producer.ProduceEvent(ctx, models.ProductEventTypeCreated, product); err != nil {
		return nil, fmt.Errorf("produce product created event: %w", err)
	}
	return product, nil
}

func (s *ProductService) GetByID(ctx context.Context, userID int64, id int64) (*models.Product, error) {
	return s.repo.GetByID(ctx, id, userID)
}

func (s *ProductService) List(ctx context.Context, userID int64) ([]models.Product, error) {
	return s.repo.List(ctx, userID)
}
