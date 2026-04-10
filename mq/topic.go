package mq

import (
	"fmt"
	"hash/fnv"
	"sort"

	internalmq "gpu-pipeline/mq/internal"
)

// Topic represents a logical stream divided into partitions.
// It's intentionally lightweight so it can be extended (persistence, replication, etc.).
type Topic struct {
	name        string
	partitions  []*internalmq.Partition
	partitioner func(key string) int
	// consistent hashing ring
	ringHashes []uint32
	hashMap    map[uint32]int
}

const defaultReplicas = 3

// NewTopic creates a topic with the requested number of partitions.
func NewTopic(name string, partitions int) *Topic {
	if partitions <= 0 {
		partitions = 1
	}
	t := &Topic{
		name:       name,
		partitions: make([]*internalmq.Partition, partitions),
		hashMap:    make(map[uint32]int),
	}
	for i := 0; i < partitions; i++ {
		t.partitions[i] = internalmq.NewPartition(fmt.Sprintf("%s-%d", name, i))
	}
	// build consistent hashing ring with replicas
	for i := 0; i < partitions; i++ {
		for r := 0; r < defaultReplicas; r++ {
			key := fmt.Sprintf("%s-%d-%d", name, i, r)
			h := fnv.New32a()
			h.Write([]byte(key))
			hv := h.Sum32()
			t.ringHashes = append(t.ringHashes, hv)
			t.hashMap[hv] = i
		}
	}
	sort.Slice(t.ringHashes, func(i, j int) bool { return t.ringHashes[i] < t.ringHashes[j] })

	// default partitioner: consistent hashing
	t.partitioner = func(key string) int {
		// if user didn't provide a key, use partition 0 to keep deterministic behavior
		if key == "" {
			return 0
		}
		h := fnv.New32a()
		h.Write([]byte(key))
		hv := h.Sum32()
		// find the first hash >= hv
		idx := sort.Search(len(t.ringHashes), func(i int) bool { return t.ringHashes[i] >= hv })
		if idx == len(t.ringHashes) {
			idx = 0
		}
		hash := t.ringHashes[idx]
		return t.hashMap[hash]
	}
	return t
}

// Name returns the topic name.
func (t *Topic) Name() string { return t.name }

// Partitions returns the number of partitions.
func (t *Topic) Partitions() int { return len(t.partitions) }

// Produce appends a message to a partition determined by the message key.
// Returns the partition index and the appended offset.
func (t *Topic) Produce(msg internalmq.Message) (int, int64) {
	idx := t.partitioner(msg.Key)
	off := t.partitions[idx].Append(msg)
	return idx, off
}

// GetPartition returns the partition by index or nil if out of range.
func (t *Topic) GetPartition(idx int) *internalmq.Partition {
	if idx < 0 || idx >= len(t.partitions) {
		return nil
	}
	return t.partitions[idx]
}

// SetPartitioner allows replacing the partitioning function (for testing or custom strategies).
func (t *Topic) SetPartitioner(fn func(key string) int) {
	if fn == nil {
		return
	}
	t.partitioner = fn
}
