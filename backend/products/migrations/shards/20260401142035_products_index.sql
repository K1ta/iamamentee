-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS products_read_idx ON products (id) INCLUDE (user_id, name, price);
-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS products_read_idx;