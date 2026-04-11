package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	mqclient "gpu-pipeline/mq/client"
)

// Publisher interface allows mocking the MQ client
type Publisher interface {
	Publish(topic string, msg mqclient.Message) (int, int, error)
}

type Streamer struct {
	client  Publisher
	config  Config
	records []Record
}

func NewStreamer(cfg Config) (*Streamer, error) {
	records, err := ReadCSV(cfg.FilePath)
	if err != nil {
		return nil, err
	}

	client := mqclient.New(cfg.BaseURL)

	return &Streamer{
		client:  client,
		config:  cfg,
		records: records,
	}, nil
}

// NewStreamerWithClient creates a streamer with a custom publisher (for testing)
func NewStreamerWithClient(cfg Config, records []Record, client Publisher) *Streamer {
	return &Streamer{
		client:  client,
		config:  cfg,
		records: records,
	}
}

// Start begins streaming in an infinite loop (original behavior)
func (s *Streamer) Start() {
	log.Println("Starting telemetry streamer...")

	for {
		s.StreamBatch(context.Background())
	}
}

// StartWithContext starts streaming with context support for graceful shutdown
func (s *Streamer) StartWithContext(ctx context.Context) {
	log.Println("Starting telemetry streamer with context...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Streamer stopped")
			return
		default:
			s.StreamBatch(ctx)
		}
	}
}

// StreamBatch processes a single batch of records (testable)
func (s *Streamer) StreamBatch(ctx context.Context) error {
	for _, rec := range s.records {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Process and publish record
		if err := s.publishRecord(rec); err != nil {
			log.Printf("publish error: %v", err)
			continue
		}

		time.Sleep(s.config.Interval)
	}
	return nil
}

// publishRecord publishes a single record to the message queue (testable)
func (s *Streamer) publishRecord(rec Record) error {
	// Create a copy to avoid modifying original record
	recordCopy := make(Record)
	for k, v := range rec {
		recordCopy[k] = v
	}

	// Add timestamp
	recordCopy["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	// Marshal to JSON
	payload, err := json.Marshal(recordCopy)
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}

	// Get partition key
	key := recordCopy["gpu_id"]
	if key == "" {
		return fmt.Errorf("missing gpu_id field")
	}

	// Publish to MQ
	_, _, err = s.client.Publish(s.config.Topic, mqclient.Message{
		Key:     key,
		Payload: payload,
	})

	if err != nil {
		return fmt.Errorf("publish failed: %w", err)
	}

	log.Printf("published record for gpu_id: %s", key)
	return nil
}

// GetRecords returns the records being streamed (for testing)
func (s *Streamer) GetRecords() []Record {
	return s.records
}

// GetConfig returns the streamer configuration (for testing)
func (s *Streamer) GetConfig() Config {
	return s.config
}
