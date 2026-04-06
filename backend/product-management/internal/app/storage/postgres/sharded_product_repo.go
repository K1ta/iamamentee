package postgres

import (
	"context"
	"errors"
	"product-management/internal/app/models"
	"product-management/internal/pkg/sharding"
	"strconv"
)

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
