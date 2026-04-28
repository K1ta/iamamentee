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

func (r *OrderDeliveryRepository) GetNextReadyInStatus(ctx context.Context, status domain.DeliveryStatus, intervalSec int) (*domain.OrderDelivery, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	const query = `
		WITH locked AS (
			SELECT order_id FROM order_deliveries
			WHERE status = $1
			  AND (attempts < max_attempts OR max_attempts = -1)
			  AND next_attempt_after <= now()
			ORDER BY next_attempt_after
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE order_deliveries
		SET attempts = attempts + 1,
		    next_attempt_after = now() + ($2 * interval '1 second')
		WHERE order_id IN (SELECT order_id FROM locked)
		RETURNING order_id, status`

	var d domain.OrderDelivery
	row := tx.QueryRowContext(ctx, query, status, intervalSec)
	if err := row.Scan(&d.OrderID, &d.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNoOrderDeliveryToProcess
		}
		return nil, fmt.Errorf("scan order_delivery: %w", err)
	}
	return &d, tx.Commit()
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
