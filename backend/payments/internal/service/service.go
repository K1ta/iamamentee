package service

import (
	"context"
	"fmt"
	"payments/internal/domain"
)

type OrderPaymentRepository interface {
	Create(ctx context.Context, p *domain.OrderPayment) error
	GetByID(ctx context.Context, orderID int64) (*domain.OrderPayment, error)
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

func (s *OrderPaymentService) MockSuccess(ctx context.Context, orderID int64) error {
	payment, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order payment: %w", err)
	}
	if err := payment.SetPaid(); err != nil {
		return fmt.Errorf("set paid: %w", err)
	}
	if err := s.repo.UpdateStatus(ctx, payment, 0); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

func (s *OrderPaymentService) MockFail(ctx context.Context, orderID int64) error {
	payment, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order payment: %w", err)
	}
	if err := payment.SetFailing(); err != nil {
		return fmt.Errorf("set failing: %w", err)
	}
	if err := s.repo.UpdateStatus(ctx, payment, 0); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}
