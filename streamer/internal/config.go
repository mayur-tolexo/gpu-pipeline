package internal

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	FilePath       string
	Topic          string
	Interval       time.Duration
	BaseURL        string
	WaitTimeout    time.Duration
	WaitRetryDelay time.Duration
}

func LoadConfig() Config {
	intervalMs, _ := strconv.Atoi(getEnv("STREAM_INTERVAL_MS", "5000"))
	waitTimeoutSec, _ := strconv.Atoi(getEnv("CSV_WAIT_TIMEOUT_SEC", "120"))
	waitRetryMs, _ := strconv.Atoi(getEnv("CSV_WAIT_RETRY_MS", "1000"))

	return Config{
		FilePath:       getEnv("CSV_FILE", "/data/telemetry.csv"),
		Topic:          getEnv("TOPIC", "telemetry"),
		BaseURL:        getEnv("MQ_URL", "http://mq-service:8080"),
		Interval:       time.Duration(intervalMs) * time.Millisecond,
		WaitTimeout:    time.Duration(waitTimeoutSec) * time.Second,
		WaitRetryDelay: time.Duration(waitRetryMs) * time.Millisecond,
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

