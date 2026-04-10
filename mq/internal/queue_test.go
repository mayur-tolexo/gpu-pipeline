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

	msg := Message{ID: "m-1", Key: "k1", Payload: []byte("hello"), Timestamp: time.Now()}
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

