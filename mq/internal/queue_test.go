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
