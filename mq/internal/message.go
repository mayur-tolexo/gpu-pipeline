package mq

import "time"

type Message struct {
	ID        string
	Key       string // partitioning key
	Payload   []byte // opaque data
	Timestamp time.Time
}
