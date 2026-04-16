package kafka

import (
	"context"
	"fmt"
	"product-management/internal/app/domain"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	w *kafka.Writer
}

func NewProducer(brokers []string, batchSize int) *Producer {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:   brokers,
		Balancer:  kafka.Murmur2Balancer{},
		BatchSize: batchSize,
	})
	return &Producer{w: writer}
}

func (p *Producer) ProduceEventsBatch(ctx context.Context, events []domain.OutboxEvent) error {
	messages := make([]kafka.Message, 0, len(events))
	for _, event := range events {
		topic := ""
		switch event.Type {
		case domain.ProductCreated:
			topic = "product-management.product"
		default:
			return fmt.Errorf("invalid event type: %s", event.Type)
		}
		messages = append(messages, kafka.Message{
			Topic: topic,
			Key:   []byte(event.Key),
			Value: []byte(event.Payload),
		})
	}
	return p.w.WriteMessages(ctx, messages...)
}

func (p *Producer) Close() error {
	return p.w.Close()
}
