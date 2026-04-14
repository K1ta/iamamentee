package service

import (
	"context"
	"fmt"
	"product-management/internal/app/models"
	"product-management/internal/infra/storage/postgres"
	"product-management/internal/pkg/snowflake"
	"strconv"
)

type ProductService struct {
	repo       ProductRepository
	snowflake  *snowflake.Snowflake
	uowManager *postgres.UnitOfWorkManager
}

type (
	ProductRepository interface {
		Create(ctx context.Context, product *models.Product) error
		GetByID(ctx context.Context, id, userID int64) (*models.Product, error)
		List(ctx context.Context, userID int64) ([]models.Product, error)
	}
)

func NewProductService(
	repo ProductRepository,
	snowflake *snowflake.Snowflake,
	uowManager *postgres.UnitOfWorkManager,
) *ProductService {
	return &ProductService{
		repo:       repo,
		snowflake:  snowflake,
		uowManager: uowManager,
	}
}

func (s *ProductService) Create(ctx context.Context, userID int64, name string, price int64) (*models.Product, error) {
	product, err := models.NewProduct(s.snowflake.NextID(), userID, name, price)
	if err != nil {
		return nil, fmt.Errorf("new product: %w", err)
	}

	s.uowManager.RunForUser(ctx, userID, func(uow *postgres.UnitOfWork) error {
		if err = uow.CreateProduct(ctx, product); err != nil {
			return fmt.Errorf("create product in repo: %w", err)
		}
		payload := models.ProductEvent{Type: models.ProductEventTypeCreated, Product: product}
		event := &models.OutboxEvent{
			ID:      s.snowflake.NextID(),
			Type:    models.ProductCreated,
			Key:     strconv.FormatInt(product.ID, 10),
			Payload: payload.ToJSON(),
		}
		if err = uow.CreateOutboxEvent(ctx, event); err != nil {
			return fmt.Errorf("produce product created event: %w", err)
		}
		return nil
	})

	return product, nil
}

func (s *ProductService) GetByID(ctx context.Context, userID int64, id int64) (*models.Product, error) {
	return s.repo.GetByID(ctx, id, userID)
}

func (s *ProductService) List(ctx context.Context, userID int64) ([]models.Product, error) {
	return s.repo.List(ctx, userID)
}
