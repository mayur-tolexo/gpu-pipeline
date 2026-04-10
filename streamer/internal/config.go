package internal

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	FilePath string
	Topic    string
	Interval time.Duration
	BaseURL  string
}

func LoadConfig() Config {
	intervalMs, _ := strconv.Atoi(getEnv("STREAM_INTERVAL_MS", "5000"))

	return Config{
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
