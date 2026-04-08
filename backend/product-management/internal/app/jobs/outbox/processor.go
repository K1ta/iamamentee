package outbox

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"maps"
	"product-management/internal/infra/messaging/kafka"
	"product-management/internal/infra/storage"
	"product-management/internal/infra/storage/postgres"
	"product-management/internal/pkg/tx"
	"time"

	"golang.org/x/sync/errgroup"
)

type Processor struct {
	shards            storage.Shards[*sql.DB]
	producer          *kafka.Producer
	pauseWhenNoWork   time.Duration
	outboxMaxAttempts int
}

func NewProcessor(
	shardMaps []storage.Shards[*sql.DB],
	producer *kafka.Producer,
	pauseWhenNoWork time.Duration,
	outboxMaxAttempts int,
) (*Processor, error) {
	shards := make(storage.Shards[*sql.DB])
	for _, shardMap := range shardMaps {
		maps.Copy(shards, shardMap)
	}
	if len(shards) == 0 {
		return nil, errors.New("no shards")
	}
	return &Processor{
		shards:            shards,
		producer:          producer,
		pauseWhenNoWork:   pauseWhenNoWork,
		outboxMaxAttempts: outboxMaxAttempts,
	}, nil
}

func (p *Processor) Run(ctx context.Context) error {
	eg, egCtx := errgroup.WithContext(ctx)
	for name, shard := range p.shards {
		runner := &shardRunner{
			shardName:         name,
			db:                shard,
			producer:          p.producer,
			pauseWhenNoWork:   p.pauseWhenNoWork,
			outboxMaxAttempts: p.outboxMaxAttempts,
		}
		eg.Go(func() error {
			log.Println("starting runner for shard", name)
			return runner.Run(egCtx)
		})
	}
	return eg.Wait()
}

type shardRunner struct {
	shardName         storage.ShardName
	db                *sql.DB
	producer          *kafka.Producer
	pauseWhenNoWork   time.Duration
	outboxMaxAttempts int
}

func (p *shardRunner) Run(ctx context.Context) error {
	for {
		hadWork := true
		if err := tx.Run(ctx, p.db, p.loop); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			hadWork = false
		}

		timeToWait := p.pauseWhenNoWork
		if hadWork {
			timeToWait = 0
		}

		select {
		case <-time.After(timeToWait):
		case <-ctx.Done():
			return nil
		}
	}
}

func (p *shardRunner) loop(ctx context.Context, tx *sql.Tx) error {
	repo := postgres.NewOutboxRepository(tx, p.outboxMaxAttempts)

	event, err := repo.SelectOneToSend(ctx)
	if err != nil {
		return fmt.Errorf("select event from shard %s: %w", p.shardName, err)
	}

	err = p.producer.ProduceEvent(ctx, event)
	if err != nil {
		log.Printf("failed to publish event %s/%d: %v", p.shardName, event.ID, err)
		err = repo.IncreaseAttemts(ctx, event.ID)
		if err != nil {
			return fmt.Errorf("failed to increase attemts for event %s/%d: %w", p.shardName, event.ID, err)
		}
	} else {
		err = repo.MarkAsSent(ctx, event.ID)
		if err != nil {
			return fmt.Errorf("failed to mark event %s/%d as sent: %w", p.shardName, event.ID, err)
		}
		log.Printf("event %s/%d sent", p.shardName, event.ID)
	}
	return nil
}
