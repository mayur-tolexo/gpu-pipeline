package mq

import (
	"errors"
	"sync"
)

// ConsumerGroup manages multiple consumers for a topic with coordinated offset tracking.
// Each consumer group tracks offsets per partition independently.
type ConsumerGroup struct {
	name       string
	topic      *Topic
	partitions map[int]*PartitionConsumer // [partitionIdx]*PartitionConsumer
	mu         sync.RWMutex
}

// PartitionConsumer tracks committed offset for a partition within a group.
type PartitionConsumer struct {
	partition int
	group     string
	// committed offset
	offset int64
}

// NewConsumerGroup creates a new consumer group for a topic.
func NewConsumerGroup(name string, topic *Topic) *ConsumerGroup {
	return &ConsumerGroup{
		name:       name,
		topic:      topic,
		partitions: make(map[int]*PartitionConsumer),
	}
}

// Name returns group name.
func (cg *ConsumerGroup) Name() string { return cg.name }

// Consume fetches up to batch messages from a partition, starting from the
// committed offset for this group. Returns (messages, nextOffset, error).
func (cg *ConsumerGroup) Consume(partition int, batch int) ([]Message, int64, error) {
	if batch <= 0 {
		return nil, -1, errors.New("batch must be > 0")
	}
	if partition < 0 || partition >= cg.topic.Partitions() {
		return nil, -1, errors.New("partition out of range")
	}
	cg.mu.RLock()
	pc, ok := cg.partitions[partition]
	cg.mu.RUnlock()
	if !ok {
		pc = &PartitionConsumer{partition: partition, group: cg.name, offset: 0}
		cg.mu.Lock()
		cg.partitions[partition] = pc
		cg.mu.Unlock()
	}
	part := cg.topic.GetPartition(partition)
	if part == nil {
		return nil, -1, errors.New("partition not found")
	}
	msgs, err := part.ReadFrom(pc.offset, batch)
	if err != nil {
		return nil, -1, err
	}
	// next offset to commit is pc.offset + len(msgs)
	nextOff := pc.offset + int64(len(msgs))
	return msgs, nextOff, nil
}

// Commit records the offset for a partition in this group.
func (cg *ConsumerGroup) Commit(partition int, offset int64) error {
	if partition < 0 || partition >= cg.topic.Partitions() {
		return errors.New("partition out of range")
	}
	cg.mu.Lock()
	pc, ok := cg.partitions[partition]
	cg.mu.Unlock()
	if !ok {
		pc = &PartitionConsumer{partition: partition, group: cg.name, offset: offset}
		cg.mu.Lock()
		cg.partitions[partition] = pc
		cg.mu.Unlock()
		return nil
	}
	cg.mu.Lock()
	pc.offset = offset
	cg.mu.Unlock()
	return nil
}

// Offset returns the committed offset for a partition (0 if not set).
func (cg *ConsumerGroup) Offset(partition int) int64 {
	cg.mu.RLock()
	defer cg.mu.RUnlock()
	if pc, ok := cg.partitions[partition]; ok {
		return pc.offset
	}
	return 0
}
