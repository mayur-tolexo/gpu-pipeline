package client

type Message struct {
	Key     string            `json:"key"`
	Payload []byte            `json:"payload"`
	Headers map[string]string `json:"headers,omitempty"`
}

type PublishResponse struct {
	Partition int `json:"partition"`
	Offset    int `json:"offset"`
}

type ConsumeResponse struct {
	Messages   []Message `json:"messages"`
	NextOffset int       `json:"next_offset"`
}

type AckRequest struct {
	Group     string `json:"group"`
	Partition int    `json:"partition"`
	Offset    int    `json:"offset"`
}
