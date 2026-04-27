package service

import (
	"context"
	"fmt"
	"product-management/internal/domain"
)

type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) error
	UpdateStatus(ctx context.Context, order *domain.Order, maxAttempts int) error
}

type ReservationItem struct {
	ProductID int64
	Amount    int
}

type OrderService struct {
	repo OrderRepository
}

func NewOrderService(repo OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) Create(ctx context.Context, orderID int64, _ []ReservationItem) error {
	order, err := domain.NewOrder(orderID)
	if err != nil {
		return fmt.Errorf("new order: %w", err)
	}
	if err := s.repo.Create(ctx, order); err != nil {
		return fmt.Errorf("create in repo: %w", err)
	}
	return nil
}
