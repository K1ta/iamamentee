package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"product-management/cmd"
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

	if err := cmd.Run(ctx); err != nil {
		log.Println("run failed:", err)
	}
}
