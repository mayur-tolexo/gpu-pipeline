package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	clientpkg "gpu-pipeline/mq/client"
)

// Collector orchestrates polling MQ, processing messages and storing them.
// It relies on InitStore/GetStore in this package for DB access.
type Collector struct {
	cfg   Config
	cli   MQClient
	dbDSN string
}

// MQClient defines the minimal client interface used by the collector.
// This allows testing with a fake client.
type MQClient interface {
	Consume(topic, group string, partition, batch int) (*clientpkg.ConsumeResponse, error)
	Ack(topic string, req clientpkg.AckRequest) error
}

// NewCollector creates a Collector. It initializes DB singleton using provided dsn.
// If dbDSN is empty, DB initialization is skipped and InsertFunc must be set by caller.
func NewCollector(cfg Config, dbDSN string) (*Collector, error) {
	// default client uses real HTTP client
	cli := clientpkg.New(cfg.BaseURL)
	return NewCollectorWithClient(cfg, cli, dbDSN)
}

// NewCollectorWithClient constructs a Collector with a provided MQClient (for tests or custom clients).
func NewCollectorWithClient(cfg Config, cli MQClient, dbDSN string) (*Collector, error) {
	if dbDSN != "" {
		if err := InitStore(dbDSN); err != nil {
			return nil, fmt.Errorf("init store: %w", err)
		}
	}
	return &Collector{cfg: cfg, cli: cli, dbDSN: dbDSN}, nil
}

// Run starts the collector loop until ctx is canceled.
func (c *Collector) Run(ctx context.Context) {
	log.Printf("collector: starting for topic=%s group=%s partitions=%d", c.cfg.Topic, c.cfg.Group, c.cfg.Partitions)
	for {
		select {
		case <-ctx.Done():
			log.Printf("collector: stopping")
			return
		default:
		}

		for p := 0; p < c.cfg.Partitions; p++ {
			// fetch batch from MQ
			res, err := c.cli.Consume(c.cfg.Topic, c.cfg.Group, p, c.cfg.BatchSize)
			if err != nil {
				log.Printf("collector: consume partition=%d error: %v", p, err)
				continue
			}

			if res == nil || len(res.Messages) == 0 {
				// nothing to do
				continue
			}

			// process messages sequentially (idempotent Insert)
			for _, msg := range res.Messages {
				var rec map[string]interface{}
				if err := json.Unmarshal(msg.Payload, &rec); err != nil {
					log.Printf("collector: failed to parse payload: %v", err)
					continue
				}

				// call InsertFunc which may be overridden in tests
				if err := InsertFunc(rec); err != nil {
					log.Printf("collector: failed to insert record: %v", err)
					continue
				}
			}

			// ack up to NextOffset
			ackReq := clientpkg.AckRequest{Group: c.cfg.Group, Partition: p, Offset: res.NextOffset}
			if err := c.cli.Ack(c.cfg.Topic, ackReq); err != nil {
				log.Printf("collector: ack failed partition=%d offset=%d: %v", p, res.NextOffset, err)
			}
		}

		// sleep until next poll
		time.Sleep(c.cfg.PollInterval)
	}
}
