package main

import (
	"gpu-pipeline/streamer/internal"
	"log"
	"os"
	"strconv"
	"time"
)

func LoadConfig() internal.Config {
	intervalMs, _ := strconv.Atoi(getEnv("STREAM_INTERVAL_MS", "500"))

	return internal.Config{
		FilePath: getEnv("CSV_FILE", "/data/telemetry.csv"),
		Topic:    getEnv("TOPIC", "telemetry"),
		BaseURL:  getEnv("MQ_URL", "http://mq-service:8080"),
		Interval: time.Duration(intervalMs) * time.Millisecond,
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func main() {
	// Load config from ENV
	cfg := LoadConfig()

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
