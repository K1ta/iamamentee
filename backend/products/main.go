package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"products/app"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		cancel()
	}()

	if err := app.Run(ctx); err != nil {
		log.Println("failed to run service:", err)
	}
}
