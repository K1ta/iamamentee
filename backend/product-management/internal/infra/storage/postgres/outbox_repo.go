package postgres

import (
	"context"
	"fmt"
	"product-management/internal/app/domain"

	"github.com/lib/pq"
)

type OutboxRepository struct {
	db          DBTX
	maxAttempts int
}

func NewOutboxRepository(db DBTX, maxAttempts int) *OutboxRepository {
	return &OutboxRepository{db: db, maxAttempts: maxAttempts}
}

func (r *OutboxRepository) Create(ctx context.Context, event *domain.OutboxEvent) error {
	query := "INSERT INTO outbox(id, type, key, payload, max_attempts) VALUES ($1, $2, $3, $4, $5)"
	_, err := r.db.ExecContext(ctx, query, event.ID, event.Type, event.Key, event.Payload, r.maxAttempts)
	return err
}

type OutboxProcessorRepository struct {
	db                 DBTX
	attemptDurationSec int
	batchLimit         int
}

func NewOutboxProcessorRepository(db DBTX, attemptDurationSec int, batchLimit int) *OutboxProcessorRepository {
	return &OutboxProcessorRepository{db: db, attemptDurationSec: attemptDurationSec, batchLimit: batchLimit}
}

func (r *OutboxProcessorRepository) SelectBatchToSend(ctx context.Context) ([]domain.OutboxEvent, error) {
	query := `
	WITH locked As (
		SELECT id FROM outbox
		WHERE sent_at IS NULL
			AND attempts < max_attempts
			AND next_attempt_after < now()
		ORDER BY created_at
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	)
	UPDATE outbox
	SET next_attempt_after=now()+($2 * interval '1 second'), attempts = attempts + 1
	WHERE id IN (SELECT id FROM locked)
	RETURNING id, type, key, payload;
	`
	rows, err := r.db.QueryContext(ctx, query, r.batchLimit, r.attemptDurationSec)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	events := make([]domain.OutboxEvent, 0)
	for rows.Next() {
		var event domain.OutboxEvent
		if err := rows.Scan(&event.ID, &event.Type, &event.Key, &event.Payload); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return events, nil
}

func (r *OutboxProcessorRepository) MarkBatchAsSent(ctx context.Context, ids []int64) error {
	query := "UPDATE outbox SET sent_at=now() WHERE id = ANY($1)"
	_, err := r.db.ExecContext(ctx, query, pq.Array(ids))
	return err
}
