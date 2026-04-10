package mq

import (
	"context"
	internalmq "gpu-pipeline/mq/internal"
	"sync/atomic"
	"testing"
	"time"
)

func TestConsumer_ProcessAndCommit(t *testing.T) {
	topic := NewTopic("ct", 1)
	part := topic.GetPartition(0)
	p := NewProducer(topic)
	for i := 0; i < 5; i++ {
		p.Send(internalmq.Message{ID: "m", Payload: []byte{byte(i)}})
	}
	var processed int32
	cfn := func(ctx context.Context, msg internalmq.Message) error {
		atomic.AddInt32(&processed, 1)
		return nil
	}
	c := NewConsumer(part, "cg", cfn)
	stop := c.Start(context.Background())
	// give it some time to process
	time.Sleep(200 * time.Millisecond)
	stop()
	if processed == 0 {
		t.Fatalf("expected some messages processed")
	}
	if part.Offset("cg") != int64(processed) {
		t.Fatalf("expected committed offset to equal processed count")
	}
}
