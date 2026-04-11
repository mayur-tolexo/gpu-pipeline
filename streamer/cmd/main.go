package main

import (
	"fmt"
	"gpu-pipeline/streamer/internal"
	"log"
	"os"
	"time"
)

func main() {
	// Load config from ENV
	cfg := internal.LoadConfig()

	// Wait for CSV file to be available
	if err := waitForFile(cfg.FilePath, cfg.WaitTimeout, cfg.WaitRetryDelay); err != nil {
		log.Fatalf("CSV file not available: %v", err)
	}

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

// waitForFile waits for a file to exist with timeout and retry logic
func waitForFile(filePath string, timeout, retryDelay time.Duration) error {
	deadline := time.Now().Add(timeout)

	log.Printf("Waiting for CSV file: %s (timeout: %v)", filePath, timeout)

	for {
		// Check if file exists
		if _, err := os.Stat(filePath); err == nil {
			log.Printf("✓ CSV file found at %s", filePath)
			return nil
		}

		// Check timeout
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for file %s after %v", filePath, timeout)
		}

		// Retry after delay
		log.Printf("  File not found, retrying in %v...", retryDelay)
		time.Sleep(retryDelay)
	}
}
