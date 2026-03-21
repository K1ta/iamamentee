package app

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type SearchRepository struct {
	db *sql.DB
}

func NewSearchRepository(db *sql.DB) *SearchRepository {
	return &SearchRepository{db: db}
}

func (r *SearchRepository) ListByFilter(ctx context.Context, filter *SearchRequest) ([]Product, error) {
	query := "SELECT id, user_id, name, price FROM products"
	args := make([]any, 0)
	clauses := make([]string, 0)
	if filter.Name != "" {
		clauses = append(clauses, fmt.Sprintf("name = $%d", len(args)+1))
		args = append(args, filter.Name)
	}
	if filter.PriceFrom > 0 {
		clauses = append(clauses, fmt.Sprintf("price >= $%d", len(args)+1))
		args = append(args, filter.PriceFrom)
	}
	if filter.PriceTo > 0 {
		clauses = append(clauses, fmt.Sprintf("price <= $%d", len(args)+1))
		args = append(args, filter.PriceTo)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	// todo pagination
	rows, err := r.db.QueryContext(ctx, query, args...)
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
