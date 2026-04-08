package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"product-management/internal/app"
	"product-management/internal/app/jobs/shardmigrator"
	"syscall"
)

const (
	cmdMigrateShards = "migrate-shards"
	cmdCleanupShards = "cleanup-shards"
	cmdOutbox        = "outbox"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		cancel()
	}()

	// Особые режимы запуска - миграция шардов и очистка старых шардов
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case cmdMigrateShards:
			log.Println("starting migrator in migration mode")
			if err := shardmigrator.Run(ctx, true); err != nil {
				log.Println("failed to run migrator:", err)
			}
		case cmdCleanupShards:
			log.Println("starting migrator in cleanup mode")
			if err := shardmigrator.Run(ctx, false); err != nil {
				log.Println("failed to run migrator:", err)
			}
		case cmdOutbox:
			log.Println("starting outbox processor")
			app, err := app.NewOutboxApp(ctx)
			if err != nil {
				log.Fatalln("failed to create new outbox app:", err)
			}
			if err := app.Run(ctx); err != nil {
				log.Println("failed to run outbox app:", err)
			}
		}
		return
	}

	app, err := app.New()
	if err != nil {
		log.Fatalln("failed to create new app:", err)
	}
	if err := app.Run(ctx); err != nil {
		log.Println("failed to run service:", err)
	}
}
