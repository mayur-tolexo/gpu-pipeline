package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	internal "gpu-pipeline/mq/internal"
)

func TestHandler_CreateTopic(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)

	body, _ := json.Marshal(CreateTopicRequest{Name: "test", Partitions: 3})
	req := httptest.NewRequest(http.MethodPost, "/topics", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleCreateTopic(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	// verify topic was created
	if _, err := q.GetTopic("test"); err != nil {
		t.Fatalf("topic not created: %v", err)
	}
}

func TestHandler_CreateTopicDuplicate(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 3, 0)
	h := NewHandler(q)

	body, _ := json.Marshal(CreateTopicRequest{Name: "test", Partitions: 3})
	req := httptest.NewRequest(http.MethodPost, "/topics", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleCreateTopic(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestHandler_Publish(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 3, 0)
	h := NewHandler(q)

	body, _ := json.Marshal(PublishRequest{ID: "m1", Key: "key1", Payload: "hello"})
	req := httptest.NewRequest(http.MethodPost, "/topics/test/publish", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandlePublish(w, req, "test")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp PublishResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Partition < 0 || resp.Partition >= 3 {
		t.Fatalf("invalid partition: %d", resp.Partition)
	}
	if resp.Offset < 0 {
		t.Fatalf("invalid offset: %d", resp.Offset)
	}
}

func TestHandler_PublishNoKey(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 3, 0)
	h := NewHandler(q)

	body, _ := json.Marshal(PublishRequest{Key: "", Payload: "hello"})
	req := httptest.NewRequest(http.MethodPost, "/topics/test/publish", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandlePublish(w, req, "test")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_Consume(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 3, 0)
	h := NewHandler(q)

	// publish message
	pubBody, _ := json.Marshal(PublishRequest{ID: "m1", Key: "key1", Payload: "data"})
	pubReq := httptest.NewRequest(http.MethodPost, "/topics/test/publish", bytes.NewReader(pubBody))
	pubW := httptest.NewRecorder()
	h.HandlePublish(pubW, pubReq, "test")

	// consume message
	consumeReq := httptest.NewRequest(http.MethodGet, "/topics/test/consume?group=g1&partition=0&batch=10", nil)
	consumeW := httptest.NewRecorder()
	h.HandleConsume(consumeW, consumeReq, "test")

	if consumeW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", consumeW.Code)
	}

	var resp ConsumeResponse
	_ = json.NewDecoder(consumeW.Body).Decode(&resp)
	// might be empty if key hashed to different partition
	_ = resp
}

func TestHandler_Ack(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 3, 0)
	h := NewHandler(q)

	body, _ := json.Marshal(AckRequest{Group: "g1", Partition: 0, Offset: 0})
	req := httptest.NewRequest(http.MethodPost, "/topics/test/ack", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.HandleAck(w, req, "test")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandler_Health(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	// call through RegisterRoutes to test routing
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// Comprehensive API Test Cases (TDD style)

func TestCreateTopic_InvalidName(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	body, _ := json.Marshal(CreateTopicRequest{Name: "", Partitions: 3})
	req := httptest.NewRequest(http.MethodPost, "/topics", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleCreateTopic(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateTopic_InvalidPartitions(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	body, _ := json.Marshal(CreateTopicRequest{Name: "test", Partitions: 0})
	req := httptest.NewRequest(http.MethodPost, "/topics", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleCreateTopic(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateTopic_InvalidJSON(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	req := httptest.NewRequest(http.MethodPost, "/topics", bytes.NewReader([]byte("{invalid json")))
	w := httptest.NewRecorder()
	h.HandleCreateTopic(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateTopic_MethodNotAllowed(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/topics", nil)
	w := httptest.NewRecorder()
	h.HandleCreateTopic(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestPublish_TopicNotFound(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	body, _ := json.Marshal(PublishRequest{Key: "key", Payload: "data"})
	req := httptest.NewRequest(http.MethodPost, "/topics/missing/publish", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.HandlePublish(w, req, "missing")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestPublish_MethodNotAllowed(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 3, 0)
	h := NewHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/topics/test/publish", nil)
	w := httptest.NewRecorder()
	h.HandlePublish(w, req, "test")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestPublish_InvalidJSON(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 3, 0)
	h := NewHandler(q)
	req := httptest.NewRequest(http.MethodPost, "/topics/test/publish", bytes.NewReader([]byte("{bad json")))
	w := httptest.NewRecorder()
	h.HandlePublish(w, req, "test")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestConsume_MethodNotAllowed(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 3, 0)
	h := NewHandler(q)
	req := httptest.NewRequest(http.MethodPost, "/topics/test/consume", nil)
	w := httptest.NewRecorder()
	h.HandleConsume(w, req, "test")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestConsume_InvalidBatchZero(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 3, 0)
	h := NewHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/topics/test/consume?group=g1&partition=0&batch=0", nil)
	w := httptest.NewRecorder()
	h.HandleConsume(w, req, "test")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestConsume_InvalidBatchNegative(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 3, 0)
	h := NewHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/topics/test/consume?group=g1&partition=0&batch=-1", nil)
	w := httptest.NewRecorder()
	h.HandleConsume(w, req, "test")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestConsume_TopicNotFound(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/topics/missing/consume?group=g1&partition=0&batch=10", nil)
	w := httptest.NewRecorder()
	h.HandleConsume(w, req, "missing")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestConsume_PartitionOutOfRange(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 3, 0)
	h := NewHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/topics/test/consume?group=g1&partition=99&batch=10", nil)
	w := httptest.NewRecorder()
	h.HandleConsume(w, req, "test")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAck_MethodNotAllowed(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 3, 0)
	h := NewHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/topics/test/ack", nil)
	w := httptest.NewRecorder()
	h.HandleAck(w, req, "test")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestAck_TopicNotFound(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	body, _ := json.Marshal(AckRequest{Group: "g1", Partition: 0, Offset: 100})
	req := httptest.NewRequest(http.MethodPost, "/topics/missing/ack", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleAck(w, req, "missing")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAck_PartitionOutOfRange(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 3, 0)
	h := NewHandler(q)
	body, _ := json.Marshal(AckRequest{Group: "g1", Partition: 99, Offset: 100})
	req := httptest.NewRequest(http.MethodPost, "/topics/test/ack", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleAck(w, req, "test")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAck_InvalidJSON(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 3, 0)
	h := NewHandler(q)
	req := httptest.NewRequest(http.MethodPost, "/topics/test/ack", bytes.NewReader([]byte("{bad json")))
	w := httptest.NewRecorder()
	h.HandleAck(w, req, "test")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCompact_MethodNotAllowed(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/admin/compact", nil)
	w := httptest.NewRecorder()
	h.HandleCompact(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestCompact_InvalidJSON(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	req := httptest.NewRequest(http.MethodPost, "/admin/compact", bytes.NewReader([]byte("{bad")))
	w := httptest.NewRecorder()
	h.HandleCompact(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCompact_EmptyTopic(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	body, _ := json.Marshal(CompactRequest{Topic: ""})
	req := httptest.NewRequest(http.MethodPost, "/admin/compact", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleCompact(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetStats_MethodNotAllowed(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 3, 0)
	h := NewHandler(q)
	req := httptest.NewRequest(http.MethodPost, "/admin/stats/test/0", nil)
	w := httptest.NewRecorder()
	h.HandleGetPartitionStats(w, req, "test", "0")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestGetStats_InvalidPartition(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 3, 0)
	h := NewHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/admin/stats/test/invalid", nil)
	w := httptest.NewRecorder()
	h.HandleGetPartitionStats(w, req, "test", "invalid")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetStats_PartitionOutOfRange(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 3, 0)
	h := NewHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/admin/stats/test/99", nil)
	w := httptest.NewRecorder()
	h.HandleGetPartitionStats(w, req, "test", "99")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetStats_TopicNotFound(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/admin/stats/missing/0", nil)
	w := httptest.NewRecorder()
	h.HandleGetPartitionStats(w, req, "missing", "0")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// Additional tests for better coverage

func TestCompact_Success(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 2, 0)
	h := NewHandler(q)

	// Publish some messages to create entries for compaction
	msg := internal.Message{Key: "k1", Payload: []byte("data1"), Timestamp: time.Now()}
	q.Publish("test", msg)

	body, _ := json.Marshal(CompactRequest{Topic: "test"})
	req := httptest.NewRequest(http.MethodPost, "/admin/compact", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleCompact(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp CompactResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Topic != "test" {
		t.Fatalf("expected topic test, got %s", resp.Topic)
	}
}

func TestCompact_TopicNotFound(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)

	body, _ := json.Marshal(CompactRequest{Topic: "missing"})
	req := httptest.NewRequest(http.MethodPost, "/admin/compact", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleCompact(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCompact_MissingTopic(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)

	body, _ := json.Marshal(CompactRequest{Topic: ""})
	req := httptest.NewRequest(http.MethodPost, "/admin/compact", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleCompact(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetStats_Success(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 1, 0)
	h := NewHandler(q)

	// Publish a message
	msg := internal.Message{Key: "k1", Payload: []byte("data1"), Timestamp: time.Now()}
	q.Publish("test", msg)

	req := httptest.NewRequest(http.MethodGet, "/admin/stats/test/0", nil)
	w := httptest.NewRecorder()
	h.HandleGetPartitionStats(w, req, "test", "0")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var stats PartitionStats
	json.NewDecoder(w.Body).Decode(&stats)
	if stats.MessageCount != 1 {
		t.Fatalf("expected 1 message, got %d", stats.MessageCount)
	}
}

func TestPublish_Success(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 1, 0)
	h := NewHandler(q)

	body, _ := json.Marshal(PublishRequest{
		ID:      "msg1",
		Key:     "key1",
		Payload: "data1",
	})
	req := httptest.NewRequest(http.MethodPost, "/topics/test/publish", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.HandlePublish(w, req, "test")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp PublishResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Partition < 0 {
		t.Fatalf("expected valid partition, got %d", resp.Partition)
	}
	if resp.Offset < 0 {
		t.Fatalf("expected valid offset, got %d", resp.Offset)
	}
}

func TestConsume_Success(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 1, 0)
	h := NewHandler(q)

	// Publish a message first
	msg := internal.Message{Key: "k1", Payload: []byte("data1"), Timestamp: time.Now()}
	q.Publish("test", msg)

	req := httptest.NewRequest(http.MethodGet, "/topics/test/consume?group=g1&partition=0&batch=10", nil)
	w := httptest.NewRecorder()
	h.HandleConsume(w, req, "test")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAck_Success(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 1, 0)
	h := NewHandler(q)

	body, _ := json.Marshal(AckRequest{
		Group:     "g1",
		Partition: 0,
		Offset:    0,
	})
	req := httptest.NewRequest(http.MethodPost, "/topics/test/ack", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleAck(w, req, "test")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetStats_WrongMethod(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 1, 0)
	h := NewHandler(q)

	req := httptest.NewRequest(http.MethodPost, "/admin/stats/test/0", nil)
	w := httptest.NewRecorder()
	h.HandleGetPartitionStats(w, req, "test", "0")

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestConsume_MissingPartition(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 1, 0)
	h := NewHandler(q)

	req := httptest.NewRequest(http.MethodGet, "/topics/test/consume?group=g1&batch=10", nil)
	w := httptest.NewRecorder()
	h.HandleConsume(w, req, "test")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing partition, got %d", w.Code)
	}
}

func TestConsume_MissingGroup(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 1, 0)
	h := NewHandler(q)

	req := httptest.NewRequest(http.MethodGet, "/topics/test/consume?partition=0&batch=10", nil)
	w := httptest.NewRecorder()
	h.HandleConsume(w, req, "test")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing group, got %d", w.Code)
	}
}

func TestConsume_InvalidPartition(t *testing.T) {
	q := internal.NewQueue(0)
	q.CreateTopic("test", 1, 0)
	h := NewHandler(q)

	req := httptest.NewRequest(http.MethodGet, "/topics/test/consume?group=g1&partition=invalid&batch=10", nil)
	w := httptest.NewRecorder()
	h.HandleConsume(w, req, "test")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid partition, got %d", w.Code)
	}
}
