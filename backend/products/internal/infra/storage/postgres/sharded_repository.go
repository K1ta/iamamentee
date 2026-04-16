package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"products/internal/domain"
	"products/internal/pkg/sharding"
	"strconv"

	"golang.org/x/sync/errgroup"
)

type ShardedSearchRepository struct {
	shards map[sharding.ShardName]domain.SearchRepository
}

func NewShardedSearchRepository(shards map[sharding.ShardName]domain.SearchRepository) (*ShardedSearchRepository, error) {
	if len(shards) == 0 {
		return nil, errors.New("no shards")
	}
	return &ShardedSearchRepository{shards: shards}, nil
}

func (r *ShardedSearchRepository) ListByIDs(ctx context.Context, ids []int64) ([]domain.Product, error) {
	shardForIDs := make(map[sharding.ShardName][]int64, len(r.shards))
	for _, id := range ids {
		shardName, _ := sharding.GetShard(r.shards, strconv.FormatInt(id, 10))
		shardForIDs[shardName] = append(shardForIDs[shardName], id)
	}

	// TODO возможно, лучше сделать eventual consistency и отдавать часть результата с успешных шардов
	productsCh := make(chan []domain.Product, len(shardForIDs)) // буфер равен числу писателей -> никто не блокируется
	eg, egCtx := errgroup.WithContext(ctx)
	for shardName, idsInShard := range shardForIDs {
		eg.Go(func() error {
			log.Println("list by ids from shard", shardName, ":", idsInShard)
			products, err := r.shards[shardName].ListByIDs(egCtx, idsInShard)
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

	res := make([]domain.Product, 0)
	for products := range productsCh { // можем не проверять контекст, так как канал закроется после eg.Wait()
		res = append(res, products...)
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return res, nil
}

func (r *ShardedSearchRepository) Create(ctx context.Context, product *domain.Product) error {
	_, repo := sharding.GetShard(r.shards, strconv.FormatInt(product.ID, 10))
	return repo.Create(ctx, product)
}

type MigratingShardedSearchRepository struct {
	repo           *ShardedSearchRepository
	prevShardsRepo *ShardedSearchRepository
}

func NewMigratingShardedSearchRepository(repo, prevShardsRepo *ShardedSearchRepository) *MigratingShardedSearchRepository {
	return &MigratingShardedSearchRepository{repo: repo, prevShardsRepo: prevShardsRepo}
}

func (r *MigratingShardedSearchRepository) ListByIDs(ctx context.Context, ids []int64) ([]domain.Product, error) {
	products, err := r.repo.ListByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list from current shards: %w", err)
	}

	if len(products) == len(ids) {
		// Рассчитываем, что репа работает правильно и возвращает нужные id)
		return products, nil
	}

	absentIDs := make([]int64, 0, len(ids)/2)
	foundProductsIDs := make(map[int64]struct{}, len(ids))
	for _, product := range products {
		foundProductsIDs[product.ID] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := foundProductsIDs[id]; !ok {
			absentIDs = append(absentIDs, id)
		}
	}
	log.Println("not all ids found, search prev shards for:", absentIDs)

	productsFromOldShards, err := r.prevShardsRepo.ListByIDs(ctx, absentIDs)
	if err != nil {
		return nil, fmt.Errorf("list from prev shards: %w", err)
	}
	return append(products, productsFromOldShards...), nil
}

func (r *MigratingShardedSearchRepository) Create(ctx context.Context, product *domain.Product) error {
	return r.repo.Create(ctx, product)
}
