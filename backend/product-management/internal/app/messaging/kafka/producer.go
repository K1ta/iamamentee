package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"product-management/internal/app/models"
	"strconv"

	"github.com/segmentio/kafka-go"
)

type ProductEvent struct {
	Type string          `json:"type"`
	Body *models.Product `json:"product"`
}

type ProductProducer struct {
	w *kafka.Writer
}

func NewKafkaProductProducer(brokers []string) *ProductProducer {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: brokers,
		Topic:   "product-management.product",
	})
	return &ProductProducer{w: writer}
}

func (p *ProductProducer) ProduceEvent(ctx context.Context, eventType string, product *models.Product) error {
	if product.ID == 0 {
		return errors.New("empty ID")
	}
	event, err := json.Marshal(ProductEvent{
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

func (p *ProductProducer) Close() error {
	return p.w.Close()
}
