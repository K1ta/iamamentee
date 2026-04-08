package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sethvargo/go-retry"
)

const (
	KafkaProductEventTypeCreated = "created"
)

type KafkaProductEvent struct {
	Type string   `json:"type"`
	Body *Product `json:"product"`
}

func (e *KafkaProductEvent) Validate() error {
	if e == nil {
		return errors.New("event is nil")
	}
	if e.Type != KafkaProductEventTypeCreated {
		return fmt.Errorf("invalid event type: %s", e.Type)
	}
	if e.Body == nil {
		return errors.New("body is nil")
	}
	return nil
}

type ProductEventConsumer struct {
	reader        *kafka.Reader
	fetchBackoff  retry.Backoff
	commitBackoff retry.Backoff
	repo          SearchRepository
	store         *SearchStore
}

func NewProductEventConsumer(brokers []string, repo SearchRepository, store *SearchStore) *ProductEventConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		GroupID: "products.product",
		Topic:   "product-management.product",
	})
	return &ProductEventConsumer{
		reader:        reader,
		fetchBackoff:  retry.WithMaxDuration(time.Minute, retry.WithJitterPercent(50, retry.NewExponential(time.Millisecond*100))),
		commitBackoff: retry.WithMaxRetries(5, retry.NewConstant(time.Millisecond*100)),
		repo:          repo,
		store:         store,
	}
}

func (c *ProductEventConsumer) Run(ctx context.Context) error {
	defer func() {
		log.Println("closing kafka reader")
		if err := c.reader.Close(); err != nil {
			log.Println("failed to close kafka reader:", err)
		} else {
			log.Println("kafka reader closed")
		}
	}()

	for {
		msg, err := c.fetchWithRetry(ctx)
		if err != nil {
			return fmt.Errorf("fetch: %w", err)
		}
		log.Println("message in topic", c.reader.Config().Topic, "received:", string(msg.Value))

		var event KafkaProductEvent
		if err = json.Unmarshal(msg.Value, &event); err != nil {
			if dlqErr := c.writeToDLQ(ctx, msg, err); dlqErr != nil {
				return fmt.Errorf("write to DLQ failed: %w", err)
			}
			continue
		}
		if err := event.Validate(); err != nil {
			if dlqErr := c.writeToDLQ(ctx, msg, err); dlqErr != nil {
				return fmt.Errorf("write to DLQ failed: %w", err)
			}
			continue
		}

		if err = c.process(ctx, event); err != nil {
			if dlqErr := c.writeToDLQ(ctx, msg, err); dlqErr != nil {
				return fmt.Errorf("write to DLQ failed: %w", err)
			}
		}

		if err = c.commitOffsetWithRetry(ctx, msg); err != nil {
			log.Println("failed to commit message", msg.Partition, msg.Offset, ":", err)
			continue
		}
	}
}

func (c *ProductEventConsumer) fetchWithRetry(ctx context.Context) (kafka.Message, error) {
	return retry.DoValue(ctx, c.fetchBackoff, func(ctx context.Context) (kafka.Message, error) {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return kafka.Message{}, err
			}
			log.Println("failed to read message from", c.reader.Config().Topic, ":", err)
			return kafka.Message{}, retry.RetryableError(err)
		}
		return msg, nil
	})
}

func (c *ProductEventConsumer) process(ctx context.Context, event KafkaProductEvent) error {
	// todo switch по типу, но пока одно событие
	if err := c.repo.Create(ctx, event.Body); err != nil {
		return fmt.Errorf("failed to create product %d: %w", event.Body.ID, err)
	}
	if err := c.store.Index(ctx, event.Body); err != nil {
		return fmt.Errorf("failed to index product %d: %w", event.Body.ID, err)
	}
	return nil
}

func (c *ProductEventConsumer) writeToDLQ(_ context.Context, msg kafka.Message, reason error) error {
	log.Println("message sent to DLQ", msg.Partition, msg.Offset, ", reason:", reason)
	// TODO implement writing to DLQ
	return nil
}

func (c *ProductEventConsumer) commitOffsetWithRetry(ctx context.Context, msg kafka.Message) error {
	return retry.Do(ctx, c.commitBackoff, func(ctx context.Context) error {
		err := c.reader.CommitMessages(ctx, msg)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}
			return retry.RetryableError(err)
		}
		return nil
	})
}
