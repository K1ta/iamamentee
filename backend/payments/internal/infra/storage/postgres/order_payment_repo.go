package postgres

import (
	"context"
	"database/sql"
	"errors"
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

func (r *OrderPaymentRepository) GetNextReadyInStatus(ctx context.Context, status domain.PaymentStatus, intervalSec int) (*domain.OrderPayment, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	const query = `
		WITH locked AS (
			SELECT order_id FROM order_payments
			WHERE status = $1
			  AND (attempts < max_attempts OR max_attempts = -1)
			  AND next_attempt_after <= now()
			ORDER BY next_attempt_after
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE order_payments
		SET attempts = attempts + 1,
		    next_attempt_after = now() + ($2 * interval '1 second')
		WHERE order_id IN (SELECT order_id FROM locked)
		RETURNING order_id, status`

	var p domain.OrderPayment
	row := tx.QueryRowContext(ctx, query, status, intervalSec)
	if err := row.Scan(&p.OrderID, &p.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNoOrderPaymentToProcess
		}
		return nil, fmt.Errorf("scan order_payment: %w", err)
	}
	return &p, tx.Commit()
}

func (r *OrderPaymentRepository) GetByID(ctx context.Context, orderID int64) (*domain.OrderPayment, error) {
	const query = `SELECT order_id, status FROM order_payments WHERE order_id = $1`
	row := r.db.QueryRowContext(ctx, query, orderID)
	var p domain.OrderPayment
	if err := row.Scan(&p.OrderID, &p.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrOrderPaymentNotFound
		}
		return nil, fmt.Errorf("scan order_payment: %w", err)
	}
	return &p, nil
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
