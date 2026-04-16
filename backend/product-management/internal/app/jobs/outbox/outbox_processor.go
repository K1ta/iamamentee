package outbox

import (
	"context"
	"errors"
	"fmt"
	"log"
	"product-management/internal/app/domain"
	"product-management/internal/infra/messaging/kafka"
	"product-management/internal/pkg/sharding"
	"time"

	"golang.org/x/sync/errgroup"
)

type Repository interface {
	SelectBatchToSend(ctx context.Context) ([]domain.OutboxEvent, error)
	MarkBatchAsSent(ctx context.Context, ids []int64) error
}

type Processor struct {
	shards          map[sharding.ShardName]Repository
	producer        *kafka.Producer
	pauseWhenNoWork time.Duration
}

func NewProcessor(
	shards map[sharding.ShardName]Repository,
	producer *kafka.Producer,
	pauseWhenNoWork time.Duration,
) (*Processor, error) {
	if len(shards) == 0 {
		return nil, errors.New("no shards")
	}
	return &Processor{
		shards:          shards,
		producer:        producer,
		pauseWhenNoWork: pauseWhenNoWork,
	}, nil
}

func (p *Processor) Run(ctx context.Context) error {
	eg, egCtx := errgroup.WithContext(ctx)
	for name, repo := range p.shards {
		runner := &shardRunner{
			shardName:       name,
			repo:            repo,
			producer:        p.producer,
			pauseWhenNoWork: p.pauseWhenNoWork,
		}
		eg.Go(func() error {
			log.Println("starting runner for shard", name)
			return runner.Run(egCtx)
		})
	}
	return eg.Wait()
}

type shardRunner struct {
	shardName       sharding.ShardName
	repo            Repository
	producer        *kafka.Producer
	pauseWhenNoWork time.Duration
}

func (p *shardRunner) Run(ctx context.Context) error {
	for {
		hadWork, err := p.loop(ctx)
		if err != nil {
			return err
		}

		if !hadWork {
			select {
			case <-time.After(p.pauseWhenNoWork):
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func (p *shardRunner) loop(ctx context.Context) (bool, error) {
	events, err := p.repo.SelectBatchToSend(ctx)
	if err != nil {
		return false, fmt.Errorf("select event from shard %s: %w", p.shardName, err)
	}
	if len(events) == 0 {
		return false, nil
	}
	if err = p.producer.ProduceEventsBatch(ctx, events); err != nil {
		return false, fmt.Errorf("produce events from %s: %w", p.shardName, err)
	}
	ids := make([]int64, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.ID)
	}
	if err = p.repo.MarkBatchAsSent(ctx, ids); err != nil {
		return false, fmt.Errorf("mark as sent on %s: %w", p.shardName, err)
	}
	log.Printf("%d events sent from %s", len(events), p.shardName)
	return true, nil
}
