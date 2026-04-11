package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gpu-pipeline/api-gateway/pkg/models"
)

// MockTelemetryService is a mock implementation of TelemetryService interface
type MockTelemetryService struct {
	ListGPUsFunc       func() (*models.ListGPUsResponse, error)
	QueryTelemetryFunc func(*models.QueryTelemetryRequest) (*models.QueryTelemetryResponse, error)
	GetGPUTelemetryFunc func(gpuID string, startTime, endTime string) (*models.QueryTelemetryResponse, error)
}

func (m *MockTelemetryService) ListGPUs() (*models.ListGPUsResponse, error) {
	if m.ListGPUsFunc != nil {
		return m.ListGPUsFunc()
	}
	return &models.ListGPUsResponse{GPUs: []string{}, Count: 0}, nil
}

func (m *MockTelemetryService) QueryTelemetry(req *models.QueryTelemetryRequest) (*models.QueryTelemetryResponse, error) {
	if m.QueryTelemetryFunc != nil {
		return m.QueryTelemetryFunc(req)
	}
	return nil, nil
}

func (m *MockTelemetryService) GetGPUTelemetry(gpuID string, startTime, endTime string) (*models.QueryTelemetryResponse, error) {
	if m.GetGPUTelemetryFunc != nil {
		return m.GetGPUTelemetryFunc(gpuID, startTime, endTime)
	}
	return nil, nil
}

// TestListGPUs_Success tests successful listing of GPUs
func TestListGPUs_Success(t *testing.T) {
	mockService := &MockTelemetryService{
		ListGPUsFunc: func() (*models.ListGPUsResponse, error) {
			return &models.ListGPUsResponse{
				GPUs:  []string{"gpu-001", "gpu-002", "gpu-003"},
				Count: 3,
			}, nil
		},
	}

	handler := NewGPUHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus", nil)
	w := httptest.NewRecorder()

	handler.ListGPUs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp models.ListGPUsResponse
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
	mockService := &MockTelemetryService{
		ListGPUsFunc: func() (*models.ListGPUsResponse, error) {
			return &models.ListGPUsResponse{
				GPUs:  []string{},
				Count: 0,
			}, nil
		},
	}

	handler := NewGPUHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus", nil)
	w := httptest.NewRecorder()

	handler.ListGPUs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response models.ListGPUsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if response.Count != 0 {
		t.Errorf("expected 0 GPUs, got %d", response.Count)
	}
}

// TestListGPUs_WrongMethod tests wrong HTTP method
func TestListGPUs_WrongMethod(t *testing.T) {
	handler := NewGPUHandler(&MockTelemetryService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/gpus", nil)
	w := httptest.NewRecorder()

	handler.ListGPUs(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// TestListGPUs_Error tests error handling
func TestListGPUs_Error(t *testing.T) {
	mockService := &MockTelemetryService{
		ListGPUsFunc: func() (*models.ListGPUsResponse, error) {
			return nil, fmt.Errorf("database error")
		},
	}

	handler := NewGPUHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus", nil)
	w := httptest.NewRecorder()

	handler.ListGPUs(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// TestQueryTelemetry_Success tests successful telemetry query
func TestQueryTelemetry_Success(t *testing.T) {
	now := time.Now()
	mockService := &MockTelemetryService{
		QueryTelemetryFunc: func(req *models.QueryTelemetryRequest) (*models.QueryTelemetryResponse, error) {
			return &models.QueryTelemetryResponse{
				GPUID: "gpu-001",
				Records: []models.TelemetryResponse{
					{
						GPUID:     "gpu-001",
						Timestamp: now,
						Data: map[string]interface{}{
							"power":       250.5,
							"temperature": 75.2,
						},
					},
				},
				Count:           1,
				StartTime:       now,
				EndTime:         now,
				FieldsAvailable: []string{"power", "temperature"},
			}, nil
		},
	}

	handler := NewGPUHandler(mockService)

	queryReq := models.QueryTelemetryRequest{
		GPUID:     "gpu-001",
		StartTime: now.Add(-1 * time.Hour),
		EndTime:   now.Add(1 * time.Hour),
	}

	body, _ := json.Marshal(queryReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/query", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.QueryTelemetry(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var response models.QueryTelemetryResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if response.GPUID != "gpu-001" {
		t.Errorf("expected gpu-001, got %s", response.GPUID)
	}
	if response.Count != 1 {
		t.Errorf("expected 1 record, got %d", response.Count)
	}
}

// TestQueryTelemetry_InvalidJSON tests invalid JSON body
func TestQueryTelemetry_InvalidJSON(t *testing.T) {
	handler := NewGPUHandler(&MockTelemetryService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/query", bytes.NewReader([]byte("invalid")))
	w := httptest.NewRecorder()

	handler.QueryTelemetry(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// TestQueryTelemetry_NoData tests query with no results
func TestQueryTelemetry_NoData(t *testing.T) {
	mockService := &MockTelemetryService{
		QueryTelemetryFunc: func(req *models.QueryTelemetryRequest) (*models.QueryTelemetryResponse, error) {
			return nil, fmt.Errorf("no telemetry data found for gpu_id: gpu-missing")
		},
	}

	handler := NewGPUHandler(mockService)

	queryReq := models.QueryTelemetryRequest{
		GPUID: "gpu-missing",
	}

	body, _ := json.Marshal(queryReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/query", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.QueryTelemetry(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// TestQueryTelemetry_WrongMethod tests wrong HTTP method
func TestQueryTelemetry_WrongMethod(t *testing.T) {
	handler := NewGPUHandler(&MockTelemetryService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/query", nil)
	w := httptest.NewRecorder()

	handler.QueryTelemetry(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// TestHealth_Success tests health check
func TestHealth_Success(t *testing.T) {
	handler := NewGPUHandler(&MockTelemetryService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("expected status ok, got %s", response["status"])
	}
}

// TestHealth_WrongMethod tests health check with wrong method
func TestHealth_WrongMethod(t *testing.T) {
	handler := NewGPUHandler(&MockTelemetryService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// TestGetGPUTelemetry_Success tests successful retrieval of GPU telemetry
func TestGetGPUTelemetry_Success(t *testing.T) {
	now := time.Now()
	mockService := &MockTelemetryService{
		GetGPUTelemetryFunc: func(gpuID string, startTime, endTime string) (*models.QueryTelemetryResponse, error) {
			return &models.QueryTelemetryResponse{
				GPUID: "gpu-001",
				Records: []models.TelemetryResponse{
					{
						GPUID:     "gpu-001",
						Timestamp: now,
						Data: map[string]interface{}{
							"gpu_util": 45,
							"memory":   8192,
							"temperature": 65,
						},
					},
				},
				Count:           1,
				StartTime:       now,
				EndTime:         now,
				FieldsAvailable: []string{"gpu_util", "memory", "temperature"},
			}, nil
		},
	}

	handler := NewGPUHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus/gpu-001/telemetry", nil)
	w := httptest.NewRecorder()

	handler.GetGPUTelemetry(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp models.QueryTelemetryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if resp.GPUID != "gpu-001" {
		t.Errorf("expected gpu-001, got %s", resp.GPUID)
	}

	if resp.Count != 1 {
		t.Errorf("expected 1 record, got %d", resp.Count)
	}
}

// TestGetGPUTelemetry_WithTimeRange tests with time range parameters
func TestGetGPUTelemetry_WithTimeRange(t *testing.T) {
	now := time.Now()
	mockService := &MockTelemetryService{
		GetGPUTelemetryFunc: func(gpuID string, startTime, endTime string) (*models.QueryTelemetryResponse, error) {
			return &models.QueryTelemetryResponse{
				GPUID:           gpuID,
				Records:         []models.TelemetryResponse{},
				Count:           0,
				FieldsAvailable: []string{},
			}, nil
		},
	}

	handler := NewGPUHandler(mockService)
	startTime := now.Add(-24 * time.Hour).Format(time.RFC3339)
	endTime := now.Add(24 * time.Hour).Format(time.RFC3339)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/gpus/gpu-002/telemetry?start_time="+startTime+"&end_time="+endTime,
		nil,
	)
	w := httptest.NewRecorder()

	handler.GetGPUTelemetry(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// TestGetGPUTelemetry_InvalidStartTime tests with invalid start_time format
func TestGetGPUTelemetry_InvalidStartTime(t *testing.T) {
	mockService := &MockTelemetryService{
		GetGPUTelemetryFunc: func(gpuID string, startTime, endTime string) (*models.QueryTelemetryResponse, error) {
			return nil, fmt.Errorf("invalid start_time format")
		},
	}

	handler := NewGPUHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus/gpu-001/telemetry?start_time=invalid", nil)
	w := httptest.NewRecorder()

	handler.GetGPUTelemetry(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// TestGetGPUTelemetry_InvalidEndTime tests with invalid end_time format
func TestGetGPUTelemetry_InvalidEndTime(t *testing.T) {
	mockService := &MockTelemetryService{
		GetGPUTelemetryFunc: func(gpuID string, startTime, endTime string) (*models.QueryTelemetryResponse, error) {
			return nil, fmt.Errorf("invalid end_time format")
		},
	}

	handler := NewGPUHandler(mockService)
	now := time.Now().UTC()
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/gpus/gpu-001/telemetry?start_time="+now.Format(time.RFC3339)+"&end_time=invalid",
		nil,
	)
	w := httptest.NewRecorder()

	handler.GetGPUTelemetry(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// TestGetGPUTelemetry_NoData tests when GPU has no telemetry data
func TestGetGPUTelemetry_NoData(t *testing.T) {
	mockService := &MockTelemetryService{
		GetGPUTelemetryFunc: func(gpuID string, startTime, endTime string) (*models.QueryTelemetryResponse, error) {
			return nil, fmt.Errorf("no telemetry data found for gpu_id: gpu-missing")
		},
	}

	handler := NewGPUHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus/gpu-missing/telemetry", nil)
	w := httptest.NewRecorder()

	handler.GetGPUTelemetry(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// TestGetGPUTelemetry_WrongMethod tests with wrong HTTP method
func TestGetGPUTelemetry_WrongMethod(t *testing.T) {
	handler := NewGPUHandler(&MockTelemetryService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gpus/gpu-001/telemetry", nil)
	w := httptest.NewRecorder()

	handler.GetGPUTelemetry(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// TestGetGPUTelemetry_InvalidPath tests with invalid path
func TestGetGPUTelemetry_InvalidPath(t *testing.T) {
	handler := NewGPUHandler(&MockTelemetryService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus/", nil)
	w := httptest.NewRecorder()

	handler.GetGPUTelemetry(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// TestGetGPUTelemetry_MultipleRecords tests with multiple telemetry records
func TestGetGPUTelemetry_MultipleRecords(t *testing.T) {
	now := time.Now()
	mockService := &MockTelemetryService{
		GetGPUTelemetryFunc: func(gpuID string, startTime, endTime string) (*models.QueryTelemetryResponse, error) {
			return &models.QueryTelemetryResponse{
				GPUID: "gpu-001",
				Records: []models.TelemetryResponse{
					{GPUID: "gpu-001", Timestamp: now.Add(-2 * time.Hour), Data: map[string]interface{}{"gpu_util": 40}},
					{GPUID: "gpu-001", Timestamp: now.Add(-1 * time.Hour), Data: map[string]interface{}{"gpu_util": 50}},
					{GPUID: "gpu-001", Timestamp: now, Data: map[string]interface{}{"gpu_util": 60}},
				},
				Count:           3,
				StartTime:       now.Add(-2 * time.Hour),
				EndTime:         now,
				FieldsAvailable: []string{"gpu_util"},
			}, nil
		},
	}

	handler := NewGPUHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus/gpu-001/telemetry", nil)
	w := httptest.NewRecorder()

	handler.GetGPUTelemetry(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp models.QueryTelemetryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Errorf("failed to decode response: %v", err)
	}

	if resp.Count != 3 {
		t.Errorf("expected 3 records, got %d", resp.Count)
	}
}

// TestNewGPUHandler tests handler creation
func TestNewGPUHandler(t *testing.T) {
	handler := NewGPUHandler(&MockTelemetryService{})

	if handler == nil {
		t.Errorf("expected handler to be created")
	}
}

// TestContentTypeHeaders tests that all responses have correct content type
func TestContentTypeHeaders_ListGPUs(t *testing.T) {
	mockService := &MockTelemetryService{
		ListGPUsFunc: func() (*models.ListGPUsResponse, error) {
			return &models.ListGPUsResponse{GPUs: []string{}, Count: 0}, nil
		},
	}

	handler := NewGPUHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus", nil)
	w := httptest.NewRecorder()

	handler.ListGPUs(w, req)

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected content-type application/json, got %s", ct)
	}
}

// TestContentTypeHeaders_ErrorResponse tests error response content type
func TestContentTypeHeaders_ErrorResponse(t *testing.T) {
	handler := NewGPUHandler(&MockTelemetryService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/gpus", nil)
	w := httptest.NewRecorder()

	handler.ListGPUs(w, req)

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected content-type application/json, got %s", ct)
	}
}

// TestGetGPUTelemetry_EmptyGPUID tests with empty GPU ID in path
func TestGetGPUTelemetry_EmptyGPUID(t *testing.T) {
	handler := NewGPUHandler(&MockTelemetryService{})
	// Request with path that has empty GPU ID: /api/v1/gpus//telemetry
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus//telemetry", nil)
	w := httptest.NewRecorder()

	handler.GetGPUTelemetry(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp models.ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp.Error, "gpu_id is required") {
		t.Errorf("expected 'gpu_id is required' error, got %q", resp.Error)
	}
}

// TestGetGPUTelemetry_InternalServerError tests with internal server error from service
func TestGetGPUTelemetry_InternalServerError(t *testing.T) {
	mockService := &MockTelemetryService{
		GetGPUTelemetryFunc: func(gpuID string, startTime, endTime string) (*models.QueryTelemetryResponse, error) {
			return nil, fmt.Errorf("database connection error")
		},
	}

	handler := NewGPUHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus/gpu-001/telemetry", nil)
	w := httptest.NewRecorder()

	handler.GetGPUTelemetry(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}

	var resp models.ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp.Error, "database connection error") {
		t.Errorf("expected 'database connection error', got %q", resp.Error)
	}
}

// TestExtractGPUIDFromPath tests GPU ID extraction
func TestExtractGPUIDFromPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name:    "valid path",
			path:    "/api/v1/gpus/gpu-001/telemetry",
			want:    "gpu-001",
			wantErr: false,
		},
		{
			name:    "invalid path - too short",
			path:    "/api/v1/gpus",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty gpu id",
			path:    "/api/v1/gpus//telemetry",
			want:    "",
			wantErr: true,
		},
		{
			name:    "gpu id with special chars",
			path:    "/api/v1/gpus/gpu-123-abc/telemetry",
			want:    "gpu-123-abc",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractGPUIDFromPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractGPUIDFromPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractGPUIDFromPath() got = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestDetermineErrorStatus tests error status determination
func TestDetermineErrorStatus(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect int
	}{
		{
			name:   "nil error",
			err:    nil,
			expect: http.StatusOK,
		},
		{
			name:   "invalid format error",
			err:    fmt.Errorf("invalid start_time format"),
			expect: http.StatusBadRequest,
		},
		{
			name:   "no telemetry data error",
			err:    fmt.Errorf("no telemetry data found for gpu_id: gpu-001"),
			expect: http.StatusNotFound,
		},
		{
			name:   "database error",
			err:    fmt.Errorf("database connection error"),
			expect: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineErrorStatus(tt.err)
			if got != tt.expect {
				t.Errorf("determineErrorStatus() got = %d, want %d", got, tt.expect)
			}
		})
	}
}

// TestGetGPUTelemetry_ValidPath tests with valid path and response
func TestGetGPUTelemetry_ValidPath(t *testing.T) {
	now := time.Now()
	mockService := &MockTelemetryService{
		GetGPUTelemetryFunc: func(gpuID string, startTime, endTime string) (*models.QueryTelemetryResponse, error) {
			return &models.QueryTelemetryResponse{
				GPUID: "gpu-001",
				Records: []models.TelemetryResponse{
					{GPUID: "gpu-001", Timestamp: now, Data: map[string]interface{}{"temp": 50}},
				},
				Count:     1,
				StartTime: now,
				EndTime:   now,
			}, nil
		},
	}

	handler := NewGPUHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus/gpu-001/telemetry", nil)
	w := httptest.NewRecorder()

	handler.GetGPUTelemetry(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
