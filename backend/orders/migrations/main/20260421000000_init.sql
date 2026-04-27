-- +goose Up
CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    status TEXT NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL,
    next_attempt_after TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id),
    product_id BIGINT NOT NULL,
    amount INT NOT NULL,
    price BIGINT NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE items;
DROP TABLE orders;
