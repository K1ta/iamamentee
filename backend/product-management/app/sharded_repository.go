package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
)

type ShardedProductRepository struct {
	shards map[string]ProductRepository
}

func NewShardedProductRepository(shards map[string]ProductRepository) (*ShardedProductRepository, error) {
	if len(shards) == 0 {
		return nil, errors.New("no shards")
	}
	return &ShardedProductRepository{shards: shards}, nil
}

func (r *ShardedProductRepository) Create(ctx context.Context, product *Product) error {
	_, repo := GetShard(r.shards, strconv.FormatInt(product.UserID, 10))
	return repo.Create(ctx, product)
}

func (r *ShardedProductRepository) GetByID(ctx context.Context, id, userID int64) (*Product, error) {
	_, repo := GetShard(r.shards, strconv.FormatInt(userID, 10))
	return repo.GetByID(ctx, id, userID)
}

func (r *ShardedProductRepository) List(ctx context.Context, userID int64) ([]Product, error) {
	_, repo := GetShard(r.shards, strconv.FormatInt(userID, 10))
	return repo.List(ctx, userID)
}

type MigratingShardedProductRepository struct {
	repo           *ShardedProductRepository
	prevShardsRepo *ShardedProductRepository
}

func NewMigratingShardedProductRepository(repo, prevShardsRepo *ShardedProductRepository) *MigratingShardedProductRepository {
	return &MigratingShardedProductRepository{repo: repo, prevShardsRepo: prevShardsRepo}
}

func (r *MigratingShardedProductRepository) Create(ctx context.Context, product *Product) error {
	return r.repo.Create(ctx, product)
}

func (r *MigratingShardedProductRepository) GetByID(ctx context.Context, id, userID int64) (*Product, error) {
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

func (r *MigratingShardedProductRepository) List(ctx context.Context, userID int64) ([]Product, error) {
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
