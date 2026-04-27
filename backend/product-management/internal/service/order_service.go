package service

import (
	"context"
	"errors"
	"fmt"
	"product-management/internal/domain"
)

type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order, maxAttempts int) error
	UpdateStatus(ctx context.Context, order *domain.Order, maxAttempts int) error
	GetNextForReservation(ctx context.Context, intervalSec int) (*domain.Order, error)
}

type ReservationItem struct {
	ProductID int64
	Amount    int
}

type OrderConfig struct {
	MaxAttempts int
	IntervalSec int
}

type OrderService struct {
	repo OrderRepository
	cfg  OrderConfig
}

func NewOrderService(repo OrderRepository, cfg OrderConfig) *OrderService {
	return &OrderService{repo: repo, cfg: cfg}
}

func (s *OrderService) Create(ctx context.Context, orderID int64, _ []ReservationItem) error {
	order, err := domain.NewOrder(orderID)
	if err != nil {
		return fmt.Errorf("new order: %w", err)
	}
	if err := s.repo.Create(ctx, order, s.cfg.MaxAttempts); err != nil {
		return fmt.Errorf("create in repo: %w", err)
	}
	return nil
}

// ReserveNextOrder выбирает следующий заказ в статусе created и переводит его в reserved.
// Возвращает (true, nil) если заказ был обработан, (false, nil) если нечего обрабатывать.
func (s *OrderService) ReserveNextOrder(ctx context.Context) (bool, error) {
	order, err := s.repo.GetNextForReservation(ctx, s.cfg.IntervalSec)
	if err != nil {
		if errors.Is(err, domain.ErrNoOrderForReservation) {
			return false, nil
		}
		return false, fmt.Errorf("get next for reservation: %w", err)
	}

	if err := order.SetReserved(); err != nil {
		return false, fmt.Errorf("set reserved: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, order, 0); err != nil {
		return false, fmt.Errorf("update status: %w", err)
	}
	return true, nil
}
