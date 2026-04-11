package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	collector "gpu-pipeline/collector/internal"
)

func main() {
	cfg := collector.LoadConfig()
	dsn := os.Getenv("DB_DSN")
	col, err := collector.NewCollector(cfg, dsn)
	if err != nil {
		log.Fatalf("failed to initialize collector: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go col.Run(ctx)

	// handle signals
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("shutting down")
	// allow graceful shutdown
	time.Sleep(1 * time.Second)
	collector.CloseStore()
}
