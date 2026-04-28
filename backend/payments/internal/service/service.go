package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"payments/internal/domain"

	"github.com/go-chi/chi/v5/middleware"
)

type OrderPaymentRepository interface {
	Create(ctx context.Context, p *domain.OrderPayment) error
	GetByID(ctx context.Context, orderID int64) (*domain.OrderPayment, error)
	GetNextReadyInStatus(ctx context.Context, status domain.PaymentStatus, intervalSec int) (*domain.OrderPayment, error)
	UpdateStatus(ctx context.Context, p *domain.OrderPayment, maxAttempts int) error
}

type DeliveryClient interface {
	CreateDelivery(ctx context.Context, orderID int64) error
}

type DeliveryWorkerConfig struct {
	IntervalSec int
}

type OrderPaymentService struct {
	repo           OrderPaymentRepository
	deliveryClient DeliveryClient
	cfg            DeliveryWorkerConfig
}

func NewOrderPaymentService(repo OrderPaymentRepository, deliveryClient DeliveryClient, cfg DeliveryWorkerConfig) *OrderPaymentService {
	return &OrderPaymentService{repo: repo, deliveryClient: deliveryClient, cfg: cfg}
}

func (s *OrderPaymentService) Create(ctx context.Context, orderID int64, amount float64) error {
	payment := domain.NewOrderPayment(orderID)
	if err := s.repo.Create(ctx, payment); err != nil {
		return fmt.Errorf("create order payment: %w", err)
	}
	getLogger(ctx, "order_id", payment.OrderID).Info("payment request created")
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
	if err := s.repo.UpdateStatus(ctx, payment, 10); err != nil { // TODO move max_attempts to config
		return fmt.Errorf("update status: %w", err)
	}
	getLogger(ctx, "order_id", payment.OrderID).Info("order paid successfully")
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
	if err := s.repo.UpdateStatus(ctx, payment, 10); err != nil { // TODO move max_attempts to config
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

// CreateDeliveryForNextOrder выбирает следующий платёж в статусе paid, создаёт доставку
// и переводит платёж в статус done.
// Возвращает (true, nil) если платёж обработан, (false, nil) если нечего обрабатывать.
func (s *OrderPaymentService) CreateDeliveryForNextOrder(ctx context.Context) (bool, error) {
	payment, err := s.repo.GetNextReadyInStatus(ctx, domain.PaymentStatusPaid, s.cfg.IntervalSec)
	if err != nil {
		if errors.Is(err, domain.ErrNoOrderPaymentToProcess) {
			return false, nil
		}
		return false, fmt.Errorf("get next order payment: %w", err)
	}

	l := getLogger(ctx, "order_id", payment.OrderID)
	l.Info("requesting delivery for order")

	if err := s.deliveryClient.CreateDelivery(ctx, payment.OrderID); err != nil {
		return false, fmt.Errorf("create delivery: %w", err)
	}

	if err := payment.SetDone(); err != nil {
		return false, fmt.Errorf("set done: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, payment, 0); err != nil {
		return false, fmt.Errorf("update status: %w", err)
	}
	return true, nil
}

func getLogger(ctx context.Context, fields ...any) *slog.Logger {
	l := slog.Default().With("x_request_id", middleware.GetReqID(ctx))
	return l.With(fields...)
}
