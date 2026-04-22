package service

import (
	"context"
	"fmt"
	"orders/internal/domain"
)

type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order, maxAttempts int) error
	GetByID(ctx context.Context, id int64) (*domain.Order, error)
	UpdateStatus(ctx context.Context, order *domain.Order, prevStatus domain.Status, maxAttempts int) error
	UpdateStatusAndSetPrices(ctx context.Context, order *domain.Order, prevStatus domain.Status, maxAttempts int) error
}

type ProductManagementClient interface {
	GetProductPrices(ctx context.Context, productIDs []int64) (map[int64]int64, error)
}

type StorageClient interface {
	CreateReservation(ctx context.Context, order *domain.Order) error
}

type AttemptsConfig struct {
	Created   int
	Confirmed int
}

type OrderService struct {
	repo       OrderRepository
	pmClient   ProductManagementClient
	storClient StorageClient
	attempts   AttemptsConfig
}

func NewOrderService(
	repo OrderRepository,
	pmClient ProductManagementClient,
	storClient StorageClient,
	attempts AttemptsConfig,
) *OrderService {
	return &OrderService{
		repo:       repo,
		pmClient:   pmClient,
		storClient: storClient,
		attempts:   attempts,
	}
}

func (s *OrderService) maxAttemptsByStatus(status domain.Status) int {
	switch status {
	case domain.StatusCreated:
		return s.attempts.Created
	case domain.StatusConfirmed:
		return s.attempts.Confirmed
	default:
		return 0
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
	if err := s.repo.Create(ctx, order, s.maxAttemptsByStatus(order.Status)); err != nil {
		return nil, fmt.Errorf("create in repo: %w", err)
	}
	return order, nil
}

func (s *OrderService) Confirm(ctx context.Context, orderID int64) error {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	prevOrderStatus := order.Status

	productIDs := make([]int64, len(order.Items))
	for i, item := range order.Items {
		productIDs[i] = item.ProductID
	}

	prices, err := s.pmClient.GetProductPrices(ctx, productIDs)
	if err != nil {
		return fmt.Errorf("get product prices: %w", err)
	}

	if err := order.Confirm(prices); err != nil {
		return fmt.Errorf("confirm order: %w", err)
	}

	if err := s.repo.UpdateStatusAndSetPrices(ctx, order, prevOrderStatus, s.maxAttemptsByStatus(order.Status)); err != nil {
		return fmt.Errorf("update status and prices: %w", err)
	}
	return nil
}

func (s *OrderService) StartProcessing(ctx context.Context, orderID int64) error {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	prevOrderStatus := order.Status

	if err := order.SetProcessing(); err != nil {
		return fmt.Errorf("set processing: %w", err)
	}

	if err := s.storClient.CreateReservation(ctx, order); err != nil {
		return fmt.Errorf("create reservation: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, order, prevOrderStatus, s.maxAttemptsByStatus(order.Status)); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

func (s *OrderService) Complete(ctx context.Context, orderID int64) error {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	prevOrderStatus := order.Status

	if err := order.SetCompleted(); err != nil {
		return fmt.Errorf("set completed: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, order, prevOrderStatus, s.maxAttemptsByStatus(order.Status)); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

func (s *OrderService) Cancel(ctx context.Context, orderID int64) error {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	prevOrderStatus := order.Status

	if err := order.SetCanceled(); err != nil {
		return fmt.Errorf("set canceled: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, order, prevOrderStatus, s.maxAttemptsByStatus(order.Status)); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

func (s *OrderService) Fail(ctx context.Context, orderID int64) error {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}
	prevOrderStatus := order.Status

	if err := order.SetFailed(); err != nil {
		return fmt.Errorf("set failed: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, order, prevOrderStatus, s.maxAttemptsByStatus(order.Status)); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}
