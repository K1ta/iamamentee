package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"products/internal/domain"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sethvargo/go-retry"
)

const kafkaProductEventTypeCreated = "created"

type productEventService interface {
	CreateProduct(ctx context.Context, product *domain.Product) error
}

type kafkaProductEvent struct {
	Type string          `json:"type"`
	Body *domain.Product `json:"product"`
}

func (e *kafkaProductEvent) validate() error {
	if e == nil {
		return errors.New("event is nil")
	}
	if e.Type != kafkaProductEventTypeCreated {
		return fmt.Errorf("invalid event type: %s", e.Type)
	}
	if e.Body == nil {
		return errors.New("body is nil")
	}
	return nil
}

type ProductEventConsumer struct {
	reader         *kafka.Reader
	fetchBackoff   retry.Backoff
	processBackoff retry.Backoff
	commitBackoff  retry.Backoff
	svc            productEventService
}

func NewProductEventConsumer(brokers []string, svc productEventService) *ProductEventConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		GroupID: "products.product",
		Topic:   "product-management.product",
	})
	return &ProductEventConsumer{
		reader:         reader,
		fetchBackoff:   retry.WithMaxDuration(time.Minute, retry.WithJitterPercent(50, retry.NewExponential(time.Millisecond*100))),
		processBackoff: retry.WithMaxRetries(3, retry.WithJitterPercent(20, retry.NewExponential(time.Millisecond*200))),
		commitBackoff:  retry.WithMaxRetries(5, retry.NewConstant(time.Millisecond*100)),
		svc:            svc,
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

		var event kafkaProductEvent
		if err = json.Unmarshal(msg.Value, &event); err != nil {
			if dlqErr := c.writeToDLQ(ctx, msg, err); dlqErr != nil {
				return fmt.Errorf("write to DLQ failed: %w", dlqErr)
			}
			c.commitOffset(ctx, msg)
			continue
		}
		if err = event.validate(); err != nil {
			if dlqErr := c.writeToDLQ(ctx, msg, err); dlqErr != nil {
				return fmt.Errorf("write to DLQ failed: %w", dlqErr)
			}
			c.commitOffset(ctx, msg)
			continue
		}

		if err = c.processWithRetry(ctx, event); err != nil {
			if dlqErr := c.writeToDLQ(ctx, msg, err); dlqErr != nil {
				return fmt.Errorf("write to DLQ failed: %w", dlqErr)
			}
		}

		c.commitOffset(ctx, msg)
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

func (c *ProductEventConsumer) processWithRetry(ctx context.Context, event kafkaProductEvent) error {
	return retry.Do(ctx, c.processBackoff, func(ctx context.Context) error {
		err := c.svc.CreateProduct(ctx, event.Body)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}
			log.Println("process failed, retrying:", err)
			return retry.RetryableError(err)
		}
		return nil
	})
}

func (c *ProductEventConsumer) writeToDLQ(_ context.Context, msg kafka.Message, reason error) error {
	log.Println("message sent to DLQ", msg.Partition, msg.Offset, ", reason:", reason)
	// TODO implement writing to DLQ
	return nil
}

func (c *ProductEventConsumer) commitOffset(ctx context.Context, msg kafka.Message) {
	if err := c.commitOffsetWithRetry(ctx, msg); err != nil {
		log.Println("failed to commit message", msg.Partition, msg.Offset, ":", err)
	}
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
