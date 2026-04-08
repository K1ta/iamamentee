package shardmigrator

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"product-management/internal/app/models"
	"product-management/internal/infra/storage"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/lib/pq"
	"golang.org/x/sync/errgroup"
)

const (
	fieldsToInsertInProduct = 4 // вместе с id
	logPrefixForRestartData = "RESTART_DATA"
)

type (
	dbConnectionName = string
	dsn              = string
)

type Config struct {
	DBConnections       map[dbConnectionName]dsn               `env:"MIGRATOR_DB_CONNECTIONS" envKeyValSeparator:">"`
	Shards              map[storage.ShardName]dbConnectionName `env:"MIGRATOR_SHARDS"`
	PrevShards          map[storage.ShardName]dbConnectionName `env:"MIGRATOR_PREV_SHARDS"`
	PrevShardsStartFrom map[storage.ShardName]int64            `env:"MIGRATOR_PREV_SHARDS_START_FROM"`
	BatchLimit          int64                                  `env:"MIGRATOR_BATCH_LIMIT"`

	// список старых шардов, которые уже мигрировали - для них воркер не запускается
	ExcludedPrevShards []string `env:"MIGRATOR_EXCLUDED_PREV_SHARDS"`
}

func Run(ctx context.Context, isMigrating bool) error {
	var conf Config
	if err := env.Parse(&conf); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	dbConnections := make(map[dbConnectionName]*sql.DB)
	for name, dsn := range conf.DBConnections {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			return fmt.Errorf("init '%s' postgres db: %w", name, err)
		}
		dbConnections[name] = db
	}

	newShards := make(storage.Shards[*sql.DB], len(conf.Shards))
	for shardName, connectionName := range conf.Shards {
		if db, ok := dbConnections[connectionName]; ok {
			newShards[shardName] = db
		} else {
			return fmt.Errorf("missing connection %s for new shard %s", connectionName, shardName)
		}
	}

	wg := new(sync.WaitGroup)
	for prevShardName, prevDBConnName := range conf.PrevShards {
		if slices.Contains(conf.ExcludedPrevShards, prevShardName) {
			log.Printf("skipping migration for prev shard %s", prevShardName)
			continue
		}
		prevDB, ok := dbConnections[prevDBConnName]
		if !ok {
			return fmt.Errorf("missing connection %s for prev shard %s", prevDBConnName, prevShardName)
		}

		migrator := shardMigrator{
			prevDB:        prevDB,
			prevShardName: prevShardName,
			newShards:     newShards,
			batchLimit:    conf.BatchLimit,
			startFrom:     conf.PrevShardsStartFrom[prevShardName],
			isMigrating:   isMigrating,
		}
		wg.Go(func() {
			migrator.run(ctx)
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

func (m *shardMigrator) loop(ctx context.Context, id int64) (int64, error) {
	// select from old shard
	query := "SELECT id, user_id, name, price FROM products WHERE id > $1 ORDER BY id LIMIT $2"
	rows, err := m.prevDB.QueryContext(ctx, query, id, m.batchLimit)
	if err != nil {
		return 0, fmt.Errorf("query ctx: %w", err)
	}
	defer rows.Close()

	products := make([]models.Product, 0, m.batchLimit)
	for rows.Next() {
		var product models.Product
		if err = rows.Scan(&product.ID, &product.UserID, &product.Name, &product.Price); err != nil {
			return 0, fmt.Errorf("scan: %w", err)
		}
		products = append(products, product)
	}
	if rows.Err() != nil {
		return 0, fmt.Errorf("rows err: %w", err)
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
			err := insertToNewShard(egCtx, products, m.newShards[newShardName])
			if err != nil {
				return fmt.Errorf("failed inserting to new shard %s: %w", newShardName, err)
			}
			return nil
		})
	}
	return eg.Wait()
}

func insertToNewShard(ctx context.Context, products []models.Product, db *sql.DB) error {
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
