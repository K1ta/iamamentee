package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"product-management/internal/app/domain"
	"product-management/internal/pkg/sharding"
	"strconv"
)

type ProductView struct {
	shardsPool     *sharding.Pool[*sql.DB]
	prevShardsPool *sharding.Pool[*sql.DB] // может быть nil
}

func NewProductView(shardsPool *sharding.Pool[*sql.DB], prevShardsPool *sharding.Pool[*sql.DB]) *ProductView {
	return &ProductView{shardsPool: shardsPool, prevShardsPool: prevShardsPool}
}

func (v *ProductView) GetByID(ctx context.Context, id, userID int64) (*domain.Product, error) {
	repo := NewProductRepository(v.shardsPool.Get(strconv.FormatInt(userID, 10)))
	product, err := repo.GetByID(ctx, id, userID)
	if err == nil {
		return product, nil
	}
	if v.prevShardsPool == nil {
		// Если мы не в режиме миграции шардов, то просто выходим
		return nil, err
	}
	// Если доступен prevShardsPool, идем туда
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get product %d from current shards: %w", id, err)
	}
	log.Printf("trying to get product %d from prev shards", id)
	repo = NewProductRepository(v.prevShardsPool.Get(strconv.FormatInt(userID, 10)))
	product, err = repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product %d from prev shards: %w", id, err)
	}
	return product, nil
}

func (v *ProductView) List(ctx context.Context, userID int64) ([]domain.Product, error) {
	repo := NewProductRepository(v.shardsPool.Get(strconv.FormatInt(userID, 10)))
	products, err := repo.List(ctx, userID)
	if err == nil && len(products) > 0 {
		return products, nil
	}
	if v.prevShardsPool == nil {
		// Если мы не в режиме миграции шардов, то просто выходим. products может быть пустым
		return products, err
	}
	// Если доступен prevShardsPool, идем туда
	if err != nil {
		// логируем ошибку, но все равно пробуем сходить в старые шарды
		log.Printf("failed to get products for user %d from current shards: %v", userID, err)
	}
	log.Printf("trying to get producers for usr %d from prev shards", userID)
	repo = NewProductRepository(v.prevShardsPool.Get(strconv.FormatInt(userID, 10)))
	products, err = repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get products for user %d from prev shards: %w", userID, err)
	}
	return products, nil
}
