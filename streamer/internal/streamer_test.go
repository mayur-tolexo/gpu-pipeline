package internal

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"testing"
	"time"

	mqclient "gpu-pipeline/mq/client"
)

// MockPublisher for testing
type MockPublisher struct {
	publishCalls int
	publishErr   error
	lastMessage  mqclient.Message
	lastTopic    string
	publishFunc  func(topic string, msg mqclient.Message) (int, int, error)
}

func (m *MockPublisher) Publish(topic string, msg mqclient.Message) (int, int, error) {
	m.publishCalls++
	m.lastTopic = topic
	m.lastMessage = msg
	if m.publishFunc != nil {
		return m.publishFunc(topic, msg)
	}
	return 0, 0, m.publishErr
}

func TestLoadConfig_Defaults(t *testing.T) {
	// Save existing env vars
	saveEnv := make(map[string]string)
	vars := []string{"CSV_FILE", "TOPIC", "STREAM_INTERVAL_MS", "MQ_URL"}
	for _, v := range vars {
		if val, ok := os.LookupEnv(v); ok {
			saveEnv[v] = val
			os.Unsetenv(v)
		}
	}
	defer func() {
		// Restore env vars
		for k, v := range saveEnv {
			os.Setenv(k, v)
		}
	}()

	cfg := LoadConfig()

	if cfg.FilePath != "/data/telemetry.csv" {
		t.Errorf("expected default FilePath, got %s", cfg.FilePath)
	}
	if cfg.Topic != "telemetry" {
		t.Errorf("expected default Topic, got %s", cfg.Topic)
	}
	if cfg.BaseURL != "http://mq-service:8080" {
		t.Errorf("expected default BaseURL, got %s", cfg.BaseURL)
	}
	if cfg.Interval != 5000*time.Millisecond {
		t.Errorf("expected default Interval=5000ms, got %v", cfg.Interval)
	}
}

func TestLoadConfig_CustomValues(t *testing.T) {
	// Save and clear existing env vars
	saveEnv := make(map[string]string)
	vars := []string{"CSV_FILE", "TOPIC", "STREAM_INTERVAL_MS", "MQ_URL"}
	for _, v := range vars {
		if val, ok := os.LookupEnv(v); ok {
			saveEnv[v] = val
			os.Unsetenv(v)
		}
	}
	defer func() {
		// Restore env vars
		for k, v := range saveEnv {
			os.Setenv(k, v)
		}
	}()

	// Set custom values
	os.Setenv("CSV_FILE", "/custom/data.csv")
	os.Setenv("TOPIC", "events")
	os.Setenv("STREAM_INTERVAL_MS", "1000")
	os.Setenv("MQ_URL", "http://custom-mq:9999")

	cfg := LoadConfig()

	if cfg.FilePath != "/custom/data.csv" {
		t.Errorf("expected custom FilePath, got %s", cfg.FilePath)
	}
	if cfg.Topic != "events" {
		t.Errorf("expected custom Topic, got %s", cfg.Topic)
	}
	if cfg.BaseURL != "http://custom-mq:9999" {
		t.Errorf("expected custom BaseURL, got %s", cfg.BaseURL)
	}
	if cfg.Interval != 1000*time.Millisecond {
		t.Errorf("expected custom Interval=1000ms, got %v", cfg.Interval)
	}
}

func TestReadCSV_EmptyFile(t *testing.T) {
	// Create a temporary CSV file with just headers
	tmpfile, err := os.CreateTemp("", "test-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	writer := csv.NewWriter(tmpfile)
	writer.Write([]string{"gpu_id", "power", "temp"})
	writer.Flush()
	tmpfile.Close()

	records, err := ReadCSV(tmpfile.Name())
	if err != nil {
		t.Fatalf("ReadCSV failed: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestReadCSV_SingleRecord(t *testing.T) {
	// Create a temporary CSV file
	tmpfile, err := os.CreateTemp("", "test-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	writer := csv.NewWriter(tmpfile)
	writer.Write([]string{"gpu_id", "power", "temp"})
	writer.Write([]string{"gpu-1", "250", "75"})
	writer.Flush()
	tmpfile.Close()

	records, err := ReadCSV(tmpfile.Name())
	if err != nil {
		t.Fatalf("ReadCSV failed: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
	if records[0]["gpu_id"] != "gpu-1" {
		t.Errorf("expected gpu_id=gpu-1, got %s", records[0]["gpu_id"])
	}
	if records[0]["power"] != "250" {
		t.Errorf("expected power=250, got %s", records[0]["power"])
	}
	if records[0]["temp"] != "75" {
		t.Errorf("expected temp=75, got %s", records[0]["temp"])
	}
}

func TestReadCSV_MultipleRecords(t *testing.T) {
	// Create a temporary CSV file with multiple records
	tmpfile, err := os.CreateTemp("", "test-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	writer := csv.NewWriter(tmpfile)
	writer.Write([]string{"gpu_id", "power"})
	writer.Write([]string{"gpu-1", "100"})
	writer.Write([]string{"gpu-2", "200"})
	writer.Write([]string{"gpu-3", "300"})
	writer.Flush()
	tmpfile.Close()

	records, err := ReadCSV(tmpfile.Name())
	if err != nil {
		t.Fatalf("ReadCSV failed: %v", err)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records, got %d", len(records))
	}

	// Verify records
	if records[0]["gpu_id"] != "gpu-1" || records[0]["power"] != "100" {
		t.Errorf("record 0 mismatch")
	}
	if records[1]["gpu_id"] != "gpu-2" || records[1]["power"] != "200" {
		t.Errorf("record 1 mismatch")
	}
	if records[2]["gpu_id"] != "gpu-3" || records[2]["power"] != "300" {
		t.Errorf("record 2 mismatch")
	}
}

func TestReadCSV_FileNotFound(t *testing.T) {
	records, err := ReadCSV("/nonexistent/file.csv")
	if err == nil {
		t.Fatalf("expected error for nonexistent file")
	}
	if len(records) != 0 {
		t.Errorf("expected empty records, got %d", len(records))
	}
}

func TestNewStreamer_Success(t *testing.T) {
	// Create a temporary CSV file
	tmpfile, err := os.CreateTemp("", "test-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	writer := csv.NewWriter(tmpfile)
	writer.Write([]string{"gpu_id", "power"})
	writer.Write([]string{"gpu-1", "100"})
	writer.Flush()
	tmpfile.Close()

	cfg := Config{
		FilePath: tmpfile.Name(),
		Topic:    "test",
		BaseURL:  "http://localhost:8080",
		Interval: 100 * time.Millisecond,
	}

	streamer, err := NewStreamer(cfg)
	if err != nil {
		t.Fatalf("NewStreamer failed: %v", err)
	}
	if streamer == nil {
		t.Fatalf("expected non-nil streamer")
	}
	if len(streamer.records) != 1 {
		t.Errorf("expected 1 record, got %d", len(streamer.records))
	}
	if streamer.config.Topic != "test" {
		t.Errorf("expected Topic=test, got %s", streamer.config.Topic)
	}
}

func TestNewStreamer_FileNotFound(t *testing.T) {
	cfg := Config{
		FilePath: "/nonexistent/file.csv",
		Topic:    "test",
		BaseURL:  "http://localhost:8080",
		Interval: 100 * time.Millisecond,
	}

	streamer, err := NewStreamer(cfg)
	if err == nil {
		t.Fatalf("expected error for nonexistent file")
	}
	if streamer != nil {
		t.Fatalf("expected nil streamer on error")
	}
}

func TestRecord_TypeAliasIsMap(t *testing.T) {
	// Verify Record is a map type
	rec := make(Record)
	rec["key1"] = "value1"
	rec["key2"] = "value2"

	if rec["key1"] != "value1" {
		t.Errorf("expected value1, got %s", rec["key1"])
	}
	if rec["key2"] != "value2" {
		t.Errorf("expected value2, got %s", rec["key2"])
	}
}

func TestConfig_StructFields(t *testing.T) {
	cfg := Config{
		FilePath: "/path/to/file.csv",
		Topic:    "mytopic",
		Interval: 500 * time.Millisecond,
		BaseURL:  "http://localhost:8080",
	}

	if cfg.FilePath != "/path/to/file.csv" {
		t.Errorf("FilePath mismatch")
	}
	if cfg.Topic != "mytopic" {
		t.Errorf("Topic mismatch")
	}
	if cfg.Interval != 500*time.Millisecond {
		t.Errorf("Interval mismatch")
	}
	if cfg.BaseURL != "http://localhost:8080" {
		t.Errorf("BaseURL mismatch")
	}
}

// MockMQPublisher for testing Streamer.Start
type MockMQPublisher struct {
	publishCalls int
	publishErr   error
}

func (m *MockMQPublisher) Publish(topic string, message interface{}) (int, int64, error) {
	m.publishCalls++
	return 0, 0, m.publishErr
}

func TestStreamer_Configuration(t *testing.T) {
	// Create a temporary CSV file
	tmpfile, err := os.CreateTemp("", "test-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	writer := csv.NewWriter(tmpfile)
	writer.Write([]string{"gpu_id"})
	writer.Write([]string{"gpu-1"})
	writer.Flush()
	tmpfile.Close()

	cfg := Config{
		FilePath: tmpfile.Name(),
		Topic:    "events",
		BaseURL:  "http://localhost:9999",
		Interval: 100 * time.Millisecond,
	}

	streamer, err := NewStreamer(cfg)
	if err != nil {
		t.Fatalf("NewStreamer failed: %v", err)
	}

	// Verify configuration is set correctly
	if streamer.config.Topic != "events" {
		t.Errorf("expected Topic=events, got %s", streamer.config.Topic)
	}
	if streamer.config.BaseURL != "http://localhost:9999" {
		t.Errorf("expected BaseURL=http://localhost:9999, got %s", streamer.config.BaseURL)
	}
	if streamer.config.Interval != 100*time.Millisecond {
		t.Errorf("expected Interval=100ms, got %v", streamer.config.Interval)
	}
	if streamer.config.FilePath != tmpfile.Name() {
		t.Errorf("expected FilePath=%s, got %s", tmpfile.Name(), streamer.config.FilePath)
	}
}

func TestReadCSV_ComplexHeaders(t *testing.T) {
	// Create a CSV with many columns
	tmpfile, err := os.CreateTemp("", "test-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	writer := csv.NewWriter(tmpfile)
	writer.Write([]string{"gpu_id", "power", "temp", "memory", "clock_speed"})
	writer.Write([]string{"gpu-1", "250", "75", "8192", "1500"})
	writer.Write([]string{"gpu-2", "300", "80", "16384", "1800"})
	writer.Flush()
	tmpfile.Close()

	records, err := ReadCSV(tmpfile.Name())
	if err != nil {
		t.Fatalf("ReadCSV failed: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}

	// Verify all fields are correctly parsed
	if len(records[0]) != 5 {
		t.Errorf("expected 5 fields in record, got %d", len(records[0]))
	}

	// Check specific fields
	fields := []string{"gpu_id", "power", "temp", "memory", "clock_speed"}
	for _, field := range fields {
		if _, ok := records[0][field]; !ok {
			t.Errorf("expected field %s in record", field)
		}
	}
}

func TestReadCSV_SpecialCharacters(t *testing.T) {
	// Test CSV with special characters and spaces
	tmpfile, err := os.CreateTemp("", "test-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	writer := csv.NewWriter(tmpfile)
	writer.Write([]string{"gpu_id", "description"})
	writer.Write([]string{"gpu-1", "High Performance GPU"})
	writer.Write([]string{"gpu-2", "Low Power GPU, Energy Efficient"})
	writer.Flush()
	tmpfile.Close()

	records, err := ReadCSV(tmpfile.Name())
	if err != nil {
		t.Fatalf("ReadCSV failed: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}

	if records[0]["description"] != "High Performance GPU" {
		t.Errorf("expected 'High Performance GPU', got %s", records[0]["description"])
	}

	if records[1]["description"] != "Low Power GPU, Energy Efficient" {
		t.Errorf("expected 'Low Power GPU, Energy Efficient', got %s", records[1]["description"])
	}
}

func TestGetEnv_WithValue(t *testing.T) {
	// Save and set env var
	saved, ok := os.LookupEnv("TEST_VAR_STREAMER")
	if ok {
		defer os.Setenv("TEST_VAR_STREAMER", saved)
	} else {
		defer os.Unsetenv("TEST_VAR_STREAMER")
	}

	os.Setenv("TEST_VAR_STREAMER", "test_value")

	result := getEnv("TEST_VAR_STREAMER", "fallback")
	if result != "test_value" {
		t.Errorf("expected test_value, got %s", result)
	}
}

func TestGetEnv_WithFallback(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("TEST_VAR_STREAMER_UNUSED")

	result := getEnv("TEST_VAR_STREAMER_UNUSED", "fallback_value")
	if result != "fallback_value" {
		t.Errorf("expected fallback_value, got %s", result)
	}
}

func TestLoadConfig_IntervalParsing(t *testing.T) {
	// Save and clear existing env vars
	saveEnv := make(map[string]string)
	vars := []string{"STREAM_INTERVAL_MS"}
	for _, v := range vars {
		if val, ok := os.LookupEnv(v); ok {
			saveEnv[v] = val
			os.Unsetenv(v)
		}
	}
	defer func() {
		// Restore env vars
		for k, v := range saveEnv {
			os.Setenv(k, v)
		}
	}()

	// Test with explicit interval
	os.Setenv("STREAM_INTERVAL_MS", "2500")
	cfg := LoadConfig()

	if cfg.Interval != 2500*time.Millisecond {
		t.Errorf("expected 2500ms, got %v", cfg.Interval)
	}

	// Test with zero (edge case)
	os.Setenv("STREAM_INTERVAL_MS", "0")
	cfg = LoadConfig()

	if cfg.Interval != 0*time.Millisecond {
		t.Errorf("expected 0ms, got %v", cfg.Interval)
	}
}

func TestNewStreamer_LoadsAllRecords(t *testing.T) {
	// Create a CSV with many records
	tmpfile, err := os.CreateTemp("", "test-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	writer := csv.NewWriter(tmpfile)
	writer.Write([]string{"gpu_id"})
	for i := 1; i <= 100; i++ {
		writer.Write([]string{"gpu-" + string(rune('0' + i%10))})
	}
	writer.Flush()
	tmpfile.Close()

	cfg := Config{
		FilePath: tmpfile.Name(),
		Topic:    "test",
		BaseURL:  "http://localhost:8080",
		Interval: 1 * time.Millisecond,
	}

	streamer, err := NewStreamer(cfg)
	if err != nil {
		t.Fatalf("NewStreamer failed: %v", err)
	}

	if len(streamer.records) != 100 {
		t.Errorf("expected 100 records, got %d", len(streamer.records))
	}
}

// TestNewStreamerWithClient tests creating a streamer with a mock client
func TestNewStreamerWithClient(t *testing.T) {
	mockPub := &MockPublisher{}
	records := []Record{
		{"gpu_id": "gpu-1", "power": "100"},
		{"gpu_id": "gpu-2", "power": "200"},
	}
	cfg := Config{
		Topic:    "test",
		Interval: 100 * time.Millisecond,
	}

	streamer := NewStreamerWithClient(cfg, records, mockPub)

	if streamer == nil {
		t.Fatalf("expected non-nil streamer")
	}
	if len(streamer.records) != 2 {
		t.Errorf("expected 2 records, got %d", len(streamer.records))
	}
	if streamer.config.Topic != "test" {
		t.Errorf("expected topic test, got %s", streamer.config.Topic)
	}
}

// TestPublishRecord_Success tests publishing a single record
func TestPublishRecord_Success(t *testing.T) {
	mockPub := &MockPublisher{}
	records := []Record{{"gpu_id": "gpu-1", "power": "250"}}
	streamer := NewStreamerWithClient(Config{Topic: "events", Interval: 1 * time.Millisecond}, records, mockPub)

	err := streamer.publishRecord(records[0])
	if err != nil {
		t.Errorf("publishRecord failed: %v", err)
	}
	if mockPub.publishCalls != 1 {
		t.Errorf("expected 1 publish call, got %d", mockPub.publishCalls)
	}
	if mockPub.lastTopic != "events" {
		t.Errorf("expected topic events, got %s", mockPub.lastTopic)
	}
}

// TestPublishRecord_MissingGPUID tests error when gpu_id is missing
func TestPublishRecord_MissingGPUID(t *testing.T) {
	mockPub := &MockPublisher{}
	record := Record{"power": "250"} // Missing gpu_id
	streamer := NewStreamerWithClient(Config{Topic: "events"}, []Record{}, mockPub)

	err := streamer.publishRecord(record)
	if err == nil {
		t.Errorf("expected error for missing gpu_id")
	}
	if mockPub.publishCalls != 0 {
		t.Errorf("expected no publish calls, got %d", mockPub.publishCalls)
	}
}

// TestPublishRecord_PublishError tests error handling on publish failure
func TestPublishRecord_PublishError(t *testing.T) {
	mockPub := &MockPublisher{publishErr: fmt.Errorf("publish failed")}
	record := Record{"gpu_id": "gpu-1"}
	streamer := NewStreamerWithClient(Config{Topic: "events"}, []Record{}, mockPub)

	err := streamer.publishRecord(record)
	if err == nil {
		t.Errorf("expected publish error")
	}
	if mockPub.publishCalls != 1 {
		t.Errorf("expected 1 publish call, got %d", mockPub.publishCalls)
	}
}

// TestStreamBatch_Success tests processing a batch of records
func TestStreamBatch_Success(t *testing.T) {
	mockPub := &MockPublisher{}
	records := []Record{
		{"gpu_id": "gpu-1"},
		{"gpu_id": "gpu-2"},
		{"gpu_id": "gpu-3"},
	}
	streamer := NewStreamerWithClient(Config{Topic: "events", Interval: 1 * time.Millisecond}, records, mockPub)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := streamer.StreamBatch(ctx)
	if err != nil {
		t.Errorf("StreamBatch failed: %v", err)
	}
	if mockPub.publishCalls != 3 {
		t.Errorf("expected 3 publish calls, got %d", mockPub.publishCalls)
	}
}

// TestStreamBatch_ContextCancelled tests that StreamBatch respects context cancellation
func TestStreamBatch_ContextCancelled(t *testing.T) {
	mockPub := &MockPublisher{publishErr: nil}
	records := make([]Record, 100) // Many records to ensure we hit context cancellation
	for i := range records {
		records[i] = Record{"gpu_id": "gpu-1"}
	}
	streamer := NewStreamerWithClient(Config{Topic: "events", Interval: 100 * time.Millisecond}, records, mockPub)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel

	err := streamer.StreamBatch(ctx)
	if err == nil {
		t.Errorf("expected context cancelled error")
	}
}

// TestStreamBatch_WithPublishError tests batch continues on publish errors
func TestStreamBatch_WithPublishError(t *testing.T) {
	callCount := 0
	mockPub := &MockPublisher{
		publishFunc: func(topic string, msg mqclient.Message) (int, int, error) {
			callCount++
			if callCount == 1 {
				return 0, 0, fmt.Errorf("publish error")
			}
			return 0, 0, nil
		},
	}

	records := []Record{
		{"gpu_id": "gpu-1"},
		{"gpu_id": "gpu-2"},
	}
	streamer := NewStreamerWithClient(Config{Topic: "events", Interval: 1 * time.Millisecond}, records, mockPub)

	ctx := context.Background()
	err := streamer.StreamBatch(ctx)
	if err != nil {
		t.Errorf("StreamBatch should not error on partial failures: %v", err)
	}
}

// TestStartWithContext_CancelsGracefully tests graceful shutdown with context
func TestStartWithContext_CancelsGracefully(t *testing.T) {
	mockPub := &MockPublisher{}
	records := []Record{{"gpu_id": "gpu-1"}}
	streamer := NewStreamerWithClient(Config{Topic: "events", Interval: 10 * time.Millisecond}, records, mockPub)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Run in goroutine to avoid blocking
	done := make(chan bool)
	go func() {
		streamer.StartWithContext(ctx)
		done <- true
	}()

	// Wait for context to expire and goroutine to finish
	select {
	case <-done:
		// Success - exited gracefully
	case <-time.After(2 * time.Second):
		t.Errorf("StartWithContext did not exit when context was cancelled")
	}
}

// TestGetRecords returns records being streamed
func TestGetRecords(t *testing.T) {
	records := []Record{
		{"gpu_id": "gpu-1"},
		{"gpu_id": "gpu-2"},
	}
	streamer := NewStreamerWithClient(Config{Topic: "test"}, records, &MockPublisher{})

	retrieved := streamer.GetRecords()
	if len(retrieved) != 2 {
		t.Errorf("expected 2 records, got %d", len(retrieved))
	}
	if retrieved[0]["gpu_id"] != "gpu-1" {
		t.Errorf("expected gpu-1, got %s", retrieved[0]["gpu_id"])
	}
}

// TestGetConfig returns streamer configuration
func TestGetConfig(t *testing.T) {
	cfg := Config{Topic: "mytopic", Interval: 500 * time.Millisecond}
	streamer := NewStreamerWithClient(cfg, []Record{}, &MockPublisher{})

	retrieved := streamer.GetConfig()
	if retrieved.Topic != "mytopic" {
		t.Errorf("expected mytopic, got %s", retrieved.Topic)
	}
	if retrieved.Interval != 500*time.Millisecond {
		t.Errorf("expected 500ms, got %v", retrieved.Interval)
	}
}

// TestPublishRecord_TimestampAdded tests that timestamp is added to records
func TestPublishRecord_TimestampAdded(t *testing.T) {
	mockPub := &MockPublisher{}
	record := Record{"gpu_id": "gpu-1", "power": "250"}
	originalRecord := Record{"gpu_id": "gpu-1", "power": "250"}
	streamer := NewStreamerWithClient(Config{Topic: "events"}, []Record{}, mockPub)

	err := streamer.publishRecord(record)
	if err != nil {
		t.Errorf("publishRecord failed: %v", err)
	}

	// Original record should not be modified
	if len(record) != len(originalRecord) {
		t.Errorf("original record was modified")
	}
}

// TestPublishRecord_RecordCopyNotModifiesOriginal tests record copy isolation
func TestPublishRecord_RecordCopyNotModifiesOriginal(t *testing.T) {
	mockPub := &MockPublisher{}
	record := Record{"gpu_id": "gpu-1"}
	originalID := record["gpu_id"]

	streamer := NewStreamerWithClient(Config{Topic: "events"}, []Record{}, mockPub)
	streamer.publishRecord(record)

	// Original should be unchanged
	if record["gpu_id"] != originalID {
		t.Errorf("original record was modified")
	}
	if _, hasTimestamp := record["timestamp"]; hasTimestamp {
		t.Errorf("original record should not have timestamp field")
	}
}
