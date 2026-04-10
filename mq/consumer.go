package mq

import (
	"context"
	"time"

	internalmq "gpu-pipeline/mq/internal"
)

// ConsumerFunc processes a message and returns whether it succeeded.
type ConsumerFunc func(ctx context.Context, msg internalmq.Message) error

// Consumer polls a partition and acknowledges messages by committing offsets.
// This is a simple pull consumer; it can be extended to push-based or group rebalancing.
type Consumer struct {
	partition *internalmq.Partition
	group     string
	fn        ConsumerFunc
	pollInterval time.Duration
	stopCh    chan struct{}
}

func NewConsumer(part *internalmq.Partition, group string, fn ConsumerFunc) *Consumer {
	return &Consumer{partition: part, group: group, fn: fn, pollInterval: 50 * time.Millisecond, stopCh: make(chan struct{})}
}

// Start begins consuming messages in a goroutine. It returns a stop function.
func (c *Consumer) Start(ctx context.Context) func() {
	quit := make(chan struct{})
	go func() {
		defer close(quit)
		for {
			select {
			case <-ctx.Done():
				return
			case <-c.stopCh:
				return
			default:
				c.pollOnce(ctx)
				time.Sleep(c.pollInterval)
			}
		}
	}()
	return func() {
		close(c.stopCh)
		<-quit
	}
}

func (c *Consumer) pollOnce(ctx context.Context) {
	offset := c.partition.Offset(c.group)
	msgs, _ := c.partition.ReadFrom(offset, 10)
	if len(msgs) == 0 {
		return
	}
	for i, m := range msgs {
		if err := c.fn(ctx, m); err == nil {
			// commit next offset
			_ = c.partition.Commit(c.group, offset+int64(i)+1)
		} else {
			// stop on error for now
			return
		}
	}
}

// SetPollInterval configures how often the consumer polls.
func (c *Consumer) SetPollInterval(d time.Duration) { c.pollInterval = d }
