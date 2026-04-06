package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"product-management/internal/app/models"
)

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
