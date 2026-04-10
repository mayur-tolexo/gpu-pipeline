package mq

import (
	"fmt"
	"sort"
)

// Topic groups partitions with consistent hashing for deterministic message routing.
// The consistent hash ring allows for zero-copy message movement during partition rebalancing.
// All topic logic is encapsulated here; external API access is via Queue.
type Topic struct {
	name        string
	partitions  []*Partition
	replicas    int          // number of replicas in the consistent hash ring per partition
	ringHashes  []uint32     // sorted list of ring hashes
	hashMap     map[uint32]int // maps hash value to partition index
	partitioner func(string) int // can be replaced for testing
}

// NewTopicWithReplicas creates a topic with consistent hashing.
// replicas controls how many times each partition appears on the hash ring (default 3).
// Replicas > 1 improves distribution but uses more memory.
func NewTopicWithReplicas(name string, numPartitions, partitionCapacity, replicas int) *Topic {
	if numPartitions <= 0 {
		numPartitions = 1
	}
	if replicas <= 0 {
		replicas = 3
	}
	
	t := &Topic{
		name:       name,
		partitions: make([]*Partition, numPartitions),
		replicas:   replicas,
		hashMap:    make(map[uint32]int),
	}
	
	// Create partitions
	for i := 0; i < numPartitions; i++ {
		t.partitions[i] = NewPartition(fmt.Sprintf("%s-%d", name, i), partitionCapacity)
	}
	
	// Build consistent hash ring:
	// For each partition, add 'replicas' entries to the ring to improve distribution.
	// This reduces the impact of adding/removing a partition on existing message distribution.
	for partIdx := 0; partIdx < numPartitions; partIdx++ {
		for replicaNum := 0; replicaNum < replicas; replicaNum++ {
			// Create unique hash for each replica
			replicaKey := fmt.Sprintf("%s-%d-%d", name, partIdx, replicaNum)
			hashVal := fnv32(replicaKey)
			t.ringHashes = append(t.ringHashes, hashVal)
			t.hashMap[hashVal] = partIdx
		}
	}
	// Sort ring for binary search during lookup
	sort.Slice(t.ringHashes, func(i, j int) bool {
		return t.ringHashes[i] < t.ringHashes[j]
	})
	
	// Consistent hash partitioner:
	// Hash the message key, find the first ring hash >= key hash.
	// If no match, wrap around to hash ring[0].
	// This ensures the same key always maps to the same partition.
	t.partitioner = func(key string) int {
		if key == "" {
			return 0 // empty keys always go to partition 0
		}
		keyHash := fnv32(key)
		// Binary search for the first hash >= keyHash
		idx := sort.Search(len(t.ringHashes), func(i int) bool {
			return t.ringHashes[i] >= keyHash
		})
		if idx == len(t.ringHashes) {
			idx = 0 // wrap around to start of ring
		}
		return t.hashMap[t.ringHashes[idx]]
	}
	
	return t
}

// NewTopic creates a topic with default 3 replicas on the consistent hash ring.
func NewTopic(name string, numPartitions int, partitionCapacity int) *Topic {
	return NewTopicWithReplicas(name, numPartitions, partitionCapacity, 3)
}

func (t *Topic) Name() string { return t.name }
func (t *Topic) Partitions() int { return len(t.partitions) }
func (t *Topic) Replicas() int { return t.replicas }

// Produce appends message to the determined partition.
// Returns partition index, message offset, and error.
func (t *Topic) Produce(msg Message) (int, int64, error) {
	if msg.Key == "" {
		return -1, -1, ErrInvalidArg
	}
	idx := t.partitioner(msg.Key)
	if idx < 0 || idx >= len(t.partitions) {
		return -1, -1, ErrPartitionRange
	}
	off, err := t.partitions[idx].Append(msg)
	if err != nil {
		return idx, -1, err
	}
	return idx, off, nil
}

func (t *Topic) GetPartition(i int) *Partition {
	if i < 0 || i >= len(t.partitions) {
		return nil
	}
	return t.partitions[i]
}

// SetPartitioner allows custom partitioning logic (primarily for testing).
func (t *Topic) SetPartitioner(fn func(string) int) {
	if fn != nil {
		t.partitioner = fn
	}
}
