package app

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

type SearchRepository struct {
	db *sql.DB
}

func NewSearchRepository(db *sql.DB) *SearchRepository {
	return &SearchRepository{db: db}
}

func (r *SearchRepository) ListByIDs(ctx context.Context, ids []int64) ([]Product, error) {
	query := "SELECT id, user_id, name, price FROM products WHERE id=ANY($1)"
	rows, err := r.db.QueryContext(ctx, query, pq.Array(ids)) // todo pagination
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	res := make([]Product, 0)
	for rows.Next() {
		var product Product
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

func (r *SearchRepository) Create(ctx context.Context, product *Product) error {
	// TODO on conflict update?
	query := "INSERT INTO products (id, user_id, name, price) VALUES ($1, $2, $3, $4)"
	_, err := r.db.ExecContext(ctx, query, product.ID, product.UserID, product.Name, product.Price)
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	return nil
}
