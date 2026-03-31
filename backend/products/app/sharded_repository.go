package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"golang.org/x/sync/errgroup"
)

type ShardedSearchRepository struct {
	shards []ShardName
	repos  []*searchRepository
}

func NewShardedSearchRepository(shards map[ShardName]DBConnectionName, dbConnections map[DBConnectionName]*sql.DB) (*ShardedSearchRepository, error) {
	if len(shards) == 0 {
		return nil, errors.New("no shards")
	}
	if len(dbConnections) == 0 {
		return nil, errors.New("no db connections")
	}
	shardNames := make([]ShardName, 0, len(shards))
	repos := make([]*searchRepository, 0, len(shards))
	for shardName, connectionName := range shards {
		if conn, ok := dbConnections[connectionName]; ok {
			shardNames = append(shardNames, shardName)
			repos = append(repos, NewSearchRepository(conn))
		} else {
			return nil, fmt.Errorf("missing connection %s for shard %s", connectionName, shardName)
		}
	}
	return &ShardedSearchRepository{shards: shardNames, repos: repos}, nil
}

func (r *ShardedSearchRepository) ListByIDs(ctx context.Context, ids []int64) ([]Product, error) {
	shardForUDS := make(map[int][]int64, len(r.shards))
	for _, id := range ids {
		shardID := GetShardID(r.shards, strconv.FormatInt(id, 10))
		shardForUDS[shardID] = append(shardForUDS[shardID], id)
	}

	productsCh := make(chan []Product, len(shardForUDS))
	defer close(productsCh)
	res := make([]Product, 0)
	go func() {
		for products := range productsCh {
			res = append(res, products...)
		}
	}()

	eg, egCtx := errgroup.WithContext(ctx)
	for shardID, idsInShard := range shardForUDS {
		eg.Go(func() error {
			products, err := r.repos[shardID].ListByIDs(egCtx, idsInShard)
			if err != nil {
				return fmt.Errorf("failed for shard %s: %w", r.shards[shardID], err)
			}
			productsCh <- products
			return nil
		})
	}

	err := eg.Wait()
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (r *ShardedSearchRepository) Create(ctx context.Context, product *Product) error {
	shardID := GetShardID(r.shards, strconv.FormatInt(product.ID, 10))
	return r.repos[shardID].Create(ctx, product)
}

type MigratingShardedSearchRepository struct {
	repo           *ShardedSearchRepository
	prevShardsRepo *ShardedSearchRepository
}

func NewMigratingShardedSearchRepository(repo, prevShardsRepo *ShardedSearchRepository) *MigratingShardedSearchRepository {
	return &MigratingShardedSearchRepository{repo: repo, prevShardsRepo: prevShardsRepo}
}

func (r *MigratingShardedSearchRepository) ListByIDs(ctx context.Context, ids []int64) ([]Product, error) {
	products, err := r.repo.ListByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list from current shards: %w", err)
	}

	if len(products) == len(ids) {
		// Рассчитываем, что репа работает правильно и возвращает нужные id)
		return products, nil
	}

	absentIDs := make([]int64, 0, len(ids)/2)
	idsSet := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		idsSet[id] = struct{}{}
	}
	for _, product := range products {
		if _, ok := idsSet[product.ID]; !ok {
			absentIDs = append(absentIDs, product.ID)
		}
	}

	productsFromOldShards, err := r.prevShardsRepo.ListByIDs(ctx, absentIDs)
	if err != nil {
		return nil, fmt.Errorf("list from prev shards: %w", err)
	}
	return append(products, productsFromOldShards...), nil
}

func (r *MigratingShardedSearchRepository) Create(ctx context.Context, product *Product) error {
	return r.repo.Create(ctx, product)
}
