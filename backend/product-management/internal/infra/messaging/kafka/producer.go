package kafka

import (
	"context"
	"fmt"
	"product-management/internal/app/models"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	w *kafka.Writer
}

func NewProducer(brokers []string) *Producer {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  brokers,
		Balancer: kafka.Murmur2Balancer{},
	})
	return &Producer{w: writer}
}

func (p *Producer) ProduceEvent(ctx context.Context, event *models.OutboxEvent) error {
	topic := ""
	switch event.Type {
	case models.ProductCreated:
		topic = "product-management.product"
	default:
		return fmt.Errorf("invalid event type: %s", event.Type)
	}
	return p.w.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(event.Key),
		Value: []byte(event.Payload),
	})
}

func (p *Producer) Close() error {
	return p.w.Close()
}
