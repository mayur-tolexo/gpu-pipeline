package mq

import (
	"errors"
	"log"
	"sync"
	"sync/atomic"
)

// ErrOffsetOutOfRange is returned when a requested offset is invalid.
var ErrOffsetOutOfRange = errors.New("offset out of range")

// ErrCommitTooLarge is returned when committing an offset beyond the tail.
var ErrCommitTooLarge = errors.New("commit offset beyond tail")

// ErrPartitionFull is returned when a partition reached its capacity.
var ErrPartitionFull = errors.New("partition full")

// Partition is an append-only log of messages with per-consumer offsets.
// It's intentionally simple so it can be extended later (persistence, WAL, etc.).
// Partition-level locking provides concurrency isolation: each partition can
// be appended/read independently without a global lock, enabling parallelism
// across partitions.
type Partition struct {
	mu         sync.RWMutex
	messages   []Message
	maxMessage int // <=0 means unlimited
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

// NewPartition creates a new partition with the given id and optional capacity.
func NewPartition(id string, capacity int) *Partition {
	return &Partition{
		messages:   make([]Message, 0),
		maxMessage: capacity,
		id:         id,
	}
}

// ID returns the partition id.
func (p *Partition) ID() string {
	return p.id
}

// Append adds a message to the partition and returns the offset of the appended message.
// Returns ErrPartitionFull if a capacity was configured and reached.
func (p *Partition) Append(msg Message) (int64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.maxMessage > 0 && len(p.messages) >= p.maxMessage {
		log.Printf("❌ [PARTITION %s] APPEND FAILED: partition full (capacity=%d, current=%d)", 
			p.id, p.maxMessage, len(p.messages))
		return -1, ErrPartitionFull
	}
	offset := int64(len(p.messages))
	p.messages = append(p.messages, msg)
	log.Printf("✅ [PARTITION %s] MESSAGE APPENDED: offset=%d, key=%s, messages_count=%d", 
		p.id, offset, msg.Key, len(p.messages))
	return offset, nil
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
// This returns an empty slice (not nil) when offset == tail to make callers simpler.
func (p *Partition) ReadFrom(offset int64, max int) ([]Message, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	tail := int64(len(p.messages))
	
	// Validate offset range
	if offset < 0 {
		log.Printf("❌ [PARTITION %s] READ FAILED: invalid offset (offset=%d, must be >= 0)", p.id, offset)
		return []Message{}, ErrOffsetOutOfRange
	}
	if offset > tail {
		log.Printf("❌ [PARTITION %s] READ FAILED: offset out of range (offset=%d, tail=%d)", 
			p.id, offset, tail)
		return []Message{}, ErrOffsetOutOfRange
	}
	
	// If reading from tail, return empty slice (no messages available)
	if offset == tail {
		log.Printf("📭 [PARTITION %s] READ: no messages available (offset=tail=%d)", p.id, offset)
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
	
	messagesRead := end - start
	log.Printf("📖 [PARTITION %s] CONSUMED: offset=%d, messages_read=%d, batch_size=%d, tail=%d", 
		p.id, offset, messagesRead, max, tail)
	
	// Return a copy to avoid data races if caller mutates
	out := make([]Message, end-start)
	copy(out, p.messages[start:end])
	return out, nil
}

// Commit records a consumer group's offset. The committed offset must be between 0 and TailOffset().
// This implementation avoids holding the partition-wide lock while updating per-consumer offsets.
// Returns ErrCommitTooLarge if offset > tail or if offset is negative.
func (p *Partition) Commit(consumer string, offset int64) error {
	// Validate consumer group name
	if consumer == "" {
		log.Printf("❌ [PARTITION %s] COMMIT FAILED: empty consumer group name", p.id)
		return errors.New("consumer group name cannot be empty")
	}
	
	// read tail under RLock only to validate bounds
	p.mu.RLock()
	tail := int64(len(p.messages))
	p.mu.RUnlock()
	
	// Validate offset range
	if offset < 0 {
		log.Printf("❌ [PARTITION %s] COMMIT FAILED: negative offset (consumer=%s, offset=%d)", 
			p.id, consumer, offset)
		return ErrCommitTooLarge
	}
	if offset > tail {
		log.Printf("❌ [PARTITION %s] COMMIT FAILED: offset beyond tail (consumer=%s, offset=%d, tail=%d)", 
			p.id, consumer, offset, tail)
		return ErrCommitTooLarge
	}
	
	// try to load existing atomic counter
	if v, ok := p.offsets.Load(consumer); ok {
		ai := v.(*atomic.Int64)
		oldOffset := ai.Load()
		ai.Store(offset)
		log.Printf("✅ [PARTITION %s] OFFSET COMMITTED: consumer=%s, old_offset=%d, new_offset=%d, tail=%d", 
			p.id, consumer, oldOffset, offset, tail)
		return nil
	}
	// create and store a new atomic.Int64
	ai := new(atomic.Int64)
	ai.Store(offset)
	p.offsets.Store(consumer, ai)
	log.Printf("✅ [PARTITION %s] NEW CONSUMER REGISTERED: consumer=%s, offset=%d, tail=%d", 
		p.id, consumer, offset, tail)
	return nil
}

// Offset returns the committed offset for a consumer group. If not present, returns 0.
func (p *Partition) Offset(consumer string) int64 {
	if v, ok := p.offsets.Load(consumer); ok {
		return v.(*atomic.Int64).Load()
	}
	return 0
}

// GetMinConsumerOffset finds the minimum committed offset across all consumer groups.
// This represents the watermark - messages before this offset can be safely deleted
// since all consumers have progressed past them.
func (p *Partition) GetMinConsumerOffset() int64 {
	var minOffset int64 = -1

	p.offsets.Range(func(key, value interface{}) bool {
		offset := value.(*atomic.Int64).Load()
		if minOffset == -1 || offset < minOffset {
			minOffset = offset
		}
		return true
	})

	// If no consumers have committed yet, return 0 (nothing can be cleaned)
	if minOffset == -1 {
		return 0
	}

	return minOffset
}

// Compact removes messages before the watermark (minimum committed offset).
// It adjusts all consumer offsets accordingly. Returns the number of messages compacted.
// This is a watermark-based cleanup strategy: messages are only deleted once ALL
// consumer groups have consumed them.
func (p *Partition) Compact() (int64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.messages) == 0 {
		log.Printf("🔄 [PARTITION %s] COMPACT: no messages to compact", p.id)
		return 0, nil
	}

	// Find minimum offset across all consumer groups
	var minOffset int64 = -1

	p.offsets.Range(func(key, value interface{}) bool {
		offset := value.(*atomic.Int64).Load()
		if minOffset == -1 || offset < minOffset {
			minOffset = offset
		}
		return true
	})

	// If no consumers or all at the beginning, nothing to clean
	if minOffset == -1 || minOffset <= 0 {
		log.Printf("🔄 [PARTITION %s] COMPACT: no compactable messages (minOffset=%d)", p.id, minOffset)
		return 0, nil
	}

	// Ensure minOffset doesn't exceed available messages
	if minOffset > int64(len(p.messages)) {
		minOffset = int64(len(p.messages))
	}

	// Nothing to compact if minOffset is 0
	if minOffset == 0 {
		log.Printf("🔄 [PARTITION %s] COMPACT: nothing to compact (minOffset=0)", p.id)
		return 0, nil
	}

	compactedCount := minOffset
	beforeCount := len(p.messages)

	// Remove messages up to minOffset
	p.messages = p.messages[minOffset:]

	// Adjust all consumer offsets by subtracting the compacted count
	p.offsets.Range(func(key, value interface{}) bool {
		ai := value.(*atomic.Int64)
		oldOffset := ai.Load()
		newOffset := oldOffset - minOffset
		if newOffset < 0 {
			newOffset = 0
		}
		ai.Store(newOffset)
		return true
	})

	log.Printf("🗑️  [PARTITION %s] COMPACTED: watermark=%d, messages_removed=%d, before=%d, after=%d", 
		p.id, minOffset, compactedCount, beforeCount, len(p.messages))

	return compactedCount, nil
}
