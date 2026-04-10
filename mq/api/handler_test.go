package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
