package internal

import "time"

type Config struct {
	FilePath string
	Topic    string
	Interval time.Duration
	BaseURL  string
}
