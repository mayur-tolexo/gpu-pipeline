package mq

import "time"

// Message represents an opaque payload with a partitioning key.
type Message struct {
	Key       string // partitioning key
	Payload   []byte // opaque data
	Timestamp time.Time
}
