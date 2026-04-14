package postgres

import (
	"context"
	"database/sql"
	"product-management/internal/app/models"
)

type OutboxRepository struct {
	tx          *sql.Tx
	maxAttempts int
}

func NewOutboxRepository(tx *sql.Tx, maxAttempts int) *OutboxRepository {
	return &OutboxRepository{tx: tx, maxAttempts: maxAttempts}
}

func (r *OutboxRepository) Create(ctx context.Context, event *models.OutboxEvent) error {
	query := "INSERT INTO outbox(id, type, key, payload, max_attempts) VALUES ($1, $2, $3, $4, $5)"
	_, err := r.tx.ExecContext(ctx, query, event.ID, event.Type, event.Key, event.Payload, r.maxAttempts)
	return err
}

func (r *OutboxRepository) SelectOneToSend(ctx context.Context) (*models.OutboxEvent, error) {
	query := "SELECT id, type, key, payload FROM outbox WHERE " +
		"sent_at IS NULL AND attempts < max_attempts ORDER BY created_at LIMIT 1 FOR UPDATE"
	row := r.tx.QueryRowContext(ctx, query)
	var event models.OutboxEvent
	if err := row.Scan(&event.ID, &event.Type, &event.Key, &event.Payload); err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *OutboxRepository) MarkAsSent(ctx context.Context, eventID int64) error {
	query := "UPDATE outbox SET sent_at=now() WHERE id=$1"
	_, err := r.tx.ExecContext(ctx, query, eventID)
	return err
}

func (r *OutboxRepository) IncreaseAttemts(ctx context.Context, eventID int64) error {
	query := "UPDATE outbox SET attempts=attempts+1 WHERE id=$1"
	_, err := r.tx.ExecContext(ctx, query, eventID)
	return err
}
