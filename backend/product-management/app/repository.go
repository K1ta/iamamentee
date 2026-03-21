package app

import (
	"context"
	"database/sql"
	"fmt"
)

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// Create принимает product без ID, записывает данные в базу и заполняет ID
func (r *ProductRepository) Create(ctx context.Context, product *Product) error {
	query := "INSERT INTO products (id, user_id, name, price) VALUES (DEFAULT, $1, $2, $3) RETURNING id"
	row := r.db.QueryRowContext(ctx, query, product.UserID, product.Name, product.Price)
	var id int64
	if err := row.Scan(&id); err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	product.ID = id
	return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id int64) (*Product, error) {
	query := "SELECT user_id, name, price FROM products WHERE id=$1"
	row := r.db.QueryRowContext(ctx, query, id)
	var product Product
	if err := row.Scan(&product.UserID, &product.Name, &product.Price); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	product.ID = id
	return &product, nil
}

func (r *ProductRepository) List(ctx context.Context, userID int64) ([]Product, error) {
	query := "SELECT id, name, price FROM products WHERE user_id=$1"
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	res := make([]Product, 0)
	for rows.Next() {
		var product Product
		if err = rows.Scan(&product.ID, &product.Name, &product.Price); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		product.UserID = userID
		res = append(res, product)
	}
	return res, nil
}
