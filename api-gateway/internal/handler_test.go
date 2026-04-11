package internal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestListGPUs_Success tests successful listing of GPUs
func TestListGPUs_Success(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	handler := NewGPUHandler(gormDB)

	// Mock the DISTINCT query
	rows := sqlmock.NewRows([]string{"gpu_id"}).
		AddRow("gpu-001").
		AddRow("gpu-002").
		AddRow("gpu-003")

	mock.ExpectQuery("SELECT DISTINCT").
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus", nil)
	w := httptest.NewRecorder()

	handler.ListGPUs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp ListGPUsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if len(resp.GPUs) != 3 {
		t.Errorf("expected 3 GPUs, got %d", len(resp.GPUs))
	}

	if resp.Count != 3 {
		t.Errorf("expected count 3, got %d", resp.Count)
	}

	if resp.GPUs[0] != "gpu-001" {
		t.Errorf("expected first GPU to be gpu-001, got %s", resp.GPUs[0])
	}
}

// TestListGPUs_EmptyResult tests listing GPUs when none exist
func TestListGPUs_EmptyResult(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	handler := NewGPUHandler(gormDB)

	// Mock empty result
	rows := sqlmock.NewRows([]string{"gpu_id"})

	mock.ExpectQuery("SELECT DISTINCT").
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus", nil)
	w := httptest.NewRecorder()

	handler.ListGPUs(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var response ListGPUsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Count != 0 {
		t.Fatalf("expected 0 GPUs, got %d", response.Count)
	}
}

func TestListGPUs_WrongMethod(t *testing.T) {
	handler := NewGPUHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/gpus", nil)
	w := httptest.NewRecorder()

	handler.ListGPUs(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestQueryTelemetry_Success(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	handler := NewGPUHandler(gormDB)

	now := time.Now().UTC()
	mockData := `{"power": 250.5, "temperature": 75.2}`

	rows := sqlmock.NewRows([]string{"id", "gpu_id", "timestamp", "data", "created_at"}).
		AddRow(1, "gpu-001", now, mockData, now)

	mock.ExpectQuery("SELECT").
		WillReturnRows(rows)

	queryReq := QueryTelemetryRequest{
		GPUID:    "gpu-001",
		StartTime: now.Add(-1 * time.Hour),
		EndTime:   now.Add(1 * time.Hour),
	}

	body, _ := json.Marshal(queryReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/query", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.QueryTelemetry(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var response QueryTelemetryResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.GPUID != "gpu-001" {
		t.Fatalf("expected gpu-001, got %s", response.GPUID)
	}
	if response.Count != 1 {
		t.Fatalf("expected 1 record, got %d", response.Count)
	}
}

func TestQueryTelemetry_MissingGPUID(t *testing.T) {
	handler := NewGPUHandler(nil)

	queryReq := QueryTelemetryRequest{
		GPUID: "", // Empty GPU ID
	}

	body, _ := json.Marshal(queryReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/query", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.QueryTelemetry(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestQueryTelemetry_InvalidJSON(t *testing.T) {
	handler := NewGPUHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/query", bytes.NewReader([]byte("invalid")))
	w := httptest.NewRecorder()

	handler.QueryTelemetry(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestQueryTelemetry_NoData(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}

	handler := NewGPUHandler(gormDB)

	// Mock empty result
	rows := sqlmock.NewRows([]string{"id", "gpu_id", "timestamp", "data", "created_at"})

	mock.ExpectQuery("SELECT").
		WillReturnRows(rows)

	queryReq := QueryTelemetryRequest{
		GPUID: "gpu-missing",
	}

	body, _ := json.Marshal(queryReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/query", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.QueryTelemetry(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestQueryTelemetry_WrongMethod(t *testing.T) {
	handler := NewGPUHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/query", nil)
	w := httptest.NewRecorder()

	handler.QueryTelemetry(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHealth_Success(t *testing.T) {
	handler := NewGPUHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Fatalf("expected status ok, got %s", response["status"])
	}
}

func TestHealth_WrongMethod(t *testing.T) {
	handler := NewGPUHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestNewGPUHandler(t *testing.T) {
	handler := NewGPUHandler(nil)

	if handler == nil {
		t.Fatalf("expected handler to be created")
	}
}

func TestTelemetryRecord_TableName(t *testing.T) {
	record := TelemetryRecord{}
	tableName := record.TableName()

	if tableName != "telemetry" {
		t.Fatalf("expected table name 'telemetry', got %s", tableName)
	}
}
