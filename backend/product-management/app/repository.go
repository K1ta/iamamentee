package app

import (
	"context"
	"database/sql"
	"fmt"
)

// ProductRepository интерфейс для базы, нужен, так как у нас появилась обертка с шардом. Пока объявляем здесь, при разделении
// на слои надо перенести по месту использования
type ProductRepository interface {
	Create(ctx context.Context, product *Product) error
	GetByID(ctx context.Context, id, userID int64) (*Product, error)
	List(ctx context.Context, userID int64) ([]Product, error)
}

type productRepository struct {
	db        *sql.DB
	snowflake *Snowflake
}

func NewProductRepository(db *sql.DB, snowflake *Snowflake) *productRepository {
	return &productRepository{db: db, snowflake: snowflake}
}

// Create принимает product без ID, генерирует его из snowflake записывает данные в базу
func (r *productRepository) Create(ctx context.Context, product *Product) error {
	product.ID = r.snowflake.NextID()
	query := "INSERT INTO products (id, user_id, name, price) VALUES ($1, $2, $3, $4)"
	_, err := r.db.ExecContext(ctx, query, product.ID, product.UserID, product.Name, product.Price)
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	return nil
}

func (r *productRepository) GetByID(ctx context.Context, id, userID int64) (*Product, error) {
	query := "SELECT user_id, name, price FROM products WHERE id=$1 AND user_id=$2"
	row := r.db.QueryRowContext(ctx, query, id, userID)
	var product Product
	if err := row.Scan(&product.UserID, &product.Name, &product.Price); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	product.ID = id
	return &product, nil
}

func (r *productRepository) List(ctx context.Context, userID int64) ([]Product, error) {
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
	if rows.Err() != nil {
		return nil, err
	}
	return res, nil
}
