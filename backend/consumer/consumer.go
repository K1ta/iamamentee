package main

import (
	"context"
	"fmt"
	"log"

	"github.com/segmentio/kafka-go"
)

func main() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{
			"kafka-0.kafka.infra.svc.cluster.local:9093",
			"kafka-1.kafka.infra.svc.cluster.local:9093",
			"kafka-2.kafka.infra.svc.cluster.local:9093",
		},
		Topic:   "test-topic",
		GroupID: "my-group",
	})

	defer reader.Close()

	fmt.Println("Consumer started. Waiting for messages...")

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Fatalf("Failed to read message: %v", err)
		}
		fmt.Printf("Received: key=%s value=%s\n", string(msg.Key), string(msg.Value))
	}
}
