package service

import (
	"context"
	"delivery/internal/domain"
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-chi/chi/v5/middleware"
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

type PaymentsClient interface {
	CancelPayment(ctx context.Context, orderID int64) error
}

type Config struct {
	MaxAttempts    int
	IntervalSec    int
	FailingIntervalSec int
}

type OrderDeliveryService struct {
	repo           OrderDeliveryRepository
	ordersClient   OrdersClient
	paymentsClient PaymentsClient
	cfg            Config
}

func NewOrderDeliveryService(repo OrderDeliveryRepository, ordersClient OrdersClient, paymentsClient PaymentsClient, cfg Config) *OrderDeliveryService {
	return &OrderDeliveryService{repo: repo, ordersClient: ordersClient, paymentsClient: paymentsClient, cfg: cfg}
}

func (s *OrderDeliveryService) Create(ctx context.Context, orderID int64) error {
	delivery := domain.NewOrderDelivery(orderID)
	if err := s.repo.Create(ctx, delivery); err != nil {
		return fmt.Errorf("create order delivery: %w", err)
	}
	getLogger(ctx, "order_id", orderID).Info("delivery request created")
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
	if err := s.repo.UpdateStatus(ctx, delivery, 10); err != nil { // TODO move max_attempts to config
		return fmt.Errorf("update status: %w", err)
	}
	getLogger(ctx, "order_id", delivery.OrderID).Info("order delivered successfully")
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
	if err := s.repo.UpdateStatus(ctx, delivery, 10); err != nil { // TODO move max_attempts to config
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

	l := getLogger(ctx, "order_id", delivery.OrderID)
	l.Info("completing order")
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

// FailNextOrder выбирает следующую доставку в статусе failing, отменяет платёж
// и переводит доставку в статус failed.
// Возвращает (true, nil) если доставка обработана, (false, nil) если нечего обрабатывать.
func (s *OrderDeliveryService) FailNextOrder(ctx context.Context) (bool, error) {
	delivery, err := s.repo.GetNextReadyInStatus(ctx, domain.DeliveryStatusFailing, s.cfg.FailingIntervalSec)
	if err != nil {
		if errors.Is(err, domain.ErrNoOrderDeliveryToProcess) {
			return false, nil
		}
		return false, fmt.Errorf("get next failing delivery: %w", err)
	}

	l := getLogger(ctx, "order_id", delivery.OrderID)
	l.Info("failing delivery, canceling payment")

	if err := s.paymentsClient.CancelPayment(ctx, delivery.OrderID); err != nil {
		l.Error("payment cancel failed", "error", err)
		return false, fmt.Errorf("cancel payment: %w", err)
	}

	if err := delivery.SetFailed(); err != nil {
		return false, fmt.Errorf("set failed: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, delivery, 0); err != nil {
		return false, fmt.Errorf("update status: %w", err)
	}
	l.Info("delivery failed")
	return true, nil
}

func getLogger(ctx context.Context, fields ...any) *slog.Logger {
	l := slog.Default().With("x_request_id", middleware.GetReqID(ctx))
	return l.With(fields...)
}
