-- +goose Up
CREATE TABLE outbox (
    id BIGINT NOT NULL,
    type TEXT NOT NULL,
    key TEXT NOT NULL,
    payload TEXT NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at TIMESTAMPTZ NULL
);
-- +goose Down
DROP TABLE outbox;