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

// HandleCreateTopic handles POST /topics
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

// HandlePublish handles POST /topics/{topic}/publish
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

// HandleConsume handles GET /topics/{topic}/consume?group=...&partition=...&batch=...
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

// HandleAck handles POST /topics/{topic}/ack
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

// HandleSwagger returns OpenAPI/Swagger documentation for the API.
func (h *Handler) HandleSwagger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	swagger := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "Message Queue API",
			"version":     "1.0.0",
			"description": "A lightweight, partitioned message queue with consumer groups and consistent hashing",
		},
		"servers": []map[string]interface{}{
			{
				"url":         "http://localhost:8080",
				"description": "Local development server",
			},
		},
		"paths": map[string]interface{}{
			"/topics": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Create a new topic",
					"operationId": "createTopic",
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"name":               map[string]string{"type": "string"},
										"partitions":         map[string]string{"type": "integer"},
										"partition_capacity": map[string]string{"type": "integer"},
									},
									"required": []string{"name", "partitions"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"201": map[string]string{"description": "Topic created successfully"},
						"400": map[string]string{"description": "Bad request"},
						"409": map[string]string{"description": "Topic already exists"},
					},
				},
			},
			"/topics/{topic}/publish": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Publish a message to a topic",
					"operationId": "publishMessage",
					"parameters": []map[string]interface{}{
						{
							"name":        "topic",
							"in":          "path",
							"required":    true,
							"schema":      map[string]string{"type": "string"},
							"description": "Topic name",
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"id":      map[string]string{"type": "string"},
										"key":     map[string]string{"type": "string"},
										"payload": map[string]string{"type": "string"},
									},
									"required": []string{"key", "payload"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]string{"description": "Message published"},
						"400": map[string]string{"description": "Bad request (missing key)"},
						"404": map[string]string{"description": "Topic not found"},
					},
				},
			},
			"/topics/{topic}/consume": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Consume messages from a topic partition",
					"operationId": "consumeMessages",
					"parameters": []map[string]interface{}{
						{
							"name":        "topic",
							"in":          "path",
							"required":    true,
							"schema":      map[string]string{"type": "string"},
							"description": "Topic name",
						},
						{
							"name":        "group",
							"in":          "query",
							"required":    true,
							"schema":      map[string]string{"type": "string"},
							"description": "Consumer group name",
						},
						{
							"name":        "partition",
							"in":          "query",
							"required":    true,
							"schema":      map[string]string{"type": "integer"},
							"description": "Partition index",
						},
						{
							"name":        "batch",
							"in":          "query",
							"required":    true,
							"schema":      map[string]string{"type": "integer"},
							"description": "Batch size",
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]string{"description": "Messages retrieved"},
						"400": map[string]string{"description": "Bad request"},
						"404": map[string]string{"description": "Topic not found"},
					},
				},
			},
			"/topics/{topic}/ack": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Acknowledge messages (commit offset)",
					"operationId": "ackMessages",
					"parameters": []map[string]interface{}{
						{
							"name":        "topic",
							"in":          "path",
							"required":    true,
							"schema":      map[string]string{"type": "string"},
							"description": "Topic name",
						},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"group":     map[string]string{"type": "string"},
										"partition": map[string]string{"type": "integer"},
										"offset":    map[string]string{"type": "integer"},
									},
									"required": []string{"group", "partition", "offset"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]string{"description": "Offset committed"},
						"400": map[string]string{"description": "Bad request"},
						"404": map[string]string{"description": "Topic not found"},
					},
				},
			},
			"/healthz": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Health check endpoint",
					"operationId": "health",
					"responses": map[string]interface{}{
						"200": map[string]string{"description": "Service is healthy"},
					},
				},
			},
			"/swagger": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "OpenAPI/Swagger specification",
					"operationId": "getSwagger",
					"responses": map[string]interface{}{
						"200": map[string]string{"description": "OpenAPI specification"},
					},
				},
			},
		},
		"components": map[string]interface{}{
			"schemas": map[string]interface{}{
				"Message": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"id":        map[string]string{"type": "string"},
						"key":       map[string]string{"type": "string"},
						"payload":   map[string]string{"type": "string"},
						"timestamp": map[string]string{"type": "string", "format": "date-time"},
						"offset":    map[string]string{"type": "integer"},
					},
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(swagger)
}
