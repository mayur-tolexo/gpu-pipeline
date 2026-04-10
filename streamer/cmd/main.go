package main

import (
	"gpu-pipeline/streamer/internal"
	"log"
)

func main() {
	// Load config from ENV
	cfg := internal.LoadConfig()

	// Initialize streamer
	streamer, err := internal.NewStreamer(cfg)
	if err != nil {
		log.Fatalf("failed to create streamer: %v", err)
	}

	log.Printf("Config: CSV=%s, Topic=%s, MQ=%s, Interval=%v\n",
		cfg.FilePath, cfg.Topic, cfg.BaseURL, cfg.Interval)

	// Start streaming (blocking)
	streamer.Start()
}
