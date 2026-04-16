package shardsmigrator

import (
	"context"
	"fmt"
	"log"
	"product-management/internal/app/domain"
	"product-management/internal/pkg/sharding"
	"slices"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type Repository interface {
	ListByIDLimited(ctx context.Context, fromID int64, limit int) ([]domain.Product, error)
	CreateBatch(ctx context.Context, products []domain.Product) error
	DeleteBatch(ctx context.Context, ids []int64) error
}

type Migrator struct {
	prevShards          map[sharding.ShardName]Repository
	newShardsPool       *sharding.Pool[Repository]
	prevShardsStartFrom map[sharding.ShardName]int64
	excludedPrevShards  []sharding.ShardName
	batchLimit          int
	isMigrating         bool
}

func New(
	prevShards map[sharding.ShardName]Repository,
	newShardsPool *sharding.Pool[Repository],
	prevShardsStartFrom map[sharding.ShardName]int64,
	excludedPrevShards []sharding.ShardName,
	batchLimit int,
	isMigrating bool,
) *Migrator {
	return &Migrator{
		prevShards:          prevShards,
		newShardsPool:       newShardsPool,
		prevShardsStartFrom: prevShardsStartFrom,
		excludedPrevShards:  excludedPrevShards,
		batchLimit:          batchLimit,
		isMigrating:         isMigrating,
	}
}

func (m *Migrator) Run(ctx context.Context) {
	migrators := make([]*shardMigrator, 0)
	wg := new(sync.WaitGroup)
	for prevShardName, prevDB := range m.prevShards {
		if slices.Contains(m.excludedPrevShards, prevShardName) {
			log.Printf("skipping migration for prev shard %s", prevShardName)
			continue
		}
		migrator := &shardMigrator{
			prevRepo:      prevDB,
			prevShardName: prevShardName,
			newShards:     m.newShardsPool,
			batchLimit:    m.batchLimit,
			startFrom:     m.prevShardsStartFrom[prevShardName],
			isMigrating:   m.isMigrating,
		}
		migrators = append(migrators, migrator)
		wg.Go(func() {
			log.Printf("running migrations for %s", prevShardName)
			migrator.run(ctx)
		})
	}
	wg.Wait()

	for _, migrator := range migrators {
		log.Println(migrator.state())
	}

	log.Println("migration finished")
}

type shardMigrator struct {
	prevRepo      Repository
	prevShardName sharding.ShardName
	newShards     *sharding.Pool[Repository]
	batchLimit    int
	startFrom     int64
	isMigrating   bool

	err        error
	lastNextID int64 // пустой если err == nil
}

func (m *shardMigrator) run(ctx context.Context) {
	nextID := m.startFrom

	for {
		start := time.Now()
		log.Printf("starting loop for %s shard, id=%d", m.prevShardName, nextID)
		lastProductID, err := m.loop(ctx, nextID)
		if err != nil {
			log.Printf("runner for %s shard failed: %v", m.prevShardName, err)
			m.lastNextID = nextID
			m.err = err
			return
		}
		if lastProductID == -1 {
			log.Printf("worker for %s finished", m.prevShardName)
			return
		}
		nextID = lastProductID + 1

		log.Printf("loop for %s shard finished, time elapsed: %dms", m.prevShardName, time.Since(start).Milliseconds())
	}
}

// state возвращает готовый лог по статусу миграции. Вызывать только после завершения [shardMigrator.run].
func (m *shardMigrator) state() string {
	if m.err == nil {
		return fmt.Sprintf("runner [%s] finished successfully.", m.prevShardName)
	}
	return fmt.Sprintf("runner [%s] FAILED, start next migration from id %d. Reason: %v", m.prevShardName, m.lastNextID, m.err)
}

func (m *shardMigrator) loop(ctx context.Context, fromID int64) (int64, error) {
	products, err := m.prevRepo.ListByIDLimited(ctx, fromID, m.batchLimit)
	if err != nil {
		return 0, err
	}

	// нормальный выход - дошли до конца таблицы
	if len(products) == 0 {
		return -1, nil
	}

	lastProductID := products[len(products)-1].ID
	if m.isMigrating {
		return lastProductID, m.insertProductsToNewShards(ctx, products)
	}
	return lastProductID, m.deleteProductsFromPrevShard(ctx, products)
}

func (m *shardMigrator) insertProductsToNewShards(ctx context.Context, products []domain.Product) error {
	// split products by new shards
	productsByNewShards := make(map[sharding.ShardName][]domain.Product)
	for _, product := range products {
		newShardName := m.newShards.GetName(strconv.FormatInt(product.UserID, 10))
		if newShardName != m.prevShardName {
			productsByNewShards[newShardName] = append(productsByNewShards[newShardName], product)
		}
	}

	eg, egCtx := errgroup.WithContext(ctx)
	// insert products to new shards
	for newShardName, products := range productsByNewShards {
		eg.Go(func() error {
			err := m.newShards.GetByName(newShardName).CreateBatch(egCtx, products)
			if err != nil {
				return fmt.Errorf("failed inserting to new shard %s: %w", newShardName, err)
			}
			return nil
		})
	}
	return eg.Wait()
}

func (m *shardMigrator) deleteProductsFromPrevShard(ctx context.Context, products []domain.Product) error {
	ids := make([]int64, 0)
	for _, product := range products {
		newShardName := m.newShards.GetName(strconv.FormatInt(product.UserID, 10))
		if newShardName != m.prevShardName { // удаляем продукты, которые уехали в новые шарды
			ids = append(ids, product.ID)
		}
	}
	return m.prevRepo.DeleteBatch(ctx, ids)
}
