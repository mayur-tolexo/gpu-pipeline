package mq

import "time"

// Message represents an opaque payload with a partitioning key.
type Message struct {
	ID        string
	Key       string // partitioning key
	Payload   []byte // opaque data
	Timestamp time.Time
}
