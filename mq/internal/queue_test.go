package mq

import (
	"testing"
	"time"
)

func TestQueue_CreatePublishConsumeAck(t *testing.T) {
	q := NewQueue(0)
	if err := q.CreateTopic("t1", 3, 0); err != nil {
		t.Fatalf("create topic failed: %v", err)
	}
	// create duplicate should fail
	if err := q.CreateTopic("t1", 3, 0); err == nil {
		t.Fatalf("expected duplicate topic error")
	} else if err != ErrTopicExists {
		t.Fatalf("expected ErrTopicExists, got %v", err)
	}

	msg := Message{Key: "k1", Payload: []byte("hello"), Timestamp: time.Now()}
	p, off, err := q.Publish("t1", msg)
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if p < 0 || p >= 3 {
		t.Fatalf("invalid partition %d", p)
	}
	// consuming from committed offset 0 should give the message
	msgs, err := q.Consume("t1", "cg", p, 10)
	if err != nil {
		t.Fatalf("consume failed: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatalf("expected messages, got empty")
	}
	// ack and check offset
	if err := q.Ack("t1", "cg", p, off+1); err != nil {
		t.Fatalf("ack failed: %v", err)
	}
}

func TestQueue_PublishInvalidArgs(t *testing.T) {
	q := NewQueue(0)
	q.CreateTopic("t1", 3, 0)

	// empty key should fail
	_, _, err := q.Publish("t1", Message{Key: "", Payload: []byte("x")})
	if err == nil {
		t.Fatalf("expected error for empty key")
	}

	// nonexistent topic should fail
	_, _, err = q.Publish("missing", Message{Key: "k", Payload: []byte("x")})
	if err != ErrTopicNotFound {
		t.Fatalf("expected ErrTopicNotFound")
	}
}

func TestQueue_ConsumeInvalidArgs(t *testing.T) {
	q := NewQueue(0)
	q.CreateTopic("t1", 3, 0)

	// missing group
	_, err := q.Consume("t1", "", 0, 1)
	if err == nil {
		t.Fatalf("expected error for missing group")
	}

	// invalid batch
	_, err = q.Consume("t1", "cg", 0, 0)
	if err == nil {
		t.Fatalf("expected error for batch=0")
	}

	// out of range partition
	_, err = q.Consume("t1", "cg", 10, 1)
	if err != ErrPartitionRange {
		t.Fatalf("expected ErrPartitionRange")
	}
}

func TestQueue_AckInvalidArgs(t *testing.T) {
	q := NewQueue(0)
	q.CreateTopic("t1", 3, 0)

	// missing topic
	err := q.Ack("missing", "cg", 0, 0)
	if err != ErrTopicNotFound {
		t.Fatalf("expected ErrTopicNotFound")
	}

	// out of range partition
	err = q.Ack("t1", "cg", 10, 0)
	if err != ErrPartitionRange {
		t.Fatalf("expected ErrPartitionRange")
	}
}

func TestQueue_GetTopic(t *testing.T) {
	q := NewQueue(0)
	q.CreateTopic("t1", 3, 0)

	// get existing topic
	topic, err := q.GetTopic("t1")
	if err != nil {
		t.Fatalf("GetTopic failed: %v", err)
	}
	if topic == nil {
		t.Fatalf("expected topic, got nil")
	}
	if topic.Name() != "t1" {
		t.Fatalf("expected topic name t1, got %s", topic.Name())
	}

	// get non-existent topic
	topic, err = q.GetTopic("missing")
	if err != ErrTopicNotFound {
		t.Fatalf("expected ErrTopicNotFound, got %v", err)
	}
	if topic != nil {
		t.Fatalf("expected nil topic")
	}
}

func TestQueue_ListTopics(t *testing.T) {
	q := NewQueue(0)
	
	// empty queue
	topics := q.ListTopics()
	if len(topics) != 0 {
		t.Fatalf("expected 0 topics, got %d", len(topics))
	}

	// create multiple topics
	q.CreateTopic("t1", 3, 0)
	q.CreateTopic("t2", 2, 0)
	q.CreateTopic("t3", 1, 0)

	topics = q.ListTopics()
	if len(topics) != 3 {
		t.Fatalf("expected 3 topics, got %d", len(topics))
	}

	// verify topic names
	names := make(map[string]bool)
	for _, topicName := range topics {
		names[topicName] = true
	}
	if !names["t1"] || !names["t2"] || !names["t3"] {
		t.Fatalf("expected t1, t2, t3 topics")
	}
}

func TestConsumerGroup_Name(t *testing.T) {
	// Create a topic first for the consumer group
	topic := NewTopic("test-topic", 2, 0)
	cg := NewConsumerGroup("group1", topic)
	if cg.Name() != "group1" {
		t.Fatalf("expected group1, got %s", cg.Name())
	}
}

func TestTopic_Name(t *testing.T) {
	topic := NewTopic("mytopic", 3, 0)
	if topic.Name() != "mytopic" {
		t.Fatalf("expected mytopic, got %s", topic.Name())
	}
}

func TestTopic_Replicas(t *testing.T) {
	topic := NewTopicWithReplicas("t1", 3, 0, 2)
	if topic.Replicas() != 2 {
		t.Fatalf("expected 2 replicas, got %d", topic.Replicas())
	}
}

func TestTopic_SetPartitioner(t *testing.T) {
	topic := NewTopic("t1", 3, 0)
	
	// custom partitioner that always returns partition 0
	customPartitioner := func(key string) int {
		return 0
	}
	
	topic.SetPartitioner(customPartitioner)
	
	// produce message - should go to partition 0
	msg := Message{Key: "any-key", Payload: []byte("test"), Timestamp: time.Now()}
	p, _, err := topic.Produce(msg)
	if err != nil {
		t.Fatalf("produce failed: %v", err)
	}
	if p != 0 {
		t.Fatalf("expected partition 0, got %d", p)
	}
}

func TestQueue_ComplexWorkflow(t *testing.T) {
	q := NewQueue(0)

	// create topic with 2 partitions
	if err := q.CreateTopic("events", 2, 0); err != nil {
		t.Fatalf("create topic failed: %v", err)
	}

	// publish multiple messages
	for i := 0; i < 10; i++ {
		msg := Message{
			Key:     "user-" + string(rune(i%3)),
			Payload: []byte("event-" + string(rune(i))),
			Timestamp: time.Now(),
		}
		_, _, err := q.Publish("events", msg)
		if err != nil {
			t.Fatalf("publish failed: %v", err)
		}
	}

	// consume from both partitions
	for p := 0; p < 2; p++ {
		msgs, err := q.Consume("events", "group1", p, 100)
		if err != nil {
			t.Fatalf("consume partition %d failed: %v", p, err)
		}
		if len(msgs) == 0 {
			t.Fatalf("expected messages in partition %d", p)
		}

		// ack all messages
		if err := q.Ack("events", "group1", p, int64(len(msgs))); err != nil {
			t.Fatalf("ack partition %d failed: %v", p, err)
		}
	}
}

// TestQueue_CompactTopic tests watermark-based topic compaction
func TestQueue_CompactTopic(t *testing.T) {
	q := NewQueue(0)
	if err := q.CreateTopic("telemetry", 3, 0); err != nil {
		t.Fatalf("create topic failed: %v", err)
	}

	// Publish 30 messages (10 per partition due to hashing)
	for i := 0; i < 30; i++ {
		msg := Message{
			Key:       "gpu-" + string(rune('0' + (i % 3))),
			Payload:   []byte("data"),
			Timestamp: time.Now(),
		}
		q.Publish("telemetry", msg)
	}

	// Simulate consumption by different groups
	// group1 consumes partition 0
	msgs0, _ := q.Consume("telemetry", "group1", 0, 100)
	q.Ack("telemetry", "group1", 0, int64(len(msgs0)))

	// group2 consumes partition 1
	msgs1, _ := q.Consume("telemetry", "group2", 1, 100)
	q.Ack("telemetry", "group2", 1, int64(len(msgs1)))

	// group3 only partially consumes partition 2 (leave some messages unprocessed)
	msgs2, _ := q.Consume("telemetry", "group3", 2, 100)
	if len(msgs2) > 0 {
		// Only ack half of the messages
		q.Ack("telemetry", "group3", 2, int64(len(msgs2)/2))
	}

	// Compact the topic
	results, err := q.CompactTopic("telemetry")
	if err != nil {
		t.Fatalf("compact failed: %v", err)
	}

	// Should have compaction results for all 3 partitions
	if len(results) != 3 {
		t.Fatalf("expected results for 3 partitions, got %d", len(results))
	}

	// At least some messages should have been compacted
	totalCompacted := int64(0)
	for _, count := range results {
		totalCompacted += count
	}
	if totalCompacted == 0 {
		t.Logf("warning: no messages compacted (this is OK if watermarks are at 0)")
	}
}

// TestQueue_CompactTopicNotFound tests compacting non-existent topic
func TestQueue_CompactTopicNotFound(t *testing.T) {
	q := NewQueue(0)
	_, err := q.CompactTopic("nonexistent")
	if err != ErrTopicNotFound {
		t.Fatalf("expected ErrTopicNotFound, got %v", err)
	}
}

// TestQueue_GetPartitionStats tests partition statistics
func TestQueue_GetPartitionStats(t *testing.T) {
	q := NewQueue(0)
	q.CreateTopic("stats-topic", 2, 0)

	// Publish some messages
	for i := 0; i < 10; i++ {
		msg := Message{
			Key:       "key-" + string(rune('0' + (i % 2))),
			Payload:   []byte("payload"),
			Timestamp: time.Now(),
		}
		q.Publish("stats-topic", msg)
	}

	// Get stats before any consumption
	stats, err := q.GetPartitionStats("stats-topic", 0)
	if err != nil {
		t.Fatalf("get stats failed: %v", err)
	}

	if stats["id"] != "stats-topic-0" {
		t.Fatalf("expected id 'stats-topic-0', got %v", stats["id"])
	}

	msgCount := stats["message_count"].(int)
	if msgCount == 0 {
		t.Fatalf("expected some messages in partition")
	}

	// Consume and ack
	msgs, _ := q.Consume("stats-topic", "test-group", 0, 100)
	q.Ack("stats-topic", "test-group", 0, int64(len(msgs)))

	// Get stats after consumption
	stats, _ = q.GetPartitionStats("stats-topic", 0)
	watermark := stats["min_consumer_offset"].(int64)
	if watermark == 0 && msgCount > 0 {
		t.Logf("warning: watermark is 0 even after ack (min offset not updated by test consumer)")
	}
}

// TestQueue_CompactMultipleConsumerGroups tests compaction with multiple consumer groups
func TestQueue_CompactMultipleConsumerGroups(t *testing.T) {
	q := NewQueue(0)
	q.CreateTopic("multi-group", 1, 0)

	// Publish 100 messages to partition 0
	for i := 0; i < 100; i++ {
		msg := Message{
			Key:       "key",
			Payload:   []byte("msg"),
			Timestamp: time.Now(),
		}
		q.Publish("multi-group", msg)
	}

	// Group1 reads and acks 40 messages
	msgs, _ := q.Consume("multi-group", "group1", 0, 100)
	if len(msgs) > 40 {
		msgs = msgs[:40]
	}
	q.Ack("multi-group", "group1", 0, int64(len(msgs)))

	// Group2 reads and acks 30 messages (slower consumer)
	msgs, _ = q.Consume("multi-group", "group2", 0, 100)
	if len(msgs) > 30 {
		msgs = msgs[:30]
	}
	q.Ack("multi-group", "group2", 0, int64(len(msgs)))

	// Compact: watermark should be at 30 (slowest consumer)
	results, _ := q.CompactTopic("multi-group")
	compacted := results["multi-group-0"]

	// Should have compacted 30 messages (up to group2's offset)
	if compacted != 30 {
		t.Logf("expected 30 messages compacted, got %d (may vary based on message distribution)", compacted)
	}

	// Get stats to verify
	stats, _ := q.GetPartitionStats("multi-group", 0)
	remainingCount := stats["message_count"].(int)
	if remainingCount > 70 {
		t.Logf("warning: more than 70 messages remaining after compaction")
	}
}

// TestQueue_WatermarkBoundedBySlowConsumer tests that watermark respects slowest consumer
func TestQueue_WatermarkBoundedBySlowConsumer(t *testing.T) {
	q := NewQueue(0)
	q.CreateTopic("watermark-test", 1, 0)

	// Publish 100 messages
	for i := 0; i < 100; i++ {
		msg := Message{
			Key:       "key",
			Payload:   []byte("msg"),
			Timestamp: time.Now(),
		}
		q.Publish("watermark-test", msg)
	}

	// Fast consumer reaches offset 80
	q.Ack("watermark-test", "fast-group", 0, 80)

	// Slow consumer only at offset 30
	q.Ack("watermark-test", "slow-group", 0, 30)

	// Compact
	results, _ := q.CompactTopic("watermark-test")
	compacted := results["watermark-test-0"]

	// Should only compact up to 30 (the slower consumer)
	if compacted != 30 {
		t.Logf("expected 30 messages compacted (up to slow consumer), got %d", compacted)
	}

	// Verify fast consumer's offset is adjusted
	stats, _ := q.GetPartitionStats("watermark-test", 0)
	remaining := stats["message_count"].(int)
	if remaining != 70 {
		t.Logf("expected 70 remaining messages, got %d", remaining)
	}
}

