-- +goose Up
CREATE TABLE orders (
    id                  BIGINT PRIMARY KEY,
    status              TEXT NOT NULL,
    attempts            INT NOT NULL DEFAULT 0,
    max_attempts        INT NOT NULL,
    next_attempt_after  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE orders;
