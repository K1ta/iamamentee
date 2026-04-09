package shardsmigrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"product-management/internal/app/models"
	"product-management/internal/infra/storage"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
	"golang.org/x/sync/errgroup"
)

const (
	fieldsToInsertInProduct = 4 // вместе с id
	logPrefixForRestartData = "RESTART_DATA"
)

type Migrator struct {
	newShards           storage.Shards[*sql.DB]
	prevShards          storage.Shards[*sql.DB]
	prevShardsStartFrom map[storage.ShardName]int64
	excludedPrevShards  []storage.ShardName
	batchLimit          int64
	isMigrating         bool
}

func New(
	newShards, prevShards storage.Shards[*sql.DB],
	prevShardsStartFrom map[storage.ShardName]int64,
	excludedPrevShards []storage.ShardName,
	batchLimit int64,
	isMigrating bool,
) (*Migrator, error) {
	if len(newShards) == 0 {
		return nil, errors.New("empty new shards")
	}
	if len(prevShards) == 0 {
		return nil, errors.New("empty prev shards")
	}
	for _, name := range excludedPrevShards {
		if _, ok := prevShards[name]; !ok {
			return nil, fmt.Errorf("unknown prev shard from excludedPrevShards: %s", name)
		}
	}
	for name := range prevShardsStartFrom {
		if _, ok := prevShards[name]; !ok {
			return nil, fmt.Errorf("unknown prev shard from prevShardsStartFrom: %s", name)
		}
	}
	return &Migrator{
		newShards:           newShards,
		prevShards:          prevShards,
		prevShardsStartFrom: prevShardsStartFrom,
		excludedPrevShards:  excludedPrevShards,
		batchLimit:          batchLimit,
		isMigrating:         isMigrating,
	}, nil
}

func (m *Migrator) Run(ctx context.Context) error {
	wg := new(sync.WaitGroup)
	for prevShardName, prevDB := range m.prevShards {
		if slices.Contains(m.excludedPrevShards, prevShardName) {
			log.Printf("skipping migration for prev shard %s", prevShardName)
			continue
		}
		productsMigrator := shardMigrator{
			prevDB:        prevDB,
			prevShardName: prevShardName,
			newShards:     m.newShards,
			batchLimit:    m.batchLimit,
			startFrom:     m.prevShardsStartFrom[prevShardName],
			isMigrating:   m.isMigrating,
		}
		wg.Go(func() {
			log.Printf("running migrations for ")
			productsMigrator.run(ctx)
		})
	}

	wg.Wait()
	log.Println("migration finished")
	return nil
}

type shardMigrator struct {
	prevDB        *sql.DB
	prevShardName storage.ShardName
	newShards     storage.Shards[*sql.DB]
	batchLimit    int64
	startFrom     int64
	isMigrating   bool
}

func (m *shardMigrator) run(ctx context.Context) {
	id := m.startFrom

	for {
		start := time.Now()
		log.Printf("starting loop for %s shard, id=%d", m.prevShardName, id)
		lastProductID, err := m.loop(ctx, id)
		if err != nil {
			restartData := fmt.Sprintf("prevShardName: %s, lastID: %d", m.prevShardName, id)
			log.Printf("%s runner failed. Data: %s. Reason: %v", logPrefixForRestartData, restartData, err)
			return
		}
		if lastProductID == -1 {
			log.Printf("worker for %s finished", m.prevShardName)
			return
		}
		id = lastProductID

		log.Printf("loop for %s shard finished, time elapsed: %dms", m.prevShardName, time.Since(start).Milliseconds())
	}
}

func (m *shardMigrator) loop(ctx context.Context, lastSelectedID int64) (int64, error) {
	products, err := m.selectProductsFromPrevShard(ctx, lastSelectedID)
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

func (m *shardMigrator) selectProductsFromPrevShard(ctx context.Context, lastSelectedID int64) ([]models.Product, error) {
	query := "SELECT id, user_id, name, price FROM products WHERE id > $1 ORDER BY id LIMIT $2"
	rows, err := m.prevDB.QueryContext(ctx, query, lastSelectedID, m.batchLimit)
	if err != nil {
		return nil, fmt.Errorf("query ctx: %w", err)
	}
	defer rows.Close()

	products := make([]models.Product, 0, m.batchLimit)
	for rows.Next() {
		var product models.Product
		if err = rows.Scan(&product.ID, &product.UserID, &product.Name, &product.Price); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		products = append(products, product)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return products, nil
}

func (m *shardMigrator) deleteProductsFromPrevShard(ctx context.Context, products []models.Product) error {
	ids := make([]int64, 0)
	for _, product := range products {
		newShardName, _ := m.newShards.Get(strconv.FormatInt(product.UserID, 10))
		if newShardName != m.prevShardName { // удаляем продукты, которые уехали в новые шарды
			ids = append(ids, product.ID)
		}
	}
	query := "DELETE FROM products WHERE id = ANY($1)"
	_, err := m.prevDB.ExecContext(ctx, query, pq.Array(ids))
	return err
}

func (m *shardMigrator) insertProductsToNewShards(ctx context.Context, products []models.Product) error {
	// split products by new shards
	productsByNewShards := make(map[storage.ShardName][]models.Product)
	for _, product := range products {
		newShardName, _ := m.newShards.Get(strconv.FormatInt(product.UserID, 10))
		if newShardName != m.prevShardName {
			productsByNewShards[newShardName] = append(productsByNewShards[newShardName], product)
		}
	}

	eg, egCtx := errgroup.WithContext(ctx)
	// insert products to new shards
	for newShardName, products := range productsByNewShards {
		eg.Go(func() error {
			err := insertProductsToShard(egCtx, products, m.newShards[newShardName])
			if err != nil {
				return fmt.Errorf("failed inserting to new shard %s: %w", newShardName, err)
			}
			return nil
		})
	}
	return eg.Wait()
}

func insertProductsToShard(ctx context.Context, products []models.Product, db *sql.DB) error {
	args := make([]any, 0, len(products)*fieldsToInsertInProduct)
	values := make([]string, 0, len(products))
	for _, product := range products {
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d)", len(args)+1, len(args)+2, len(args)+3, len(args)+4))
		args = append(args, product.ID, product.UserID, product.Name, product.Price)
	}
	// запрос не обновляет уже перенесенные данные. Они могут быть новее в новом шарде
	query := fmt.Sprintf("INSERT INTO products (id, user_id, name, price) VALUES %s "+
		"ON CONFLICT (id) DO NOTHING", strings.Join(values, ","))
	_, err := db.ExecContext(ctx, query, args...)
	return err
}
