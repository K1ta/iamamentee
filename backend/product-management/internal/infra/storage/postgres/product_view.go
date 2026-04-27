package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"maps"
	"product-management/internal/domain"
	"product-management/internal/pkg/sharding"
	"strconv"
	"sync"

	"golang.org/x/sync/errgroup"
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

func (v *ProductView) GetPrices(ctx context.Context, ids []int64) (map[int64]int64, error) {
	prices, err := v.getPricesFromPool(ctx, v.shardsPool, ids)
	if err != nil {
		return nil, err
	}
	if len(prices) == len(ids) || v.prevShardsPool == nil {
		return prices, nil
	}

	missingIDs := make([]int64, 0, len(ids)-len(prices))
	for _, id := range ids {
		if _, ok := prices[id]; !ok {
			missingIDs = append(missingIDs, id)
		}
	}
	log.Printf("trying to get prices for %d products from prev shards", len(missingIDs))
	prevPrices, err := v.getPricesFromPool(ctx, v.prevShardsPool, missingIDs)
	if err != nil {
		return nil, err
	}
	maps.Copy(prices, prevPrices)
	return prices, nil
}

func (v *ProductView) getPricesFromPool(ctx context.Context, pool *sharding.Pool[*sql.DB], ids []int64) (map[int64]int64, error) {
	var mu sync.Mutex
	merged := make(map[int64]int64, len(ids))

	g, ctx := errgroup.WithContext(ctx)
	for _, db := range pool.All() {
		g.Go(func() error {
			prices, err := NewProductRepository(db).GetPricesByIDs(ctx, ids)
			if err != nil {
				return err
			}
			mu.Lock()
			maps.Copy(merged, prices)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return merged, nil
}

func (v *ProductView) List(ctx context.Context, userID int64) ([]domain.Product, error) {
	repo := NewProductRepository(v.shardsPool.Get(strconv.FormatInt(userID, 10)))
	products, err := repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get products for user %d from current shards: %w", userID, err)
	}
	if len(products) > 0 || v.prevShardsPool == nil {
		return products, nil
	}
	log.Printf("trying to get products for user %d from prev shards", userID)
	repo = NewProductRepository(v.prevShardsPool.Get(strconv.FormatInt(userID, 10)))
	products, err = repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get products for user %d from prev shards: %w", userID, err)
	}
	return products, nil
}
