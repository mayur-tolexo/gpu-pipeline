package mq

import (
	internalmq "gpu-pipeline/mq/internal"
	"strconv"
	"testing"
)

func TestTopic_ProducePartitioning(t *testing.T) {
	topic := NewTopic("top", 3)
	producer := NewProducer(topic)
	// messages with different keys should route to some partition
	for i := 0; i < 10; i++ {
		idx, _ := producer.Send(internalmq.Message{ID: "id", Key: "k" + strconv.Itoa(i)})
		if idx < 0 || idx >= topic.Partitions() {
			t.Fatalf("invalid partition index %d", idx)
		}
	}
	// empty key goes to partition 0 by default
	idx, _ := producer.Send(internalmq.Message{ID: "id2", Key: ""})
	if idx != 0 {
		t.Fatalf("expected empty key to go to partition 0, got %d", idx)
	}
}
