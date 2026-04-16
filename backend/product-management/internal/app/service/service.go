package service

import (
	"context"
	"fmt"
	"product-management/internal/app/domain"
	"product-management/internal/pkg/snowflake"
	"strconv"
)

type ProductService struct {
	productView domain.ProductView
	snowflake   *snowflake.Snowflake
	uowFactory  domain.UnitOfWorkFactory
}

func NewProductService(
	view domain.ProductView,
	snowflake *snowflake.Snowflake,
	uowFactory domain.UnitOfWorkFactory,
) *ProductService {
	return &ProductService{
		productView: view,
		snowflake:   snowflake,
		uowFactory:  uowFactory,
	}
}

func (s *ProductService) Create(ctx context.Context, userID int64, name string, price int64) (*domain.Product, error) {
	product, err := domain.NewProduct(s.snowflake.NextID(), userID, name, price)
	if err != nil {
		return nil, fmt.Errorf("new product: %w", err)
	}

	uow, err := s.uowFactory.ForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("create unit of work: %w", err)
	}
	defer uow.Rollback()

	if err := uow.ProductRepository().Create(ctx, product); err != nil {
		return nil, fmt.Errorf("create in repo: %w", err)
	}
	payload := domain.ProductEvent{Type: domain.ProductEventTypeCreated, Product: product}
	event := &domain.OutboxEvent{
		ID:      s.snowflake.NextID(),
		Type:    domain.ProductCreated,
		Key:     strconv.FormatInt(product.ID, 10),
		Payload: payload.ToJSON(),
	}
	if err = uow.OutboxRepository().Create(ctx, event); err != nil {
		return nil, fmt.Errorf("create outbox event: %w", err)
	}
	if err = uow.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	return product, nil
}

func (s *ProductService) GetByID(ctx context.Context, userID int64, id int64) (*domain.Product, error) {
	return s.productView.GetByID(ctx, id, userID)
}

func (s *ProductService) List(ctx context.Context, userID int64) ([]domain.Product, error) {
	return s.productView.List(ctx, userID)
}
