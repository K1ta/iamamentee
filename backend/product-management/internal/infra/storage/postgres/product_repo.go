package postgres

import (
	"context"
	"fmt"
	"product-management/internal/app/domain"
	"strings"

	"github.com/lib/pq"
)

const fieldsToInsertInProduct = 4 // вместе с id

type ProductRepository struct {
	db DBTX
}

func NewProductRepository(db DBTX) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(ctx context.Context, product *domain.Product) error {
	query := "INSERT INTO products (id, user_id, name, price) VALUES ($1, $2, $3, $4)"
	_, err := r.db.ExecContext(ctx, query, product.ID, product.UserID, product.Name, product.Price)
	return err
}

func (r *ProductRepository) CreateBatch(ctx context.Context, products []domain.Product) error {
	args := make([]any, 0, len(products)*fieldsToInsertInProduct)
	values := make([]string, 0, len(products))
	for _, product := range products {
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d)", len(args)+1, len(args)+2, len(args)+3, len(args)+4))
		args = append(args, product.ID, product.UserID, product.Name, product.Price)
	}
	query := fmt.Sprintf("INSERT INTO products (id, user_id, name, price) VALUES %s "+
		"ON CONFLICT (id) DO NOTHING", strings.Join(values, ","))
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *ProductRepository) GetByID(ctx context.Context, id, userID int64) (*domain.Product, error) {
	query := "SELECT user_id, name, price FROM products WHERE id=$1 AND user_id=$2"
	row := r.db.QueryRowContext(ctx, query, id, userID)
	var product domain.Product
	if err := row.Scan(&product.UserID, &product.Name, &product.Price); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	product.ID = id
	return &product, nil
}

func (r *ProductRepository) List(ctx context.Context, userID int64) ([]domain.Product, error) {
	query := "SELECT id, name, price FROM products WHERE user_id=$1"
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	res := make([]domain.Product, 0)
	for rows.Next() {
		var product domain.Product
		if err = rows.Scan(&product.ID, &product.Name, &product.Price); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		product.UserID = userID
		res = append(res, product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return res, nil
}

func (r *ProductRepository) ListByIDLimited(ctx context.Context, fromID int64, limit int) ([]domain.Product, error) {
	query := "SELECT id, user_id, name, price FROM products WHERE id >= $1 ORDER BY id LIMIT $2"
	rows, err := r.db.QueryContext(ctx, query, fromID, limit)
	if err != nil {
		return nil, fmt.Errorf("query ctx: %w", err)
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
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return products, nil
}

func (r *ProductRepository) DeleteBatch(ctx context.Context, ids []int64) error {
	query := "DELETE FROM products WHERE id = ANY($1)"
	_, err := r.db.ExecContext(ctx, query, pq.Array(ids))
	return err
}
