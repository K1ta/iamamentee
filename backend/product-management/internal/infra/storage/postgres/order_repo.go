package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"product-management/internal/domain"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	const query = `INSERT INTO orders (id, status, max_attempts) VALUES ($1, $2, 0)`
	_, err := r.db.ExecContext(ctx, query, order.ID, order.Status)
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	return nil
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
