package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"orders/internal/domain"
)

type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order, maxAttempts int) error
	GetByID(ctx context.Context, id int64) (*domain.Order, error)
	UpdateStatus(ctx context.Context, order *domain.Order, prevStatus domain.Status, maxAttempts int) error
	GetOneForProcessing(ctx context.Context, status domain.Status, intervalSec int) (*domain.Order, error)
	GetOneExceededAttempts(ctx context.Context, statuses ...domain.Status) (*domain.Order, error)
}

type ProductManagementClient interface {
	GetProductPrices(ctx context.Context, items []domain.Item) (map[int64]int64, error)
	CreateReservation(ctx context.Context, order *domain.Order) error
}

type StatusConfig struct {
	MaxAttempts int
	IntervalSec int
}

type ProcessingConfig struct {
	Created StatusConfig
}

type OrderService struct {
	repo     OrderRepository
	pmClient ProductManagementClient
	cfg      ProcessingConfig
}

func NewOrderService(
	repo OrderRepository,
	pmClient ProductManagementClient,
	cfg ProcessingConfig,
) *OrderService {
	return &OrderService{
		repo:     repo,
		pmClient: pmClient,
		cfg:      cfg,
	}
}

func (s *OrderService) GetByID(ctx context.Context, orderID int64) (*domain.Order, error) {
	return s.repo.GetByID(ctx, orderID)
}

func (s *OrderService) Create(ctx context.Context, userID int64, items []domain.Item) (*domain.Order, error) {
	prices, err := s.pmClient.GetProductPrices(ctx, items)
	if err != nil {
		return nil, fmt.Errorf("get product prices: %w", err)
	}
	for i, item := range items {
		price, ok := prices[item.ProductID]
		if !ok {
			return nil, fmt.Errorf("price for product %d not found", item.ProductID)
		}
		items[i].Price = price
	}

	order, err := domain.NewOrder(userID, items)
	if err != nil {
		return nil, fmt.Errorf("new order: %w", err)
	}
	if err := s.repo.Create(ctx, order, s.cfg.Created.MaxAttempts); err != nil {
		return nil, fmt.Errorf("create in repo: %w", err)
	}
	log.Printf("order %d created", order.ID)
	return order, nil
}

// StartNextOrder выбирает следующий заказ в статусе created, создаёт резервацию
// в product-management и переводит заказ в статус processing.
// Возвращает (true, nil) если заказ был обработан, (false, nil) если нечего обрабатывать.
func (s *OrderService) StartNextOrder(ctx context.Context) (bool, error) {
	order, err := s.repo.GetOneForProcessing(ctx, domain.StatusCreated, s.cfg.Created.IntervalSec)
	if err != nil {
		if errors.Is(err, domain.ErrNoOrderToProcess) {
			return false, nil
		}
		return false, fmt.Errorf("get order for processing: %w", err)
	}
	prevStatus := order.Status

	log.Printf("reserving products for %d order", order.ID)
	if err := s.pmClient.CreateReservation(ctx, order); err != nil {
		return false, fmt.Errorf("create reservation: %w", err)
	}

	if err := order.SetProcessing(); err != nil {
		return false, fmt.Errorf("set processing: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, order, prevStatus, 0); err != nil {
		return false, fmt.Errorf("update status: %w", err)
	}
	log.Printf("products for %d order reserved", order.ID)
	return true, nil
}

// FailNextExhaustedOrder выбирает следующий заказ в статусе created, исчерпавший
// все попытки обработки, и переводит его в статус failed.
// Возвращает (true, nil) если заказ был обработан, (false, nil) если нечего обрабатывать.
func (s *OrderService) FailNextExhaustedOrder(ctx context.Context) (bool, error) {
	order, err := s.repo.GetOneExceededAttempts(ctx, domain.StatusCreated)
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

	if err := s.repo.UpdateStatus(ctx, order, prevStatus, 0); err != nil {
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

	if err := s.repo.UpdateStatus(ctx, order, prevStatus, 0); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	log.Printf("order %d completed", orderID)
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

	if err := s.repo.UpdateStatus(ctx, order, prevStatus, 0); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}
