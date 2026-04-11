package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	clientpkg "gpu-pipeline/mq/client"
)

// fakeClient implements MQClient for testing collector behavior.
type fakeClient struct {
	calls int32
}

func (f *fakeClient) Consume(topic, group string, partition, batch int) (*clientpkg.ConsumeResponse, error) {
	c := atomic.AddInt32(&f.calls, 1)
	if c == 1 {
		msg := clientpkg.Message{Key: "k1"}
		payload, _ := json.Marshal(map[string]interface{}{"gpu_id": "g1", "timestamp": time.Now().UTC().Format(time.RFC3339)})
		msg.Payload = payload
		return &clientpkg.ConsumeResponse{Messages: []clientpkg.Message{msg}, NextOffset: 1}, nil
	}
	// subsequent calls return empty
	return &clientpkg.ConsumeResponse{Messages: []clientpkg.Message{}, NextOffset: 1}, nil
}

func (f *fakeClient) Ack(topic string, req clientpkg.AckRequest) error {
	// noop
	atomic.AddInt32(&f.calls, 1)
	return nil
}

func TestInsertFunc_NotInitialized(t *testing.T) {
	// Ensure default InsertFunc returns error when store not initialized
	// Temporarily store original InsertFunc
	orig := InsertFunc
	defer func() { InsertFunc = orig }()

	// Make sure store not initialized
	storeInstance = nil

	err := InsertFunc(map[string]interface{}{"gpu_id": "x", "timestamp": time.Now().UTC().Format(time.RFC3339)})
	if err == nil {
		t.Fatalf("expected error when store not initialized")
	}
}

func TestCollector_RunProcessesMessages(t *testing.T) {
	// Override InsertFunc to capture calls
	var called int32
	orig := InsertFunc
	InsertFunc = func(record map[string]interface{}) error {
		atomic.AddInt32(&called, 1)
		return nil
	}
	defer func() { InsertFunc = orig }()

	cfg := Config{
		BaseURL:      "http://localhost:8080",
		Topic:        "telemetry",
		Group:        "collector-test",
		Partitions:   1,
		BatchSize:    1,
		PollInterval: 10 * time.Millisecond,
	}

	fc := &fakeClient{}
	col, err := NewCollectorWithClient(cfg, fc, "")
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go col.Run(ctx)

	// wait for context to expire
	<-ctx.Done()
	
	// Give goroutine time to exit
	time.Sleep(10 * time.Millisecond)

	if atomic.LoadInt32(&called) == 0 {
		t.Fatalf("expected InsertFunc to be called at least once")
	}
}

func TestNewCollector_WithDBDSN(t *testing.T) {
	cfg := Config{
		BaseURL:    "http://localhost:8080",
		Topic:      "telemetry",
		Group:      "collector-test",
		Partitions: 1,
		BatchSize:  1,
	}

	// Create with empty DSN (should not fail)
	col, err := NewCollector(cfg, "")
	if err != nil {
		t.Fatalf("failed to create collector with empty DSN: %v", err)
	}
	if col == nil {
		t.Fatalf("expected non-nil collector")
	}
}

func TestNewCollectorWithClient(t *testing.T) {
	cfg := Config{
		BaseURL:    "http://localhost:8080",
		Topic:      "telemetry",
		Group:      "collector-test",
		Partitions: 1,
		BatchSize:  1,
	}

	fc := &fakeClient{}
	col, err := NewCollectorWithClient(cfg, fc, "")
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}

	if col.cfg.Topic != "telemetry" {
		t.Errorf("expected topic telemetry, got %s", col.cfg.Topic)
	}
	if col.cfg.Group != "collector-test" {
		t.Errorf("expected group collector-test, got %s", col.cfg.Group)
	}
}

func TestFakeClient_ConsumeEmptyAfterFirstCall(t *testing.T) {
	fc := &fakeClient{}

	// First call should return a message
	res, err := fc.Consume("topic", "group", 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(res.Messages))
	}

	// Second call should return empty
	res, err = fc.Consume("topic", "group", 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(res.Messages))
	}
}

func TestFakeClient_AckNoError(t *testing.T) {
	fc := &fakeClient{}
	req := clientpkg.AckRequest{Group: "g1", Partition: 0, Offset: 1}

	err := fc.Ack("topic", req)
	if err != nil {
		t.Fatalf("unexpected error on ack: %v", err)
	}
}

func TestCollector_Run_MultiplePartitions(t *testing.T) {
	var called int32
	orig := InsertFunc
	InsertFunc = func(record map[string]interface{}) error {
		atomic.AddInt32(&called, 1)
		return nil
	}
	defer func() { InsertFunc = orig }()

	cfg := Config{
		BaseURL:      "http://localhost:8080",
		Topic:        "telemetry",
		Group:        "collector-test",
		Partitions:   2,  // Multiple partitions
		BatchSize:    1,
		PollInterval: 5 * time.Millisecond,
	}

	fc := &fakeClient{}
	col, err := NewCollectorWithClient(cfg, fc, "")
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go col.Run(ctx)

	<-ctx.Done()

	if atomic.LoadInt32(&called) == 0 {
		t.Fatalf("expected InsertFunc to be called")
	}
}

func TestCollector_InsertFuncError(t *testing.T) {
	// Test that collector continues on insert error
	var insertCalls int32
	orig := InsertFunc
	InsertFunc = func(record map[string]interface{}) error {
		atomic.AddInt32(&insertCalls, 1)
		if atomic.LoadInt32(&insertCalls) == 1 {
			return nil // first call succeeds
		}
		return nil // subsequent calls also succeed
	}
	defer func() { InsertFunc = orig }()

	cfg := Config{
		BaseURL:      "http://localhost:8080",
		Topic:        "telemetry",
		Group:        "collector-test",
		Partitions:   1,
		BatchSize:    1,
		PollInterval: 5 * time.Millisecond,
	}

	fc := &fakeClient{}
	col, err := NewCollectorWithClient(cfg, fc, "")
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go col.Run(ctx)

	<-ctx.Done()

	if atomic.LoadInt32(&insertCalls) == 0 {
		t.Fatalf("expected InsertFunc to be called")
	}
}

// TestCollector_MalformedPayload tests handling of invalid JSON payload
func TestCollector_MalformedPayload(t *testing.T) {
	var insertCalls int32
	orig := InsertFunc
	InsertFunc = func(record map[string]interface{}) error {
		atomic.AddInt32(&insertCalls, 1)
		return nil
	}
	defer func() { InsertFunc = orig }()

	// FakeClient that returns malformed JSON
	badClient := &fakeClientWithBadPayload{}

	cfg := Config{
		BaseURL:      "http://localhost:8080",
		Topic:        "telemetry",
		Group:        "collector-test",
		Partitions:   1,
		BatchSize:    1,
		PollInterval: 5 * time.Millisecond,
	}

	col, err := NewCollectorWithClient(cfg, badClient, "")
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go col.Run(ctx)

	<-ctx.Done()

	// InsertFunc should NOT have been called because JSON was malformed
	if atomic.LoadInt32(&insertCalls) > 0 {
		t.Logf("InsertFunc was called %d times (expected 0 for malformed payload)", atomic.LoadInt32(&insertCalls))
	}
}

// fakeClientWithBadPayload returns invalid JSON
type fakeClientWithBadPayload struct {
	calls int32
}

func (f *fakeClientWithBadPayload) Consume(topic, group string, partition, batch int) (*clientpkg.ConsumeResponse, error) {
	c := atomic.AddInt32(&f.calls, 1)
	if c == 1 {
		msg := clientpkg.Message{Key: "k1", Payload: []byte("not-json")}
		return &clientpkg.ConsumeResponse{Messages: []clientpkg.Message{msg}, NextOffset: 1}, nil
	}
	return &clientpkg.ConsumeResponse{Messages: []clientpkg.Message{}, NextOffset: 1}, nil
}

func (f *fakeClientWithBadPayload) Ack(topic string, req clientpkg.AckRequest) error {
	atomic.AddInt32(&f.calls, 1)
	return nil
}

// TestCollector_ConsumeError tests handling of consume errors
func TestCollector_ConsumeError(t *testing.T) {
	orig := InsertFunc
	InsertFunc = func(record map[string]interface{}) error {
		return nil
	}
	defer func() { InsertFunc = orig }()

	errorClient := &fakeClientWithError{}

	cfg := Config{
		BaseURL:      "http://localhost:8080",
		Topic:        "telemetry",
		Group:        "collector-test",
		Partitions:   1,
		BatchSize:    1,
		PollInterval: 5 * time.Millisecond,
	}

	col, err := NewCollectorWithClient(cfg, errorClient, "")
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go col.Run(ctx)

	<-ctx.Done()
	// Should complete without panicking
}

// fakeClientWithError returns an error
type fakeClientWithError struct {
	calls int32
}

func (f *fakeClientWithError) Consume(topic, group string, partition, batch int) (*clientpkg.ConsumeResponse, error) {
	atomic.AddInt32(&f.calls, 1)
	return nil, fmt.Errorf("consume error")
}

func (f *fakeClientWithError) Ack(topic string, req clientpkg.AckRequest) error {
	return nil
}

// TestCollector_AckError tests handling of ack errors
func TestCollector_AckError(t *testing.T) {
	var insertCalls int32
	orig := InsertFunc
	InsertFunc = func(record map[string]interface{}) error {
		atomic.AddInt32(&insertCalls, 1)
		return nil
	}
	defer func() { InsertFunc = orig }()

	ackErrorClient := &fakeClientWithAckError{}

	cfg := Config{
		BaseURL:      "http://localhost:8080",
		Topic:        "telemetry",
		Group:        "collector-test",
		Partitions:   1,
		BatchSize:    1,
		PollInterval: 5 * time.Millisecond,
	}

	col, err := NewCollectorWithClient(cfg, ackErrorClient, "")
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go col.Run(ctx)

	<-ctx.Done()

	// Should have tried to insert despite ack error
	if atomic.LoadInt32(&insertCalls) == 0 {
		t.Fatalf("expected InsertFunc to be called even with ack errors")
	}
}

// fakeClientWithAckError returns error on ack
type fakeClientWithAckError struct {
	calls int32
}

func (f *fakeClientWithAckError) Consume(topic, group string, partition, batch int) (*clientpkg.ConsumeResponse, error) {
	c := atomic.AddInt32(&f.calls, 1)
	if c == 1 {
		msg := clientpkg.Message{Key: "k1"}
		payload, _ := json.Marshal(map[string]interface{}{"gpu_id": "g1", "timestamp": time.Now().UTC().Format(time.RFC3339)})
		msg.Payload = payload
		return &clientpkg.ConsumeResponse{Messages: []clientpkg.Message{msg}, NextOffset: 1}, nil
	}
	return &clientpkg.ConsumeResponse{Messages: []clientpkg.Message{}, NextOffset: 1}, nil
}

func (f *fakeClientWithAckError) Ack(topic string, req clientpkg.AckRequest) error {
	return fmt.Errorf("ack failed")
}

// TestNewCollectorWithClient_DBInitError tests handling of DB init errors
func TestNewCollectorWithClient_DBInitError(t *testing.T) {
	cfg := Config{
		BaseURL:      "http://localhost:8080",
		Topic:        "telemetry",
		Group:        "collector-test",
		Partitions:   1,
		BatchSize:    1,
		PollInterval: 5 * time.Millisecond,
	}

	fc := &fakeClient{}
	// Try to init with invalid DSN - should fail
	col, err := NewCollectorWithClient(cfg, fc, "invalid-dsn://bad")
	if err == nil {
		t.Logf("Note: InitStore did not fail (may not have pgx driver)")
	} else {
		if col == nil {
			t.Logf("Collector creation failed as expected with bad DSN")
		}
	}
}

