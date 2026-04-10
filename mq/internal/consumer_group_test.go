package mq

import (
	"testing"
	"time"
)

func setupTopic() *Topic {
	topic := NewTopic("test-topic", 2, 1000)
	return topic
}

func TestConsumerGroup_BasicFlow(t *testing.T) {
	topic := setupTopic()
	cg := NewConsumerGroup("group-1", topic)

	msg := Message{
		Key:       "key",
		Payload:   []byte("data"),
		Timestamp: time.Now(),
	}

	topic.GetPartition(0).Append(msg)

	msgs, nextOff, err := cg.Consume(0, 10)
	if err != nil {
		t.Fatalf("consume failed: %v", err)
	}

	if len(msgs) != 1 {
		t.Fatalf("expected 1 msg, got %d", len(msgs))
	}

	if nextOff != 1 {
		t.Fatalf("expected nextOff 1, got %d", nextOff)
	}
}

func TestConsumerGroup_BatchLimit(t *testing.T) {
	topic := setupTopic()
	cg := NewConsumerGroup("group-1", topic)

	for i := 0; i < 5; i++ {
		topic.GetPartition(0).Append(Message{
			Key:     "k",
			Payload: []byte("d"),
		})
	}

	msgs, nextOff, _ := cg.Consume(0, 3)

	if len(msgs) != 3 {
		t.Fatalf("expected 3 msgs, got %d", len(msgs))
	}

	if nextOff != 3 {
		t.Fatalf("expected nextOff 3, got %d", nextOff)
	}
}

func TestConsumerGroup_EmptyPartition(t *testing.T) {
	topic := setupTopic()
	cg := NewConsumerGroup("group-1", topic)

	msgs, nextOff, err := cg.Consume(0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(msgs) != 0 {
		t.Fatalf("expected empty msgs")
	}

	if nextOff != 0 {
		t.Fatalf("expected offset 0")
	}
}

func TestConsumerGroup_InvalidBatch(t *testing.T) {
	topic := setupTopic()
	cg := NewConsumerGroup("group-1", topic)

	_, _, err := cg.Consume(0, 0)
	if err == nil {
		t.Fatalf("expected error for batch=0")
	}
}

func TestConsumerGroup_InvalidPartition(t *testing.T) {
	topic := setupTopic()
	cg := NewConsumerGroup("group-1", topic)

	_, _, err := cg.Consume(10, 10)
	if err == nil {
		t.Fatalf("expected partition error")
	}
}

func TestConsumerGroup_CommitAndOffset(t *testing.T) {
	topic := setupTopic()
	cg := NewConsumerGroup("group-1", topic)

	topic.GetPartition(0).Append(Message{})

	_, nextOff, _ := cg.Consume(0, 10)

	err := cg.Commit(0, nextOff)
	if err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	offset := cg.Offset(0)
	if offset != nextOff {
		t.Fatalf("expected offset %d, got %d", nextOff, offset)
	}
}

func TestConsumerGroup_MultiplePartitions(t *testing.T) {
	topic := setupTopic()
	cg := NewConsumerGroup("group-1", topic)

	topic.GetPartition(0).Append(Message{})
	topic.GetPartition(1).Append(Message{})

	_, off0, _ := cg.Consume(0, 10)
	_, off1, _ := cg.Consume(1, 10)

	cg.Commit(0, off0)
	cg.Commit(1, off1)

	if cg.Offset(0) != off0 {
		t.Fatalf("partition 0 offset mismatch")
	}

	if cg.Offset(1) != off1 {
		t.Fatalf("partition 1 offset mismatch")
	}
}

func TestConsumerGroup_BeyondAvailable(t *testing.T) {
	topic := setupTopic()
	cg := NewConsumerGroup("group-1", topic)

	topic.GetPartition(0).Append(Message{})

	// First consume
	msgs, nextOff, _ := cg.Consume(0, 10)

	if len(msgs) != 1 {
		t.Fatalf("expected 1 msg")
	}

	if nextOff != 1 {
		t.Fatalf("expected offset 1")
	}

	// Commit offset
	cg.Commit(0, nextOff)

	// Now consume again
	msgs, nextOff, _ = cg.Consume(0, 10)

	if len(msgs) != 0 {
		t.Fatalf("expected no msgs after commit")
	}

	if nextOff != 1 {
		t.Fatalf("offset should remain 1")
	}
}
