-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS items_order_id_idx ON items (order_id);
-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS products_read_idx;