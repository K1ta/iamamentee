package postgres

import (
	"context"
	"database/sql"
	"delivery/internal/domain"
	"errors"
	"fmt"
)

type OrderDeliveryRepository struct {
	db *sql.DB
}

func NewOrderDeliveryRepository(db *sql.DB) *OrderDeliveryRepository {
	return &OrderDeliveryRepository{db: db}
}

func (r *OrderDeliveryRepository) Create(ctx context.Context, d *domain.OrderDelivery) error {
	const query = `INSERT INTO order_deliveries (order_id, status) VALUES ($1, $2)`
	_, err := r.db.ExecContext(ctx, query, d.OrderID, d.Status)
	if err != nil {
		return fmt.Errorf("insert order_delivery: %w", err)
	}
	return nil
}

func (r *OrderDeliveryRepository) GetByID(ctx context.Context, orderID int64) (*domain.OrderDelivery, error) {
	const query = `SELECT order_id, status FROM order_deliveries WHERE order_id = $1`
	row := r.db.QueryRowContext(ctx, query, orderID)
	var d domain.OrderDelivery
	if err := row.Scan(&d.OrderID, &d.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrOrderDeliveryNotFound
		}
		return nil, fmt.Errorf("scan order_delivery: %w", err)
	}
	return &d, nil
}

// UpdateStatus сбрасывает attempts, устанавливает max_attempts и next_attempt_after=now().
func (r *OrderDeliveryRepository) UpdateStatus(ctx context.Context, d *domain.OrderDelivery, maxAttempts int) error {
	const query = `
		UPDATE order_deliveries
		SET status = $1, attempts = 0, max_attempts = $2, next_attempt_after = now()
		WHERE order_id = $3`
	_, err := r.db.ExecContext(ctx, query, d.Status, maxAttempts, d.OrderID)
	if err != nil {
		return fmt.Errorf("update order_delivery status: %w", err)
	}
	return nil
}
