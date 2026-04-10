package internal

import (
	"encoding/json"
	"log"
	"time"

	mqclient "gpu-pipeline/mq/client"
)

type Streamer struct {
	client  *mqclient.Client
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

func (s *Streamer) Start() {
	log.Println("Starting telemetry streamer...")

	for {
		for _, rec := range s.records {

			rec["timestamp"] = time.Now().UTC().Format(time.RFC3339)

			payload, _ := json.Marshal(rec)

			// Choose partition key
			key := rec["gpu_id"] // assumes CSV has gpu_id column

			_, _, err := s.client.Publish(s.config.Topic, mqclient.Message{
				Key:     key,
				Payload: payload,
			})

			if err != nil {
				log.Println("publish error:", err)
				continue
			}

			time.Sleep(s.config.Interval)
		}
	}
}
