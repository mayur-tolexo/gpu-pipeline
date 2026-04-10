package mq

import internalmq "gpu-pipeline/mq/internal"

// Producer is a thin wrapper that sends messages to a Topic.
type Producer struct {
	topic *Topic
}

func NewProducer(t *Topic) *Producer { return &Producer{topic: t} }

// Send sends the message and returns partition index and offset.
func (p *Producer) Send(msg internalmq.Message) (int, int64) {
	return p.topic.Produce(msg)
}
