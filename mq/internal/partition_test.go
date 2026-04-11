package mq

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestPartition_BasicOperations(t *testing.T) {
	p := NewPartition("p0", 0)
	if p.ID() != "p0" {
		t.Fatalf("unexpected id: %s", p.ID())
	}
	if p.Len() != 0 {
		t.Fatalf("expected empty partition")
	}
	if p.TailOffset() != 0 {
		t.Fatalf("expected tail 0")
	}

	m1 := Message{Key: "k", Payload: []byte("a"), Timestamp: time.Now()}
	off1, err := p.Append(m1)
	if err != nil {
		t.Fatalf("unexpected append error: %v", err)
	}
	if off1 != 0 {
		t.Fatalf("expected offset 0, got %d", off1)
	}
	if p.Len() != 1 {
		t.Fatalf("expected len 1")
	}

	m, err := p.Get(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(m.Payload) != "a" {
		t.Fatalf("unexpected payload: %s", string(m.Payload))
	}

	// out of range
	if _, err := p.Get(1); err == nil {
		t.Fatalf("expected error for Get out of range")
	}
}

func TestPartition_ReadFrom(t *testing.T) {
	p := NewPartition("p1", 0)
	for i := 0; i < 5; i++ {
		if _, err := p.Append(Message{Payload: []byte{byte(i)}}); err != nil {
			t.Fatalf("append failed: %v", err)
		}
	}

	// read subset
	msgs, err := p.ReadFrom(1, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}

	// read all from offset
	all, err := p.ReadFrom(0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(all))
	}

	// offset == tail
	empty, err := p.ReadFrom(p.TailOffset(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty slice for offset==tail")
	}

	// offset > tail
	if _, err := p.ReadFrom(p.TailOffset()+1, 1); err == nil {
		t.Fatalf("expected error for offset > tail")
	}
}

func TestPartition_CommitAndOffset(t *testing.T) {
	p := NewPartition("p2", 0)
	for i := 0; i < 3; i++ {
		if _, err := p.Append(Message{Payload: []byte(fmt.Sprintf("%c", rune('a')+rune(i)))}); err != nil {
			t.Fatalf("append failed: %v", err)
		}
	}
	// default offset
	if p.Offset("cg") != 0 {
		t.Fatalf("expected default offset 0")
	}
	// valid commit
	if err := p.Commit("cg", 2); err != nil {
		t.Fatalf("unexpected commit error: %v", err)
	}
	if p.Offset("cg") != 2 {
		t.Fatalf("expected committed offset 2, got %d", p.Offset("cg"))
	}
	// commit beyond tail should fail
	if err := p.Commit("cg", p.TailOffset()+1); err == nil {
		t.Fatalf("expected error when committing beyond tail")
	}
}

func TestPartition_Concurrency(t *testing.T) {
	p := NewPartition("p3", 0)
	wg := sync.WaitGroup{}
	n := 200
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			if _, err := p.Append(Message{Payload: []byte(fmt.Sprintf("%c", rune('A')+rune(i%26)))}); err != nil {
				t.Fatalf("append failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	if p.Len() != n {
		t.Fatalf("expected len %d, got %d", n, p.Len())
	}

	// concurrent reads and commits
	wg = sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = p.ReadFrom(0, 10)
	}()
	go func() {
		defer wg.Done()
		_ = p.Commit("cg2", p.TailOffset())
	}()
	wg.Wait()

	if p.Offset("cg2") != int64(n) {
		t.Fatalf("expected committed offset %d, got %d", n, p.Offset("cg2"))
	}
}

func TestPartition_Errors(t *testing.T) {
	p := NewPartition("p4", 0)
	// negative get
	if _, err := p.Get(-1); err == nil {
		t.Fatalf("expected error for negative offset")
	}
	// negative read
	if _, err := p.ReadFrom(-1, 1); err == nil {
		t.Fatalf("expected error for negative read offset")
	}
	// negative commit
	if err := p.Commit("c", -1); err == nil {
		t.Fatalf("expected error for negative commit")
	}
}

// TestPartition_WatermarkBasedCleanup tests the watermark-based compaction strategy
func TestPartition_GetMinConsumerOffset(t *testing.T) {
	p := NewPartition("p-watermark", 0)

	// Add 10 messages
	for i := 0; i < 10; i++ {
		p.Append(Message{Key: "k" + string(rune('0'+i)), Payload: []byte("payload"), Timestamp: time.Now()})
	}

	// No commits yet - min offset should be 0
	minOff := p.GetMinConsumerOffset()
	if minOff != 0 {
		t.Fatalf("expected min offset 0 with no commits, got %d", minOff)
	}

	// Commit for group1 at offset 3
	p.Commit("group1", 3)
	minOff = p.GetMinConsumerOffset()
	if minOff != 3 {
		t.Fatalf("expected min offset 3 after group1 commit, got %d", minOff)
	}

	// Commit for group2 at offset 5
	p.Commit("group2", 5)
	minOff = p.GetMinConsumerOffset()
	if minOff != 3 {
		t.Fatalf("expected min offset 3 (group1 is slower), got %d", minOff)
	}

	// Group1 catches up to offset 7
	p.Commit("group1", 7)
	minOff = p.GetMinConsumerOffset()
	if minOff != 5 {
		t.Fatalf("expected min offset 5 (group2 is now slower), got %d", minOff)
	}

	// Both groups at offset 9
	p.Commit("group1", 9)
	p.Commit("group2", 9)
	minOff = p.GetMinConsumerOffset()
	if minOff != 9 {
		t.Fatalf("expected min offset 9 (both at 9), got %d", minOff)
	}
}

// TestPartition_Compact tests watermark-based compaction
func TestPartition_Compact(t *testing.T) {
	p := NewPartition("p-compact", 0)

	// Add 10 messages (offsets 0-9)
	for i := 0; i < 10; i++ {
		p.Append(Message{Key: "k" + string(rune('0'+i)), Payload: []byte("msg"+string(rune('0'+i))), Timestamp: time.Now()})
	}

	// Both groups consume but don't commit yet
	// Group1 reads offsets 0-4, Group2 reads offsets 0-6
	p.Commit("group1", 5)
	p.Commit("group2", 7)

	// Compact: should remove messages 0-4 (before watermark of 5)
	compacted, err := p.Compact()
	if err != nil {
		t.Fatalf("unexpected compact error: %v", err)
	}
	if compacted != 5 {
		t.Fatalf("expected 5 messages compacted, got %d", compacted)
	}

	// After compact: partition should have 5 messages (original 5-9, now at indices 0-4)
	if p.Len() != 5 {
		t.Fatalf("expected 5 messages after compact, got %d", p.Len())
	}

	// Tail offset should be 5 now
	if p.TailOffset() != 5 {
		t.Fatalf("expected tail offset 5 after compact, got %d", p.TailOffset())
	}

	// Consumer offsets should be adjusted
	group1Off := p.Offset("group1")
	if group1Off != 0 {
		t.Fatalf("expected group1 offset adjusted to 0, got %d", group1Off)
	}

	group2Off := p.Offset("group2")
	if group2Off != 2 {
		t.Fatalf("expected group2 offset adjusted to 2, got %d", group2Off)
	}

	// Verify message content is correct after compact
	msg, err := p.Get(0)
	if err != nil {
		t.Fatalf("unexpected error getting message at offset 0: %v", err)
	}
	if string(msg.Payload) != "msg5" {
		t.Fatalf("expected payload 'msg5' at offset 0 after compact, got '%s'", string(msg.Payload))
	}

	msg, err = p.Get(4)
	if err != nil {
		t.Fatalf("unexpected error getting message at offset 4: %v", err)
	}
	if string(msg.Payload) != "msg9" {
		t.Fatalf("expected payload 'msg9' at offset 4 after compact, got '%s'", string(msg.Payload))
	}
}

// TestPartition_CompactNoConsumers tests that compact returns 0 when no consumers
func TestPartition_CompactNoConsumers(t *testing.T) {
	p := NewPartition("p-no-consumers", 0)

	// Add 10 messages
	for i := 0; i < 10; i++ {
		p.Append(Message{Key: "k" + string(rune('0'+i)), Payload: []byte("msg"), Timestamp: time.Now()})
	}

	// Compact with no consumer commits - should return 0
	compacted, err := p.Compact()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compacted != 0 {
		t.Fatalf("expected 0 compacted with no consumer commits, got %d", compacted)
	}

	// Partition size should be unchanged
	if p.Len() != 10 {
		t.Fatalf("expected 10 messages still in partition, got %d", p.Len())
	}
}

// TestPartition_CompactMultipleGroups tests compaction with many consumer groups
func TestPartition_CompactMultipleGroups(t *testing.T) {
	p := NewPartition("p-multi-group", 0)

	// Add 100 messages
	for i := 0; i < 100; i++ {
		p.Append(Message{Key: "k" + fmt.Sprintf("%03d", i), Payload: []byte("msg"), Timestamp: time.Now()})
	}

	// 5 consumer groups at different offsets
	p.Commit("group1", 50)
	p.Commit("group2", 75)
	p.Commit("group3", 25)
	p.Commit("group4", 60)
	p.Commit("group5", 40)

	// Watermark should be 25 (slowest consumer: group3)
	minOff := p.GetMinConsumerOffset()
	if minOff != 25 {
		t.Fatalf("expected watermark 25, got %d", minOff)
	}

	// Compact
	compacted, err := p.Compact()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compacted != 25 {
		t.Fatalf("expected 25 messages compacted, got %d", compacted)
	}

	// All offsets should be adjusted
	if p.Offset("group1") != 25 {
		t.Fatalf("expected group1 offset 25, got %d", p.Offset("group1"))
	}
	if p.Offset("group3") != 0 {
		t.Fatalf("expected group3 offset 0, got %d", p.Offset("group3"))
	}

	// Remaining messages
	if p.Len() != 75 {
		t.Fatalf("expected 75 messages after compact, got %d", p.Len())
	}
}

// TestPartition_CompactAfterMultipleRounds tests repeated compaction
func TestPartition_CompactAfterMultipleRounds(t *testing.T) {
	p := NewPartition("p-multi-compact", 0)

	// Add 100 messages
	for i := 0; i < 100; i++ {
		p.Append(Message{Key: "k" + fmt.Sprintf("%03d", i), Payload: []byte("msg"), Timestamp: time.Now()})
	}

	// Round 1: Groups at different stages
	p.Commit("group1", 30)
	p.Commit("group2", 40)

	compacted, _ := p.Compact()
	if compacted != 30 {
		t.Fatalf("round 1: expected 30 compacted, got %d", compacted)
	}

	// After round 1 compact:
	// - Partition has 70 messages (original 30-99)
	// - group1 offset: 30-30=0
	// - group2 offset: 40-30=10

	// Round 2: Groups progress (using adjusted offsets)
	// group1 progresses to offset 20 (i.e., 20 more messages consumed)
	p.Commit("group1", 20)
	// group2 progresses to offset 30
	p.Commit("group2", 30)

	compacted, _ = p.Compact()
	if compacted != 20 {
		t.Fatalf("round 2: expected 20 compacted, got %d", compacted)
	}

	// After round 2 compact:
	// - Partition has 50 messages (original 50-99)
	// - group1 offset: 20-20=0
	// - group2 offset: 30-20=10

	// Final state check
	if p.Len() != 50 {
		t.Fatalf("expected 50 messages total, got %d", p.Len())
	}
	if p.Offset("group1") != 0 {
		t.Fatalf("expected group1 offset 0, got %d", p.Offset("group1"))
	}
	if p.Offset("group2") != 10 {
		t.Fatalf("expected group2 offset 10, got %d", p.Offset("group2"))
	}
}

// TestPartition_CompactEmptyPartition tests compacting empty partition
func TestPartition_CompactEmptyPartition(t *testing.T) {
	p := NewPartition("p-empty", 0)

	// Compact empty partition
	compacted, err := p.Compact()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compacted != 0 {
		t.Fatalf("expected 0 compacted on empty partition, got %d", compacted)
	}
}

// TestPartition_CompactConcurrent tests thread-safe compaction with concurrent reads
func TestPartition_CompactConcurrent(t *testing.T) {
	p := NewPartition("p-concurrent", 0)

	// Add 1000 messages
	for i := 0; i < 1000; i++ {
		p.Append(Message{Key: "k" + fmt.Sprintf("%04d", i), Payload: []byte("msg"), Timestamp: time.Now()})
	}

	p.Commit("group1", 500)
	p.Commit("group2", 500)

	var wg sync.WaitGroup
	errors := make([]error, 0)
	var mu sync.Mutex

	// Start 10 goroutines reading
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(offset int64) {
			defer wg.Done()
			msgs, err := p.ReadFrom(offset, 10)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
			if len(msgs) > 10 {
				mu.Lock()
				errors = append(errors, fmt.Errorf("expected max 10 messages, got %d", len(msgs)))
				mu.Unlock()
			}
		}(int64(i * 50))
	}

	// Compact in main goroutine
	compacted, _ := p.Compact()
	if compacted != 500 {
		t.Fatalf("expected 500 compacted, got %d", compacted)
	}

	wg.Wait()

	if len(errors) > 0 {
		t.Fatalf("concurrent reads failed: %v", errors[0])
	}
}

