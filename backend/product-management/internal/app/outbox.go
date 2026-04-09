package app

import (
	"context"
	"database/sql"
	"log"
	"product-management/internal/app/jobs/outbox"
	"product-management/internal/infra/messaging/kafka"
	"sync"
	"time"
)

type OutboxApp struct {
	processor     *outbox.Processor
	kafkaProducer *kafka.Producer
	dbs           map[string]*sql.DB
}

func NewOutboxApp(processor *outbox.Processor, kafkaProducer *kafka.Producer, dbs map[string]*sql.DB) *OutboxApp {
	return &OutboxApp{processor: processor, kafkaProducer: kafkaProducer, dbs: dbs}
}

func (app *OutboxApp) Run(ctx context.Context) error {
	log.Println("outbox processor is running")
	err := app.processor.Run(ctx)
	app.shutdown()
	return err
}

func (app *OutboxApp) shutdown() {
	log.Println("shutting down outbox processor")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	wg := &sync.WaitGroup{}
	for name, db := range app.dbs {
		wg.Go(func() {
			if err := db.Close(); err != nil {
				log.Printf("failed to close %s db: %v", name, err)
			}
		})
	}
	wg.Go(func() {
		if err := app.kafkaProducer.Close(); err != nil {
			log.Printf("failed to close kafka producer: %v", err)
		}
	})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		log.Println("shutting down context timeout")
	}
}
