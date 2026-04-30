package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"product-management/internal/domain"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(ctx context.Context, order *domain.Order, maxAttempts int) error {
	const query = `INSERT INTO orders (id, status, max_attempts) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, query, order.ID, order.Status, maxAttempts)
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	return nil
}

func (r *OrderRepository) GetNextReadyInStatus(ctx context.Context, status domain.OrderStatus, intervalSec int) (*domain.Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	const query = `
		WITH locked AS (
			SELECT id FROM orders
			WHERE status = $1
			  AND (attempts < max_attempts OR max_attempts = -1)
			  AND next_attempt_after <= now()
			ORDER BY next_attempt_after
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE orders
		SET attempts = attempts + 1,
		    next_attempt_after = now() + ($2 * interval '1 second')
		WHERE id IN (SELECT id FROM locked)
		RETURNING id, status`

	var order domain.Order
	row := tx.QueryRowContext(ctx, query, status, intervalSec)
	if err := row.Scan(&order.ID, &order.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNoOrderFound
		}
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &order, tx.Commit()
}

func (r *OrderRepository) GetByID(ctx context.Context, id int64) (*domain.Order, error) {
	const query = `SELECT id, status FROM orders WHERE id = $1`
	var order domain.Order
	row := r.db.QueryRowContext(ctx, query, id)
	if err := row.Scan(&order.ID, &order.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNoOrderFound
		}
		return nil, fmt.Errorf("scan: %w", err)
	}
	return &order, nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, order *domain.Order, maxAttempts int) error {
	const query = `
		UPDATE orders
		SET status = $1, attempts = 0, max_attempts = $2, next_attempt_after = now()
		WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, order.Status, maxAttempts, order.ID)
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	return nil
}
