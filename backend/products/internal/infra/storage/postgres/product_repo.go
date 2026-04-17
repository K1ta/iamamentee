package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"products/internal/domain"
	"strings"

	"github.com/lib/pq"
)

const fieldsToInsertInProduct = 4 // вместе с id

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) ListByIDs(ctx context.Context, ids []int64) ([]domain.Product, error) {
	query := "SELECT id, user_id, name, price FROM products WHERE id=ANY($1)"
	rows, err := r.db.QueryContext(ctx, query, pq.Array(ids)) // todo pagination
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	res := make([]domain.Product, 0)
	for rows.Next() {
		var product domain.Product
		if err = rows.Scan(&product.ID, &product.UserID, &product.Name, &product.Price); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		res = append(res, product)
	}
	if rows.Err() != nil {
		return nil, err
	}
	return res, nil
}

func (r *ProductRepository) Create(ctx context.Context, product *domain.Product) error {
	query := "INSERT INTO products (id, user_id, name, price) VALUES ($1, $2, $3, $4) ON CONFLICT (id) " +
		"DO UPDATE SET name=EXCLUDED.name, price=EXCLUDED.price"
	_, err := r.db.ExecContext(ctx, query, product.ID, product.UserID, product.Name, product.Price)
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	return nil
}

func (r *ProductRepository) ListByIDLimited(ctx context.Context, fromID int64, limit int) ([]domain.Product, error) {
	query := "SELECT id, user_id, name, price FROM products WHERE id >= $1 ORDER BY id LIMIT $2"
	rows, err := r.db.QueryContext(ctx, query, fromID, limit)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	products := make([]domain.Product, 0, limit)
	for rows.Next() {
		var product domain.Product
		if err = rows.Scan(&product.ID, &product.UserID, &product.Name, &product.Price); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		products = append(products, product)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return products, nil
}

func (r *ProductRepository) CreateBatch(ctx context.Context, products []domain.Product) error {
	args := make([]any, 0, len(products)*fieldsToInsertInProduct)
	values := make([]string, 0, len(products))
	for _, product := range products {
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d)", len(args)+1, len(args)+2, len(args)+3, len(args)+4))
		args = append(args, product.ID, product.UserID, product.Name, product.Price)
	}
	// не обновляем уже перенесённые данные — они могут быть новее в новом шарде
	query := fmt.Sprintf("INSERT INTO products (id, user_id, name, price) VALUES %s ON CONFLICT (id) DO NOTHING",
		strings.Join(values, ","))
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *ProductRepository) DeleteBatch(ctx context.Context, ids []int64) error {
	query := "DELETE FROM products WHERE id = ANY($1)"
	_, err := r.db.ExecContext(ctx, query, pq.Array(ids))
	return err
}
