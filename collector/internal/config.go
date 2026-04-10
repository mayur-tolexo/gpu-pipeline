package internal

import (
	"os"
	"strconv"
	"time"
)

// Config holds collector configuration from environment variables.
type Config struct {
	BaseURL      string
	Topic        string
	Group        string
	Partitions   int
	BatchSize    int
	PollInterval time.Duration
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() Config {
	partitions, _ := strconv.Atoi(getEnv("PARTITIONS", "3"))
	batch, _ := strconv.Atoi(getEnv("BATCH_SIZE", "10"))
	pollMs, _ := strconv.Atoi(getEnv("POLL_INTERVAL_MS", "500"))

	return Config{
		BaseURL:      getEnv("MQ_URL", "http://mq-service:8080"),
		Topic:        getEnv("TOPIC", "telemetry"),
		Group:        getEnv("GROUP", "collector-group"),
		Partitions:   partitions,
		BatchSize:    batch,
		PollInterval: time.Duration(pollMs) * time.Millisecond,
	}
}

func getEnv(k, d string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return d
}
