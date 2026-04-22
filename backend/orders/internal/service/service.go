package service

import (
	"context"
	"errors"
	"fmt"
	"orders/internal/domain"
)

type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order, maxAttempts int) error
	GetByID(ctx context.Context, id int64) (*domain.Order, error)
	UpdateStatus(ctx context.Context, order *domain.Order, prevStatus domain.Status, maxAttempts int) error
	UpdateStatusAndSetPrices(ctx context.Context, order *domain.Order, prevStatus domain.Status, maxAttempts int) error
	GetOneForProcessing(ctx context.Context, status domain.Status, intervalSec int) (*domain.Order, error)
	GetOneExceededAttempts(ctx context.Context, statuses ...domain.Status) (*domain.Order, error)
}

type ProductManagementClient interface {
	GetProductPrices(ctx context.Context, productIDs []int64) (map[int64]int64, error)
}

type StorageClient interface {
	CreateReservation(ctx context.Context, order *domain.Order) error
}

type StatusConfig struct {
	MaxAttempts int
	IntervalSec int
}

type ProcessingConfig struct {
	Created   StatusConfig
	Confirmed StatusConfig
}

type OrderService struct {
	repo       OrderRepository
	pmClient   ProductManagementClient
	storClient StorageClient
	cfg        ProcessingConfig
}

func NewOrderService(
	repo OrderRepository,
	pmClient ProductManagementClient,
	storClient StorageClient,
	cfg ProcessingConfig,
) *OrderService {
	return &OrderService{
		repo:       repo,
		pmClient:   pmClient,
		storClient: storClient,
		cfg:        cfg,
	}
}

func (s *OrderService) statusConfig(status domain.Status) StatusConfig {
	switch status {
	case domain.StatusCreated:
		return s.cfg.Created
	case domain.StatusConfirmed:
		return s.cfg.Confirmed
	default:
		return StatusConfig{}
	}
}

func (s *OrderService) GetByID(ctx context.Context, orderID int64) (*domain.Order, error) {
	return s.repo.GetByID(ctx, orderID)
}

func (s *OrderService) Create(ctx context.Context, userID int64, items []domain.Item) (*domain.Order, error) {
	order, err := domain.NewOrder(userID, items)
	if err != nil {
		return nil, fmt.Errorf("new order: %w", err)
	}
	if err := s.repo.Create(ctx, order, s.statusConfig(order.Status).MaxAttempts); err != nil {
		return nil, fmt.Errorf("create in repo: %w", err)
	}
	return order, nil
}

// ConfirmNextOrder picks the next created order ready for processing, fetches prices,
// confirms the order, and persists the new status and prices.
// Returns (true, nil) if an order was processed, (false, nil) if there was nothing to do.
func (s *OrderService) ConfirmNextOrder(ctx context.Context) (bool, error) {
	cfg := s.cfg.Created
	order, err := s.repo.GetOneForProcessing(ctx, domain.StatusCreated, cfg.IntervalSec)
	if err != nil {
		if errors.Is(err, domain.ErrNoOrderToProcess) {
			return false, nil
		}
		return false, fmt.Errorf("get order for processing: %w", err)
	}
	prevStatus := order.Status

	productIDs := make([]int64, len(order.Items))
	for i, item := range order.Items {
		productIDs[i] = item.ProductID
	}

	prices, err := s.pmClient.GetProductPrices(ctx, productIDs)
	if err != nil {
		return false, fmt.Errorf("get product prices: %w", err)
	}

	if err := order.Confirm(prices); err != nil {
		return false, fmt.Errorf("confirm order: %w", err)
	}

	if err := s.repo.UpdateStatusAndSetPrices(ctx, order, prevStatus, s.statusConfig(order.Status).MaxAttempts); err != nil {
		return false, fmt.Errorf("update status and prices: %w", err)
	}
	return true, nil
}

// StartNextOrder picks the next confirmed order ready for processing, creates a storage
// reservation, and moves the order to processing status.
// Returns (true, nil) if an order was processed, (false, nil) if there was nothing to do.
func (s *OrderService) StartNextOrder(ctx context.Context) (bool, error) {
	cfg := s.cfg.Confirmed
	order, err := s.repo.GetOneForProcessing(ctx, domain.StatusConfirmed, cfg.IntervalSec)
	if err != nil {
		if errors.Is(err, domain.ErrNoOrderToProcess) {
			return false, nil
		}
		return false, fmt.Errorf("get order for processing: %w", err)
	}
	prevStatus := order.Status

	if err := order.SetProcessing(); err != nil {
		return false, fmt.Errorf("set processing: %w", err)
	}

	if err := s.storClient.CreateReservation(ctx, order); err != nil {
		return false, fmt.Errorf("create reservation: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, order, prevStatus, s.statusConfig(order.Status).MaxAttempts); err != nil {
		return false, fmt.Errorf("update status: %w", err)
	}
	return true, nil
}

// FailNextExhaustedOrder picks the next created or confirmed order that has exhausted
// all processing attempts and marks it as failed.
// Returns (true, nil) if an order was processed, (false, nil) if there was nothing to do.
func (s *OrderService) FailNextExhaustedOrder(ctx context.Context) (bool, error) {
	order, err := s.repo.GetOneExceededAttempts(ctx, domain.StatusCreated, domain.StatusConfirmed)
	if err != nil {
		if errors.Is(err, domain.ErrNoOrderToProcess) {
			return false, nil
		}
		return false, fmt.Errorf("get order with exceeded attempts: %w", err)
	}
	prevStatus := order.Status

	if err := order.SetFailed(); err != nil {
		return false, fmt.Errorf("set failed: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, order, prevStatus, s.statusConfig(order.Status).MaxAttempts); err != nil {
		return false, fmt.Errorf("update status: %w", err)
	}
	return true, nil
}

func (s *OrderService) Complete(ctx context.Context, orderID int64) error {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	prevStatus := order.Status

	if err := order.SetCompleted(); err != nil {
		return fmt.Errorf("set completed: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, order, prevStatus, s.statusConfig(order.Status).MaxAttempts); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

func (s *OrderService) Cancel(ctx context.Context, orderID int64) error {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	prevStatus := order.Status

	if err := order.SetCanceled(); err != nil {
		return fmt.Errorf("set canceled: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, order, prevStatus, s.statusConfig(order.Status).MaxAttempts); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}
