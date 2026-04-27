-- +goose Up
CREATE TABLE products (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    price BIGINT NOT NULL
);
-- +goose Down
DROP TABLE products;