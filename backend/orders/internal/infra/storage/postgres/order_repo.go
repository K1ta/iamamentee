package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"orders/internal/domain"

	"github.com/lib/pq"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create вставляет заказ и его items. Заполняет order.ID сгенерированным значением.
func (r *OrderRepository) Create(ctx context.Context, order *domain.Order, maxAttempts int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	const orderQuery = `
		INSERT INTO orders (user_id, status, max_attempts, next_attempt_after)
		VALUES ($1, $2, $3, now())
		RETURNING id`
	row := tx.QueryRowContext(ctx, orderQuery, order.UserID, order.Status, maxAttempts)
	if err := row.Scan(&order.ID); err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	const itemQuery = `INSERT INTO items (order_id, product_id, amount, price) VALUES ($1, $2, $3, $4)`
	for _, item := range order.Items {
		if _, err := tx.ExecContext(ctx, itemQuery, order.ID, item.ProductID, item.Amount, item.Price); err != nil {
			return fmt.Errorf("insert item: %w", err)
		}
	}
	return tx.Commit()
}

// GetByID читает заказ с items в read-only транзакции для консистентного снимка.
func (r *OrderRepository) GetByID(ctx context.Context, id int64) (*domain.Order, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	const orderQuery = `SELECT user_id, status FROM orders WHERE id = $1`
	row := tx.QueryRowContext(ctx, orderQuery, id)
	var (
		userID int64
		status domain.Status
	)
	if err := row.Scan(&userID, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("scan order: %w", err)
	}

	items, err := getItems(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	order, err := domain.RestoreOrder(id, userID, status, items)
	if err != nil {
		return nil, fmt.Errorf("restore order: %w", err)
	}
	return order, tx.Commit()
}

// UpdateStatus обновляет статус заказа и сбрасывает счётчик попыток.
func (r *OrderRepository) UpdateStatus(ctx context.Context, order *domain.Order, prevStatus domain.Status, maxAttempts int) error {
	return updateStatus(ctx, r.db, order, prevStatus, maxAttempts)
}

// UpdateStatusAndSetPrices обновляет статус и фиксирует цены в items в одной транзакции.
func (r *OrderRepository) UpdateStatusAndSetPrices(ctx context.Context, order *domain.Order, prevStatus domain.Status, maxAttempts int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := updateStatus(ctx, tx, order, prevStatus, maxAttempts); err != nil {
		return err
	}
	if err := setPrices(ctx, tx, order); err != nil {
		return err
	}
	return tx.Commit()
}

// GetOneForProcessing атомарно выбирает один заказ в переданном статусе,
// готовый к обработке, и инкрементирует attempts.
// intervalSec задаёт, на сколько секунд сдвинуть next_attempt_after.
// max_attempts = -1 означает неограниченное количество попыток.
func (r *OrderRepository) GetOneForProcessing(ctx context.Context, status domain.Status, intervalSec int) (*domain.Order, error) {
	const query = `
		WITH locked AS (
			SELECT id FROM orders
			WHERE status = $1
			  AND (attempts < max_attempts OR max_attempts = -1)
			  AND next_attempt_after <= now()
			ORDER BY next_attempt_after
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE orders
		SET attempts = attempts + 1,
		    next_attempt_after = now() + ($2 * interval '1 second')
		WHERE id IN (SELECT id FROM locked)
		RETURNING id, user_id`
	row := r.db.QueryRowContext(ctx, query, status, intervalSec)
	var (
		id     int64
		userID int64
	)
	if err := row.Scan(&id, &userID); err != nil {
		return nil, fmt.Errorf("scan order: %w", err)
	}

	items, err := getItems(ctx, r.db, id)
	if err != nil {
		return nil, err
	}
	return domain.RestoreOrder(id, userID, status, items)
}

// GetOneExceededAttempts выбирает один заказ в переданном статусе,
// у которого исчерпаны попытки обработки (attempts >= max_attempts).
func (r *OrderRepository) GetOneExceededAttempts(ctx context.Context, status domain.Status) (*domain.Order, error) {
	const query = `
		SELECT id, user_id
		FROM orders
		WHERE status = $1
		  AND attempts >= max_attempts
		  AND max_attempts != -1
		LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, status)
	var (
		id     int64
		userID int64
	)
	if err := row.Scan(&id, &userID); err != nil {
		return nil, fmt.Errorf("scan order: %w", err)
	}

	items, err := getItems(ctx, r.db, id)
	if err != nil {
		return nil, err
	}
	return domain.RestoreOrder(id, userID, status, items)
}

func updateStatus(ctx context.Context, db DBTX, order *domain.Order, prevStatus domain.Status, maxAttempts int) error {
	const query = `
		UPDATE orders
		SET status = $1, attempts = 0, max_attempts = $2, next_attempt_after = now()
		WHERE id = $3 AND status = $4`
	res, err := db.ExecContext(ctx, query, order.Status, maxAttempts, order.ID, prevStatus)
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if n == 0 {
		return domain.ErrOrderConflict
	}
	return nil
}

func setPrices(ctx context.Context, db DBTX, order *domain.Order) error {
	productIDs := make([]int64, len(order.Items))
	prices := make([]int64, len(order.Items))
	for i, item := range order.Items {
		productIDs[i] = item.ProductID
		prices[i] = item.Price
	}

	const query = `
		UPDATE items SET price = v.price
		FROM (
			SELECT unnest($1::bigint[]) AS product_id,
			       unnest($2::bigint[]) AS price
		) AS v
		WHERE items.order_id = $3 AND items.product_id = v.product_id`
	_, err := db.ExecContext(ctx, query, pq.Array(productIDs), pq.Array(prices), order.ID)
	return err
}

func getItems(ctx context.Context, db DBTX, orderID int64) ([]domain.Item, error) {
	const query = `SELECT product_id, amount, price FROM items WHERE order_id = $1`
	rows, err := db.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Item, 0)
	for rows.Next() {
		var item domain.Item
		if err := rows.Scan(&item.ProductID, &item.Amount, &item.Price); err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return items, nil
}
