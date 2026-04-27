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

func (s *OrderDeliveryService) MockSuccess(ctx context.Context, orderID int64) error {
	delivery, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order delivery: %w", err)
	}
	if err := delivery.SetDelivered(); err != nil {
		return fmt.Errorf("set delivered: %w", err)
	}
	if err := s.repo.UpdateStatus(ctx, delivery, 0); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

func (s *OrderDeliveryService) MockFail(ctx context.Context, orderID int64) error {
	delivery, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order delivery: %w", err)
	}
	if err := delivery.SetFailing(); err != nil {
		return fmt.Errorf("set failing: %w", err)
	}
	if err := s.repo.UpdateStatus(ctx, delivery, 0); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}
