package mq

import (
	"errors"
	"fmt"
	"sync"
)

// Queue is the top-level in-memory message queue. It manages topics and provides
// thread-safe operations for topic lifecycle and message publish/consume.
// This core package has no HTTP dependency to keep transport agnostic.

type Queue struct {
	mu     sync.RWMutex
	topics map[string]*Topic
	// default partition capacity if topic created without explicit capacity
	defaultPartitionCapacity int
}

// NewQueue creates a new Queue.
func NewQueue(defaultPartitionCapacity int) *Queue {
	return &Queue{topics: make(map[string]*Topic), defaultPartitionCapacity: defaultPartitionCapacity}
}

// Errors
var (
	ErrTopicExists    = errors.New("topic already exists")
	ErrTopicNotFound  = errors.New("topic not found")
	ErrInvalidArg     = errors.New("invalid argument")
	ErrPartitionRange = errors.New("partition out of range")
)

// CreateTopic creates a topic with given name and partitions. Returns error if topic exists.
func (q *Queue) CreateTopic(name string, partitions int, partitionCapacity int) error {
	if name == "" || partitions <= 0 {
		return ErrInvalidArg
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, ok := q.topics[name]; ok {
		return ErrTopicExists
	}
	// build topic
	t := &Topic{name: name, partitions: make([]*Partition, partitions)}
	cap := partitionCapacity
	if cap == 0 {
		cap = q.defaultPartitionCapacity
	}
	for i := 0; i < partitions; i++ {
		t.partitions[i] = NewPartition(fmt.Sprintf("%s-%d", name, i), cap)
	}
	// default partitioner: simple hash
	t.partitioner = func(key string) int {
		if key == "" {
			return 0
		}
		h := fnv32(key)
		return int(h % uint32(len(t.partitions)))
	}
	q.topics[name] = t
	return nil
}

// GetTopic returns a topic by name or error
func (q *Queue) GetTopic(name string) (*Topic, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if t, ok := q.topics[name]; ok {
		return t, nil
	}
	return nil, ErrTopicNotFound
}

// Publish validates and publishes a message to a topic.
func (q *Queue) Publish(topic string, msg Message) (int, int64, error) {
	if msg.Key == "" { return -1, -1, ErrInvalidArg }
	q.mu.RLock()
	t, ok := q.topics[topic]
	q.mu.RUnlock()
	if !ok { return -1, -1, ErrTopicNotFound }
	idx, off, err := t.Produce(msg)
	return idx, off, err
}

// Consume reads messages from a partition. Validates inputs.
func (q *Queue) Consume(topic string, group string, partition int, batch int) ([]Message, error) {
	if group == "" { return []Message{}, ErrInvalidArg }
	if batch <= 0 { return []Message{}, ErrInvalidArg }
	q.mu.RLock()
	t, ok := q.topics[topic]
	q.mu.RUnlock()
	if !ok { return []Message{}, ErrTopicNotFound }
	if partition < 0 || partition >= t.Partitions() { return []Message{}, ErrPartitionRange }
	part := t.GetPartition(partition)
	start := part.Offset(group)
	msgs, err := part.ReadFrom(start, batch)
	return msgs, err
}

// Ack commits offset (group) for a partition
func (q *Queue) Ack(topic string, group string, partition int, offset int64) error {
	q.mu.RLock()
	t, ok := q.topics[topic]
	q.mu.RUnlock()
	if !ok { return ErrTopicNotFound }
	if partition < 0 || partition >= t.Partitions() { return ErrPartitionRange }
	part := t.GetPartition(partition)
	return part.Commit(group, offset)
}

// ListTopics returns names (helper)
func (q *Queue) ListTopics() []string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	out := make([]string, 0, len(q.topics))
	for k := range q.topics { out = append(out, k) }
	return out
}

// CompactTopic compacts all partitions in a topic using watermark-based cleanup.
// It removes messages that have been consumed by all consumer groups in that topic.
// Returns a map of partition IDs to the number of messages compacted, and any error.
func (q *Queue) CompactTopic(topicName string) (map[string]int64, error) {
	q.mu.RLock()
	t, ok := q.topics[topicName]
	q.mu.RUnlock()
	if !ok {
		return nil, ErrTopicNotFound
	}

	results := make(map[string]int64)
	for i := 0; i < t.Partitions(); i++ {
		part := t.GetPartition(i)
		if part == nil {
			continue
		}
		compacted, err := part.Compact()
		if err != nil {
			return results, err
		}
		results[part.ID()] = compacted
	}
	return results, nil
}

// GetPartitionStats returns statistics for a partition (useful for monitoring).
// This includes: number of messages, tail offset, min consumer offset (watermark).
func (q *Queue) GetPartitionStats(topicName string, partitionIdx int) (map[string]interface{}, error) {
	q.mu.RLock()
	t, ok := q.topics[topicName]
	q.mu.RUnlock()
	if !ok {
		return nil, ErrTopicNotFound
	}

	if partitionIdx < 0 || partitionIdx >= t.Partitions() {
		return nil, ErrPartitionRange
	}

	part := t.GetPartition(partitionIdx)
	if part == nil {
		return nil, errors.New("partition not found")
	}

	stats := map[string]interface{}{
		"id":              part.ID(),
		"message_count":   part.Len(),
		"tail_offset":     part.TailOffset(),
		"min_consumer_offset": part.GetMinConsumerOffset(),
		"compactable_count": part.GetMinConsumerOffset(),
	}
	return stats, nil
}

// helper: consistent fnv32
func fnv32(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
