package service

import (
	"context"
	"fmt"
	"payments/internal/domain"
)

type OrderPaymentRepository interface {
	Create(ctx context.Context, p *domain.OrderPayment) error
	UpdateStatus(ctx context.Context, p *domain.OrderPayment, maxAttempts int) error
}

type OrderPaymentService struct {
	repo OrderPaymentRepository
}

func NewOrderPaymentService(repo OrderPaymentRepository) *OrderPaymentService {
	return &OrderPaymentService{repo: repo}
}

func (s *OrderPaymentService) Create(ctx context.Context, orderID int64, amount float64) error {
	payment := domain.NewOrderPayment(orderID)
	if err := s.repo.Create(ctx, payment); err != nil {
		return fmt.Errorf("create order payment: %w", err)
	}
	return nil
}
