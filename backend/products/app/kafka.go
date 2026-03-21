package app

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/segmentio/kafka-go"
)

const (
	KafkaProductEventTypeCreated = "created"
)

type KafkaProductEvent struct {
	Type string   `json:"type"`
	Body *Product `json:"product"`
}

func ConsumeProductEvents(ctx context.Context, wg *sync.WaitGroup, repo *SearchRepository, brokers []string) {
	defer wg.Done()
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		GroupID: "products.product",
		Topic:   "product-management.product",
	})
	defer reader.Close()

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("closing consumer", reader.Config().Topic)
				return
			}
			log.Println("failed to read message from", reader.Config().Topic, ":", err)
			continue
		}
		log.Println("message in topic", reader.Config().Topic, "received:", string(msg.Value))

		var event KafkaProductEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Println("failed to unmarshal message", msg.Partition, msg.Offset, "from", reader.Config().Topic, ":", err)
			continue
		}

		// todo switch по типу, но пока одно событие
		if err := repo.Create(ctx, event.Body); err != nil {
			log.Println("failed to create product", event.Body.ID, ":", err)
		}
	}
}
