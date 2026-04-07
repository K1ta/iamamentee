package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"product-management/internal/app/models"
	"product-management/internal/pkg/sharding"
	"strconv"
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

// ShardedProductRepository репозиторий для работы с шардами
type ShardedProductRepository struct {
	shards sharding.Shards[*ProductRepository]
}

func NewShardedProductRepository(shards sharding.Shards[*ProductRepository]) (*ShardedProductRepository, error) {
	if len(shards) == 0 {
		return nil, errors.New("no shards")
	}
	return &ShardedProductRepository{shards: shards}, nil
}

func (r *ShardedProductRepository) Create(ctx context.Context, product *models.Product) error {
	_, repo := r.shards.Get(strconv.FormatInt(product.UserID, 10))
	return repo.Create(ctx, product)
}

func (r *ShardedProductRepository) GetByID(ctx context.Context, id, userID int64) (*models.Product, error) {
	_, repo := r.shards.Get(strconv.FormatInt(userID, 10))
	return repo.GetByID(ctx, id, userID)
}

func (r *ShardedProductRepository) List(ctx context.Context, userID int64) ([]models.Product, error) {
	_, repo := r.shards.Get(strconv.FormatInt(userID, 10))
	return repo.List(ctx, userID)
}

// MigratingProductRepository репозиторий для работы с шардами во время миграции. WRITE в новые шарды, READ из новых с fallback к старым
type MigratingProductRepository struct {
	repo           *ShardedProductRepository
	prevShardsRepo *ShardedProductRepository
}

func NewMigratingProductRepository(repo, prevShardsRepo *ShardedProductRepository) *MigratingProductRepository {
	return &MigratingProductRepository{repo: repo, prevShardsRepo: prevShardsRepo}
}

func (r *MigratingProductRepository) Create(ctx context.Context, product *models.Product) error {
	return r.repo.Create(ctx, product)
}

func (r *MigratingProductRepository) GetByID(ctx context.Context, id, userID int64) (*models.Product, error) {
	product, err := r.repo.GetByID(ctx, id, userID)
	if err == nil {
		return product, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get product %d from current shards: %w", id, err)
	}
	log.Printf("trying to get product %d from prev shards", id)
	product, err = r.prevShardsRepo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product %d from prev shards: %w", id, err)
	}
	return product, nil
}

func (r *MigratingProductRepository) List(ctx context.Context, userID int64) ([]models.Product, error) {
	products, err := r.repo.List(ctx, userID)
	if err == nil && len(products) > 0 {
		return products, nil
	}
	if err != nil {
		// логируем ошибку, но все равно пробуем сходить в старые шарды
		log.Printf("failed to get products for user %d from current shards: %v", userID, err)
	}
	log.Printf("trying to get producers for usr %d from prev shards", userID)
	products, err = r.prevShardsRepo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get products for user %d from prev shards: %w", userID, err)
	}
	return products, nil
}
