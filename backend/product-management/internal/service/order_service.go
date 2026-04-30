package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"product-management/internal/domain"

	"github.com/go-chi/chi/v5/middleware"
)

type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order, maxAttempts int) error
	GetByID(ctx context.Context, id int64) (*domain.Order, error)
	UpdateStatus(ctx context.Context, order *domain.Order, maxAttempts int) error
	GetNextReadyInStatus(ctx context.Context, status domain.OrderStatus, intervalSec int) (*domain.Order, error)
}

type PaymentsClient interface {
	RequestPayment(ctx context.Context, orderID int64) error
}

type OrdersClient interface {
	Cancel(ctx context.Context, orderID int64) error
}

type ReservationItem struct {
	ProductID int64
	Amount    int
}

type OrderConfig struct {
	MaxAttempts             int
	ReservationIntervalSec  int
	PaymentIntervalSec      int
	CompensationIntervalSec int
	CancellationIntervalSec int
}

type OrderService struct {
	repo           OrderRepository
	paymentsClient PaymentsClient
	ordersClient   OrdersClient
	cfg            OrderConfig
}

func NewOrderService(repo OrderRepository, paymentsClient PaymentsClient, ordersClient OrdersClient, cfg OrderConfig) *OrderService {
	return &OrderService{repo: repo, paymentsClient: paymentsClient, ordersClient: ordersClient, cfg: cfg}
}

func (s *OrderService) Create(ctx context.Context, orderID int64, _ []ReservationItem) error {
	order, err := domain.NewOrder(orderID)
	if err != nil {
		return fmt.Errorf("new order: %w", err)
	}
	if err := s.repo.Create(ctx, order, s.cfg.MaxAttempts); err != nil {
		return fmt.Errorf("create in repo: %w", err)
	}
	getLogger(ctx, "order_id", orderID).Info("reservation request created")
	return nil
}

// RequestPaymentForNextOrder выбирает следующий заказ в статусе reserved, запрашивает оплату
// и переводит заказ в статус done.
// Возвращает (true, nil) если заказ был обработан, (false, nil) если нечего обрабатывать.
func (s *OrderService) RequestPaymentForNextOrder(ctx context.Context) (bool, error) {
	order, err := s.repo.GetNextReadyInStatus(ctx, domain.OrderStatusReserved, s.cfg.PaymentIntervalSec)
	if err != nil {
		if errors.Is(err, domain.ErrNoOrderFound) {
			return false, nil
		}
		return false, fmt.Errorf("get next for payment: %w", err)
	}

	l := getLogger(ctx, "order_id", order.ID)
	l.Info("requesting payment for order")

	if err := s.paymentsClient.RequestPayment(ctx, order.ID); err != nil {
		return false, fmt.Errorf("request payment: %w", err)
	}

	if err := order.SetDone(); err != nil {
		return false, fmt.Errorf("set done: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, order, 0); err != nil {
		return false, fmt.Errorf("update status: %w", err)
	}
	return true, nil
}

// ReserveNextOrder выбирает следующий заказ в статусе created и переводит его в reserved.
// Возвращает (true, nil) если заказ был обработан, (false, nil) если нечего обрабатывать.
func (s *OrderService) ReserveNextOrder(ctx context.Context) (bool, error) {
	order, err := s.repo.GetNextReadyInStatus(ctx, domain.OrderStatusCreated, s.cfg.ReservationIntervalSec)
	if err != nil {
		if errors.Is(err, domain.ErrNoOrderFound) {
			return false, nil
		}
		return false, fmt.Errorf("get next for reservation: %w", err)
	}

	l := getLogger(ctx, "order_id", order.ID)
	l.Info("reserving products")

	if err := order.SetReserved(); err != nil {
		return false, fmt.Errorf("set reserved: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, order, -1); err != nil {
		return false, fmt.Errorf("update status: %w", err)
	}
	l.Info("products reserved")
	return true, nil
}

// Cancel переводит резервацию заказа в статус compensating для начала компенсационного потока.
func (s *OrderService) Cancel(ctx context.Context, orderID int64) error {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	if err := order.SetCompensating(); err != nil {
		return fmt.Errorf("set compensating: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, order, -1); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	getLogger(ctx, "order_id", orderID).Info("reservation cancellation started")
	return nil
}

// CompensateNextOrder выбирает следующий заказ в статусе compensating и переводит его в compensated.
// Возвращает (true, nil) если заказ был обработан, (false, nil) если нечего обрабатывать.
func (s *OrderService) CompensateNextOrder(ctx context.Context) (bool, error) {
	order, err := s.repo.GetNextReadyInStatus(ctx, domain.OrderStatusCompensating, s.cfg.CompensationIntervalSec)
	if err != nil {
		if errors.Is(err, domain.ErrNoOrderFound) {
			return false, nil
		}
		return false, fmt.Errorf("get next for compensation: %w", err)
	}

	if err := order.SetCompensated(); err != nil {
		return false, fmt.Errorf("set compensated: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, order, -1); err != nil {
		return false, fmt.Errorf("update status: %w", err)
	}
	getLogger(ctx, "order_id", order.ID).Info("reservation compensated")
	return true, nil
}

// CancelNextOrder выбирает следующий заказ в статусе compensated, отправляет отмену в сервис заказов
// и переводит заказ в статус canceled.
// Возвращает (true, nil) если заказ был обработан, (false, nil) если нечего обрабатывать.
func (s *OrderService) CancelNextOrder(ctx context.Context) (bool, error) {
	order, err := s.repo.GetNextReadyInStatus(ctx, domain.OrderStatusCompensated, s.cfg.CancellationIntervalSec)
	if err != nil {
		if errors.Is(err, domain.ErrNoOrderFound) {
			return false, nil
		}
		return false, fmt.Errorf("get next for cancellation: %w", err)
	}

	l := getLogger(ctx, "order_id", order.ID)
	l.Info("canceling order")

	if err := s.ordersClient.Cancel(ctx, order.ID); err != nil {
		return false, fmt.Errorf("cancel order: %w", err)
	}

	if err := order.SetCanceled(); err != nil {
		return false, fmt.Errorf("set canceled: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, order, 0); err != nil {
		return false, fmt.Errorf("update status: %w", err)
	}
	l.Info("reservation canceled")
	return true, nil
}

func getLogger(ctx context.Context, fields ...any) *slog.Logger {
	l := slog.Default().With("x_request_id", middleware.GetReqID(ctx))
	return l.With(fields...)
}
