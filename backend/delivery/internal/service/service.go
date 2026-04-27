package service

import (
	"context"
	"delivery/internal/domain"
	"fmt"
)

type OrderDeliveryRepository interface {
	Create(ctx context.Context, d *domain.OrderDelivery) error
	GetByID(ctx context.Context, orderID int64) (*domain.OrderDelivery, error)
	UpdateStatus(ctx context.Context, d *domain.OrderDelivery, maxAttempts int) error
}

type Config struct {
	MaxAttempts int
}

type OrderDeliveryService struct {
	repo OrderDeliveryRepository
	cfg  Config
}

func NewOrderDeliveryService(repo OrderDeliveryRepository, cfg Config) *OrderDeliveryService {
	return &OrderDeliveryService{repo: repo, cfg: cfg}
}

func (s *OrderDeliveryService) Create(ctx context.Context, orderID int64) error {
	delivery := domain.NewOrderDelivery(orderID)
	if err := s.repo.Create(ctx, delivery); err != nil {
		return fmt.Errorf("create order delivery: %w", err)
	}
	return nil
}
