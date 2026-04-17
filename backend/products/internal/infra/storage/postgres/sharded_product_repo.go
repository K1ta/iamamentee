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
	shards     map[sharding.ShardName]*ProductRepository
	prevShards map[sharding.ShardName]*ProductRepository // nil если нет миграции шардов
}

func NewShardedProductRepository(
	shards map[sharding.ShardName]*ProductRepository,
	prevShards map[sharding.ShardName]*ProductRepository,
) *ShardedProductRepository {
	return &ShardedProductRepository{shards: shards, prevShards: prevShards}
}

func (r *ShardedProductRepository) Create(ctx context.Context, product *domain.Product) error {
	_, repo := sharding.GetShard(r.shards, strconv.FormatInt(product.ID, 10))
	return repo.Create(ctx, product)
}

func (r *ShardedProductRepository) ListByIDs(ctx context.Context, ids []int64) ([]domain.Product, error) {
	products, err := listByIDsFromShards(ctx, r.shards, ids)
	if err != nil {
		return nil, fmt.Errorf("list from current shards: %w", err)
	}

	if len(r.prevShards) == 0 || len(products) == len(ids) {
		return products, nil
	}

	absent := absentIDs(ids, products)
	log.Println("not all ids found, search prev shards for:", absent)

	prevProducts, err := listByIDsFromShards(ctx, r.prevShards, absent)
	if err != nil {
		return nil, fmt.Errorf("list from prev shards: %w", err)
	}
	return append(products, prevProducts...), nil
}

func listByIDsFromShards(
	ctx context.Context,
	shards map[sharding.ShardName]*ProductRepository,
	ids []int64,
) ([]domain.Product, error) {
	shardForIDs := make(map[sharding.ShardName][]int64, len(shards))
	for _, id := range ids {
		shardName, _ := sharding.GetShard(shards, strconv.FormatInt(id, 10))
		shardForIDs[shardName] = append(shardForIDs[shardName], id)
	}

	// буфер равен числу писателей → никто не блокируется
	productsCh := make(chan []domain.Product, len(shardForIDs))
	eg, egCtx := errgroup.WithContext(ctx)
	for shardName, idsInShard := range shardForIDs {
		eg.Go(func() error {
			log.Println("list by ids from shard", shardName, ":", idsInShard)
			products, err := shards[shardName].ListByIDs(egCtx, idsInShard)
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
