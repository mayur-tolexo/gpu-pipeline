package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	internalmq "gpu-pipeline/mq/internal"
)

// Handler wraps the queue and provides HTTP request handlers.
type Handler struct {
	Queue *internalmq.Queue
}

// NewHandler creates a new API handler.
func NewHandler(q *internalmq.Queue) *Handler {
	return &Handler{Queue: q}
}

// CreateTopicRequest for POST /topics
type CreateTopicRequest struct {
	Name              string `json:"name"`
	Partitions        int    `json:"partitions"`
	PartitionCapacity int    `json:"partition_capacity,omitempty"`
}

// PublishRequest for POST /topics/{topic}/publish
type PublishRequest struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	Payload string `json:"payload"`
}

// PublishResponse for POST /topics/{topic}/publish
type PublishResponse struct {
	Partition int   `json:"partition"`
	Offset    int64 `json:"offset"`
}

// ConsumeResponse for GET /topics/{topic}/consume
type ConsumeResponse struct {
	Messages   []MessageResponse `json:"messages"`
	NextOffset int64             `json:"next_offset"`
}

// MessageResponse for a message in consume response
type MessageResponse struct {
	Offset  int64  `json:"offset"`
	ID      string `json:"id"`
	Key     string `json:"key"`
	Payload string `json:"payload"`
}

// AckRequest for POST /topics/{topic}/ack
type AckRequest struct {
	Group     string `json:"group"`
	Partition int    `json:"partition"`
	Offset    int64  `json:"offset"`
}

// HandleCreateTopic godoc
// @Summary Create a new topic
// @Description Create a topic with given number of partitions
// @Tags topics
// @Accept json
// @Produce json
// @Param request body CreateTopicRequest true "Create Topic"
// @Success 201 {object} map[string]string
// @Failure 400 {string} string
// @Failure 409 {string} string
// @Router /topics [post]
func (h *Handler) HandleCreateTopic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req CreateTopicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Partitions <= 0 {
		http.Error(w, "name and partitions required", http.StatusBadRequest)
		return
	}
	if err := h.Queue.CreateTopic(req.Name, req.Partitions, req.PartitionCapacity); err != nil {
		if err == internalmq.ErrTopicExists {
			http.Error(w, "topic already exists", http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "topic created"})
}

// HandlePublish godoc
// @Summary Publish a message
// @Description Publish a message to a topic
// @Tags messages
// @Accept json
// @Produce json
// @Param topic path string true "Topic name"
// @Param request body PublishRequest true "Message payload"
// @Success 200 {object} PublishResponse
// @Failure 400 {string} string
// @Failure 404 {string} string
// @Router /topics/{topic}/publish [post]
func (h *Handler) HandlePublish(w http.ResponseWriter, r *http.Request, topic string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req PublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Key == "" {
		http.Error(w, "key required", http.StatusBadRequest)
		return
	}
	partition, offset, err := h.Queue.Publish(topic, internalmq.Message{
		ID:      req.ID,
		Key:     req.Key,
		Payload: []byte(req.Payload),
	})
	if err != nil {
		if err == internalmq.ErrTopicNotFound {
			http.Error(w, "topic not found", http.StatusNotFound)
		} else if err == internalmq.ErrPartitionFull {
			http.Error(w, "partition full", http.StatusInsufficientStorage)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(PublishResponse{Partition: partition, Offset: offset})
}

// HandleConsume godoc
// @Summary Consume messages
// @Description Fetch messages from a partition
// @Tags messages
// @Produce json
// @Param topic path string true "Topic name"
// @Param group query string true "Consumer group"
// @Param partition query int true "Partition ID"
// @Param batch query int false "Batch size"
// @Success 200 {object} ConsumeResponse
// @Failure 400 {string} string
// @Failure 404 {string} string
// @Router /topics/{topic}/consume [get]
func (h *Handler) HandleConsume(w http.ResponseWriter, r *http.Request, topic string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	group := r.URL.Query().Get("group")
	partStr := r.URL.Query().Get("partition")
	batchStr := r.URL.Query().Get("batch")

	if group == "" || partStr == "" {
		http.Error(w, "group and partition required", http.StatusBadRequest)
		return
	}
	partition, err := strconv.Atoi(partStr)
	if err != nil {
		http.Error(w, "invalid partition", http.StatusBadRequest)
		return
	}
	batch := 1
	if batchStr != "" {
		batch, err = strconv.Atoi(batchStr)
		if err != nil || batch <= 0 {
			http.Error(w, "batch must be > 0", http.StatusBadRequest)
			return
		}
	}
	msgs, err := h.Queue.Consume(topic, group, partition, batch)
	if err != nil {
		if err == internalmq.ErrTopicNotFound {
			http.Error(w, "topic not found", http.StatusNotFound)
		} else if err == internalmq.ErrPartitionRange {
			http.Error(w, "partition out of range", http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// build response with offsets
	msgResp := make([]MessageResponse, 0, len(msgs))
	t, _ := h.Queue.GetTopic(topic)
	if t != nil {
		part := t.GetPartition(partition)
		if part != nil {
			startOffset := part.Offset(group)
			for i, m := range msgs {
				msgResp = append(msgResp, MessageResponse{
					Offset:  startOffset + int64(i),
					ID:      m.ID,
					Key:     m.Key,
					Payload: string(m.Payload),
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ConsumeResponse{
		Messages:   msgResp,
		NextOffset: int64(len(msgs)),
	})
}

// HandleAck godoc
// @Summary Commit offset
// @Description Acknowledge processed messages
// @Tags messages
// @Accept json
// @Produce json
// @Param topic path string true "Topic name"
// @Param request body AckRequest true "Ack payload"
// @Success 200 {object} map[string]string
// @Failure 400 {string} string
// @Failure 404 {string} string
// @Router /topics/{topic}/ack [post]
func (h *Handler) HandleAck(w http.ResponseWriter, r *http.Request, topic string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req AckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Group == "" {
		http.Error(w, "group required", http.StatusBadRequest)
		return
	}
	if err := h.Queue.Ack(topic, req.Group, req.Partition, req.Offset); err != nil {
		if err == internalmq.ErrTopicNotFound {
			http.Error(w, "topic not found", http.StatusNotFound)
		} else if err == internalmq.ErrPartitionRange {
			http.Error(w, "partition out of range", http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "ack received"})
}
