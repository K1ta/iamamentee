package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"product-management/internal/app/models"
)

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(ctx context.Context, product *models.Product) error {
	query := "INSERT INTO products (id, user_id, name, price) VALUES ($1, $2, $3, $4)"
	_, err := r.db.ExecContext(ctx, query, product.ID, product.UserID, product.Name, product.Price)
	return err
}

func (r *ProductRepository) GetByID(ctx context.Context, id, userID int64) (*models.Product, error) {
	query := "SELECT user_id, name, price FROM products WHERE id=$1 AND user_id=$2"
	row := r.db.QueryRowContext(ctx, query, id, userID)
	var product models.Product
	if err := row.Scan(&product.UserID, &product.Name, &product.Price); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	product.ID = id
	return &product, nil
}

func (r *ProductRepository) List(ctx context.Context, userID int64) ([]models.Product, error) {
	query := "SELECT id, name, price FROM products WHERE user_id=$1"
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	res := make([]models.Product, 0)
	for rows.Next() {
		var product models.Product
		if err = rows.Scan(&product.ID, &product.Name, &product.Price); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		product.UserID = userID
		res = append(res, product)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return res, nil
}
