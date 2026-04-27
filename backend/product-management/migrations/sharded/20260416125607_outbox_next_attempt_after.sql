-- +goose Up
ALTER TABLE outbox
ADD COLUMN IF NOT EXISTS next_attempt_after TIMESTAMPTZ NOT NULL DEFAULT now();
-- +goose Down
ALTER TABLE outbox DROP COLUMN IF EXISTS next_attempt_after;