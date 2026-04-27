package service

import (
	"context"
	"delivery/internal/domain"
	"errors"
	"fmt"
)

type OrderDeliveryRepository interface {
	Create(ctx context.Context, d *domain.OrderDelivery) error
	GetByID(ctx context.Context, orderID int64) (*domain.OrderDelivery, error)
	GetNextReadyInStatus(ctx context.Context, status domain.DeliveryStatus, intervalSec int) (*domain.OrderDelivery, error)
	UpdateStatus(ctx context.Context, d *domain.OrderDelivery, maxAttempts int) error
}

type OrdersClient interface {
	CompleteOrder(ctx context.Context, orderID int64) error
}

type Config struct {
	MaxAttempts    int
	IntervalSec    int
}

type OrderDeliveryService struct {
	repo         OrderDeliveryRepository
	ordersClient OrdersClient
	cfg          Config
}

func NewOrderDeliveryService(repo OrderDeliveryRepository, ordersClient OrdersClient, cfg Config) *OrderDeliveryService {
	return &OrderDeliveryService{repo: repo, ordersClient: ordersClient, cfg: cfg}
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

// CompleteNextOrder выбирает следующую доставку в статусе delivered, завершает заказ
// и переводит доставку в статус done.
// Возвращает (true, nil) если доставка обработана, (false, nil) если нечего обрабатывать.
func (s *OrderDeliveryService) CompleteNextOrder(ctx context.Context) (bool, error) {
	delivery, err := s.repo.GetNextReadyInStatus(ctx, domain.DeliveryStatusDelivered, s.cfg.IntervalSec)
	if err != nil {
		if errors.Is(err, domain.ErrNoOrderDeliveryToProcess) {
			return false, nil
		}
		return false, fmt.Errorf("get next order delivery: %w", err)
	}

	if err := s.ordersClient.CompleteOrder(ctx, delivery.OrderID); err != nil {
		return false, fmt.Errorf("complete order: %w", err)
	}

	if err := delivery.SetDone(); err != nil {
		return false, fmt.Errorf("set done: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, delivery, 0); err != nil {
		return false, fmt.Errorf("update status: %w", err)
	}
	return true, nil
}
