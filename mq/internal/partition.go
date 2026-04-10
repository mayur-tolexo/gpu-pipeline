package mq

import (
	"errors"
	"sync"
	"sync/atomic"
)

// ErrOffsetOutOfRange is returned when a requested offset is invalid.
var ErrOffsetOutOfRange = errors.New("offset out of range")

// ErrCommitTooLarge is returned when committing an offset beyond the tail.
var ErrCommitTooLarge = errors.New("commit offset beyond tail")

// Partition is an append-only log of messages with per-consumer offsets.
// It's intentionally simple so it can be extended later (persistence, WAL, etc.).
type Partition struct {
	mu       sync.RWMutex
	messages []Message
	// committed offsets per consumer group - use sync.Map to avoid locking the
	// whole partition when different consumers commit concurrently.
	offsets sync.Map // map[string]*atomic.Int64
	id      string
}

// TODO: To add WAL support, follow these steps:
// 1. Define a WAL interface in internal/wal, e.g.:
//    type WAL interface {
//        Append(msg Message) (offset int64, err error)
//        ReadFrom(offset int64, max int) ([]Message, error)
//        Close() error
//    }
// 2. Modify Partition to accept a WAL implementation (field wal WAL).
// 3. In Append, first write to the WAL (wal.Append) and only after it's durably
//    stored, append to the in-memory slice. This ensures durability before
//    acknowledging to producers.
// 4. On startup, if wal != nil, replay entries from WAL to rebuild in-memory
//    messages slice (call ReadFrom(0, 0) or an efficient snapshot/recovery API).
// 5. Ensure WAL is properly closed on shutdown.
// The current in-memory implementation keeps the API surface stable so these
// changes can be introduced with minimal impact to callers.

// NewPartition creates a new partition with the given id.
func NewPartition(id string) *Partition {
	return &Partition{
		messages: make([]Message, 0),
		id:       id,
	}
}

// ID returns the partition id.
func (p *Partition) ID() string {
	return p.id
}

// Append adds a message to the partition and returns the offset of the appended message.
func (p *Partition) Append(msg Message) int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	offset := int64(len(p.messages))
	p.messages = append(p.messages, msg)
	return offset
}

// TailOffset returns the next offset to be assigned (i.e., length of the log).
func (p *Partition) TailOffset() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return int64(len(p.messages))
}

// Len returns the number of messages in the partition.
func (p *Partition) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.messages)
}

// Get retrieves the message at the given offset.
func (p *Partition) Get(offset int64) (Message, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if offset < 0 || offset >= int64(len(p.messages)) {
		return Message{}, ErrOffsetOutOfRange
	}
	return p.messages[offset], nil
}

// ReadFrom reads up to max messages starting from offset. If max <= 0, all messages are returned.
// Returns ErrOffsetOutOfRange when offset is greater than the tail offset.
func (p *Partition) ReadFrom(offset int64, max int) ([]Message, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	tail := int64(len(p.messages))
	if offset < 0 || offset > tail {
		return nil, ErrOffsetOutOfRange
	}
	if offset == tail {
		return []Message{}, nil
	}
	start := int(offset)
	end := len(p.messages)
	if max > 0 {
		candidate := start + max
		if candidate < end {
			end = candidate
		}
	}
	// Return a copy to avoid data races if caller mutates
	out := make([]Message, end-start)
	copy(out, p.messages[start:end])
	return out, nil
}

// Commit records a consumer group's offset. The committed offset must be between 0 and TailOffset().
// This implementation avoids holding the partition-wide lock while updating per-consumer offsets.
func (p *Partition) Commit(consumer string, offset int64) error {
	// read tail under RLock only to validate bounds
	p.mu.RLock()
	tail := int64(len(p.messages))
	p.mu.RUnlock()
	if offset < 0 || offset > tail {
		return ErrCommitTooLarge
	}
	// try to load existing atomic counter
	if v, ok := p.offsets.Load(consumer); ok {
		ai := v.(*atomic.Int64)
		ai.Store(offset)
		return nil
	}
	// create and store a new atomic.Int64
	ai := new(atomic.Int64)
	ai.Store(offset)
	p.offsets.Store(consumer, ai)
	return nil
}

// Offset returns the committed offset for a consumer group. If not present, returns 0.
func (p *Partition) Offset(consumer string) int64 {
	if v, ok := p.offsets.Load(consumer); ok {
		return v.(*atomic.Int64).Load()
	}
	return 0
}
