package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"payments/internal/domain"
)

type OrderPaymentRepository struct {
	db *sql.DB
}

func NewOrderPaymentRepository(db *sql.DB) *OrderPaymentRepository {
	return &OrderPaymentRepository{db: db}
}

func (r *OrderPaymentRepository) Create(ctx context.Context, p *domain.OrderPayment) error {
	const query = `INSERT INTO order_payments (order_id, status) VALUES ($1, $2)`
	_, err := r.db.ExecContext(ctx, query, p.OrderID, p.Status)
	if err != nil {
		return fmt.Errorf("insert order_payment: %w", err)
	}
	return nil
}

// UpdateStatus сбрасывает attempts, устанавливает max_attempts и next_attempt_after=now().
func (r *OrderPaymentRepository) UpdateStatus(ctx context.Context, p *domain.OrderPayment, maxAttempts int) error {
	const query = `
		UPDATE order_payments
		SET status = $1, attempts = 0, max_attempts = $2, next_attempt_after = now()
		WHERE order_id = $3`
	_, err := r.db.ExecContext(ctx, query, p.Status, maxAttempts, p.OrderID)
	if err != nil {
		return fmt.Errorf("update order_payment status: %w", err)
	}
	return nil
}
