package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestClient_CreateTopic(t *testing.T) {
	server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/topics" {
			t.Fatalf("wrong path")
		}
		w.WriteHeader(http.StatusCreated)
	})

	defer server.Close()

	c := New(server.URL)

	err := c.CreateTopic("test", 3)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestClient_Publish(t *testing.T) {
	server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/topics/test/publish" {
			t.Fatalf("wrong path")
		}

		resp := PublishResponse{
			Partition: 1,
			Offset:    10,
		}

		json.NewEncoder(w).Encode(resp)
	})

	defer server.Close()

	c := New(server.URL)

	p, o, err := c.Publish("test", Message{
		Key:     "k1",
		Payload: []byte("data"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p != 1 || o != 10 {
		t.Fatalf("unexpected response")
	}
}

func TestClient_Consume(t *testing.T) {
	server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/topics/test/consume" {
			t.Fatalf("wrong path")
		}

		resp := ConsumeResponse{
			Messages: []Message{
				{Key: "k1", Payload: []byte("data")},
			},
			NextOffset: 5,
		}

		json.NewEncoder(w).Encode(resp)
	})

	defer server.Close()

	c := New(server.URL)

	res, err := c.Consume("test", "g1", 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res.Messages) != 1 {
		t.Fatalf("expected 1 message")
	}

	if res.NextOffset != 5 {
		t.Fatalf("wrong offset")
	}
}

func TestClient_Ack(t *testing.T) {
	server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/topics/test/ack" {
			t.Fatalf("wrong path")
		}
		w.WriteHeader(http.StatusOK)
	})

	defer server.Close()

	c := New(server.URL)

	err := c.Ack("test", AckRequest{
		Group:     "g1",
		Partition: 0,
		Offset:    5,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Publish_Error(t *testing.T) {
	server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusBadRequest)
	})

	defer server.Close()

	c := New(server.URL)

	_, _, err := c.Publish("test", Message{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestClient_Consume_Error(t *testing.T) {
	server := setupTestServer(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	})

	defer server.Close()

	c := New(server.URL)

	_, err := c.Consume("test", "g1", 0, 10)
	if err == nil {
		t.Fatalf("expected error")
	}
}
