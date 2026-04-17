package postgres

import (
	"context"
	"fmt"
	"log"
	"products/internal/domain"
	"products/internal/pkg/sharding"
	"strconv"

	"golang.org/x/sync/errgroup"
)

// ShardedProductRepository реализует domain.ProductRepository.
// Знает про текущие шарды и опционально про предыдущие (при миграции шардов).
type ShardedProductRepository struct {
	shards     *sharding.Pool[*ProductRepository]
	prevShards *sharding.Pool[*ProductRepository] // nil если нет миграции шардов
}

func NewShardedProductRepository(
	shards *sharding.Pool[*ProductRepository],
	prevShards *sharding.Pool[*ProductRepository],
) *ShardedProductRepository {
	return &ShardedProductRepository{shards: shards, prevShards: prevShards}
}

func (r *ShardedProductRepository) Create(ctx context.Context, product *domain.Product) error {
	repo := r.shards.Get(strconv.FormatInt(product.ID, 10))
	return repo.Create(ctx, product)
}

func (r *ShardedProductRepository) ListByIDs(ctx context.Context, ids []int64) ([]domain.Product, error) {
	products, err := listByIDsFromPool(ctx, r.shards, ids)
	if err != nil {
		return nil, fmt.Errorf("list from current shards: %w", err)
	}

	if r.prevShards == nil || len(products) == len(ids) {
		return products, nil
	}

	absent := absentIDs(ids, products)
	log.Println("not all ids found, search prev shards for:", absent)

	prevProducts, err := listByIDsFromPool(ctx, r.prevShards, absent)
	if err != nil {
		return nil, fmt.Errorf("list from prev shards: %w", err)
	}
	return append(products, prevProducts...), nil
}

func listByIDsFromPool(
	ctx context.Context,
	pool *sharding.Pool[*ProductRepository],
	ids []int64,
) ([]domain.Product, error) {
	shardForIDs := make(map[sharding.ShardName][]int64)
	for _, id := range ids {
		shardName := pool.GetName(strconv.FormatInt(id, 10))
		shardForIDs[shardName] = append(shardForIDs[shardName], id)
	}

	// буфер равен числу писателей → никто не блокируется
	productsCh := make(chan []domain.Product, len(shardForIDs))
	eg, egCtx := errgroup.WithContext(ctx)
	for shardName, idsInShard := range shardForIDs {
		eg.Go(func() error {
			log.Println("list by ids from shard", shardName, ":", idsInShard)
			products, err := pool.GetByName(shardName).ListByIDs(egCtx, idsInShard)
			if err != nil {
				return fmt.Errorf("failed for shard %s: %w", shardName, err)
			}
			log.Println("data from", shardName, ":", products)
			productsCh <- products
			return nil
		})
	}

	go func() {
		eg.Wait()
		close(productsCh)
	}()

	res := make([]domain.Product, 0, len(ids))
	for products := range productsCh {
		res = append(res, products...)
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return res, nil
}

func absentIDs(ids []int64, found []domain.Product) []int64 {
	foundIDs := make(map[int64]struct{}, len(found))
	for _, p := range found {
		foundIDs[p.ID] = struct{}{}
	}
	absent := make([]int64, 0, len(ids)-len(found))
	for _, id := range ids {
		if _, ok := foundIDs[id]; !ok {
			absent = append(absent, id)
		}
	}
	return absent
}
