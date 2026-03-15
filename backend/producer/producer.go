package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

func main() {
	log.Println("Hello!")

	// Подключаемся к Kafka брокеру
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{
			"kafka-client.infra:9092",
			// "kafka-0.kafka.infra.svc.cluster.local:9092",
			// "kafka-1.kafka.infra.svc.cluster.local:9092",
			// "kafka-2.kafka.infra.svc.cluster.local:9092",
		},
		Topic:    "test-topic",
		Balancer: &kafka.LeastBytes{},
	})

	defer writer.Close()

	time.Sleep(3 * time.Second) // ждем пока Kafka стабилизируется
	log.Println("start producing")

	i := 0
	for {
		msg := fmt.Sprintf("Hello Kafka message %d", i)
		err := writer.WriteMessages(context.Background(),
			kafka.Message{
				Value: []byte(msg),
			},
		)
		if err != nil {
			log.Printf("Failed to write message: %v", err)
			time.Sleep(time.Second)
			continue
		}
		fmt.Println("Sent:", msg)
		time.Sleep(time.Second * 5)
	}
}
