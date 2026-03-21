package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/segmentio/kafka-go"
)

const (
	KafkaProductEventTypeCreated = "created"
)

type KafkaProductEvent struct {
	Type string   `json:"type"`
	Body *Product `json:"product"`
}

type KafkaProductProducer struct {
	w *kafka.Writer
}

func NewKafkaProductProducer(brokers []string) *KafkaProductProducer {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: brokers,
		Topic:   "product-management.product",
	})
	return &KafkaProductProducer{w: writer}
}

func (p *KafkaProductProducer) ProduceEvent(ctx context.Context, eventType string, product *Product) error {
	if product.ID == 0 {
		return errors.New("empty ID")
	}
	event, err := json.Marshal(KafkaProductEvent{
		Type: eventType,
		Body: product,
	})
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return p.w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(strconv.FormatInt(product.ID, 10)),
		Value: event,
	})
}

func (p *KafkaProductProducer) Close() error {
	return p.w.Close()
}
