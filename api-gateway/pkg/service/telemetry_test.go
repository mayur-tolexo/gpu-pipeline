package service_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"gpu-pipeline/api-gateway/pkg/models"
	"gpu-pipeline/api-gateway/pkg/service"
)

// MockTelemetryRepository is a mock implementation of TelemetryRepository
type MockTelemetryRepository struct {
	gpuIDs   []string
	records  []models.TelemetryRecord
	err      error
	errCount int
	callCount int
}

// GetGPUIDs mocks the GetGPUIDs method
func (m *MockTelemetryRepository) GetGPUIDs() ([]string, error) {
	m.callCount++
	if m.err != nil && m.errCount > 0 {
		m.errCount--
		return nil, m.err
	}
	return m.gpuIDs, nil
}

// GetTelemetryByGPU mocks the GetTelemetryByGPU method
func (m *MockTelemetryRepository) GetTelemetryByGPU(gpuID string) ([]models.TelemetryRecord, error) {
	m.callCount++
	if m.err != nil && m.errCount > 0 {
		m.errCount--
		return nil, m.err
	}
	return m.records, nil
}

// GetTelemetryByGPUAndTimeRange mocks the GetTelemetryByGPUAndTimeRange method
func (m *MockTelemetryRepository) GetTelemetryByGPUAndTimeRange(gpuID string, startTime, endTime time.Time) ([]models.TelemetryRecord, error) {
	m.callCount++
	if m.err != nil && m.errCount > 0 {
		m.errCount--
		return nil, m.err
	}
	return m.records, nil
}

// TestListGPUs_Success tests successful GPU list retrieval
func TestListGPUs_Success(t *testing.T) {
	mockRepo := &MockTelemetryRepository{
		gpuIDs: []string{"gpu-001", "gpu-002", "gpu-003"},
	}
	svc := service.NewTelemetryService(mockRepo)

	response, err := svc.ListGPUs()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response.Count != 3 {
		t.Errorf("expected 3 GPUs, got %d", response.Count)
	}

	if len(response.GPUs) != 3 {
		t.Errorf("expected 3 GPU IDs, got %d", len(response.GPUs))
	}

	if response.GPUs[0] != "gpu-001" {
		t.Errorf("expected first GPU to be gpu-001, got %s", response.GPUs[0])
	}
}

// TestListGPUs_EmptyResult tests with no GPUs
func TestListGPUs_EmptyResult(t *testing.T) {
	mockRepo := &MockTelemetryRepository{
		gpuIDs: []string{},
	}
	svc := service.NewTelemetryService(mockRepo)

	response, err := svc.ListGPUs()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response.Count != 0 {
		t.Errorf("expected 0 GPUs, got %d", response.Count)
	}
}

// TestListGPUs_RepositoryError tests error handling
func TestListGPUs_RepositoryError(t *testing.T) {
	mockRepo := &MockTelemetryRepository{
		err:      fmt.Errorf("database error"),
		errCount: 1,
	}
	svc := service.NewTelemetryService(mockRepo)

	response, err := svc.ListGPUs()
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if response != nil {
		t.Errorf("expected nil response on error, got %v", response)
	}
}

// TestQueryTelemetry_Success tests successful telemetry query
func TestQueryTelemetry_Success(t *testing.T) {
	now := time.Now()
	mockRepo := &MockTelemetryRepository{
		records: []models.TelemetryRecord{
			{
				ID:        1,
				GPUID:     "gpu-001",
				Timestamp: now,
				Data:      []byte(`{"temp": 45, "power": 100}`),
			},
		},
	}
	svc := service.NewTelemetryService(mockRepo)

	req := &models.QueryTelemetryRequest{
		GPUID:     "gpu-001",
		StartTime: now.Add(-1 * time.Hour),
		EndTime:   now.Add(1 * time.Hour),
	}

	response, err := svc.QueryTelemetry(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response.Count != 1 {
		t.Errorf("expected 1 record, got %d", response.Count)
	}

	if response.GPUID != "gpu-001" {
		t.Errorf("expected gpu-001, got %s", response.GPUID)
	}
}

// TestQueryTelemetry_NoData tests query with no results
func TestQueryTelemetry_NoData(t *testing.T) {
	mockRepo := &MockTelemetryRepository{
		records: []models.TelemetryRecord{},
	}
	svc := service.NewTelemetryService(mockRepo)

	req := &models.QueryTelemetryRequest{
		GPUID: "gpu-invalid",
	}

	response, err := svc.QueryTelemetry(req)
	if err == nil {
		t.Errorf("expected error for no data, got nil")
	}

	if response != nil {
		t.Errorf("expected nil response on error, got %v", response)
	}
}

// TestQueryTelemetry_InvalidGPUID tests with empty GPU ID
func TestQueryTelemetry_InvalidGPUID(t *testing.T) {
	mockRepo := &MockTelemetryRepository{}
	svc := service.NewTelemetryService(mockRepo)

	req := &models.QueryTelemetryRequest{
		GPUID: "",
	}

	response, err := svc.QueryTelemetry(req)
	if err == nil {
		t.Errorf("expected error for empty gpu_id, got nil")
	}

	if response != nil {
		t.Errorf("expected nil response on error, got %v", response)
	}
}

// TestQueryTelemetry_RepositoryError tests when repository returns error
func TestQueryTelemetry_RepositoryError(t *testing.T) {
	mockRepo := &MockTelemetryRepository{
		err:      fmt.Errorf("database error"),
		errCount: 1,
	}
	svc := service.NewTelemetryService(mockRepo)

	req := &models.QueryTelemetryRequest{
		GPUID: "gpu-001",
	}

	response, err := svc.QueryTelemetry(req)
	if err == nil {
		t.Errorf("expected error from repository, got nil")
	}

	if response != nil {
		t.Errorf("expected nil response on error, got %v", response)
	}

	if !strings.Contains(err.Error(), "failed to query telemetry") {
		t.Errorf("expected 'failed to query telemetry' error, got %q", err.Error())
	}
}

// TestGetGPUTelemetry_Success tests successful GPU telemetry retrieval
func TestGetGPUTelemetry_Success(t *testing.T) {
	now := time.Now()
	mockRepo := &MockTelemetryRepository{
		records: []models.TelemetryRecord{
			{
				ID:        1,
				GPUID:     "gpu-001",
				Timestamp: now,
				Data:      []byte(`{"temp": 45}`),
			},
		},
	}
	svc := service.NewTelemetryService(mockRepo)

	response, err := svc.GetGPUTelemetry("gpu-001", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response.Count != 1 {
		t.Errorf("expected 1 record, got %d", response.Count)
	}
}

// TestGetGPUTelemetry_WithTimeRange tests with RFC3339 time strings
func TestGetGPUTelemetry_WithTimeRange(t *testing.T) {
	now := time.Now()
	mockRepo := &MockTelemetryRepository{
		records: []models.TelemetryRecord{
			{
				ID:        1,
				GPUID:     "gpu-001",
				Timestamp: now,
				Data:      []byte(`{"temp": 45}`),
			},
		},
	}
	svc := service.NewTelemetryService(mockRepo)

	startTimeStr := now.Add(-1 * time.Hour).Format(time.RFC3339)
	endTimeStr := now.Add(1 * time.Hour).Format(time.RFC3339)

	response, err := svc.GetGPUTelemetry("gpu-001", startTimeStr, endTimeStr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response.Count != 1 {
		t.Errorf("expected 1 record, got %d", response.Count)
	}
}

// TestGetGPUTelemetry_InvalidStartTime tests with invalid start_time format
func TestGetGPUTelemetry_InvalidStartTime(t *testing.T) {
	mockRepo := &MockTelemetryRepository{}
	svc := service.NewTelemetryService(mockRepo)

	response, err := svc.GetGPUTelemetry("gpu-001", "invalid-time", "")
	if err == nil {
		t.Errorf("expected error for invalid start_time, got nil")
	}

	if response != nil {
		t.Errorf("expected nil response on error, got %v", response)
	}
}

// TestGetGPUTelemetry_InvalidEndTime tests with invalid end_time format
func TestGetGPUTelemetry_InvalidEndTime(t *testing.T) {
	mockRepo := &MockTelemetryRepository{}
	svc := service.NewTelemetryService(mockRepo)

	now := time.Now().Format(time.RFC3339)
	response, err := svc.GetGPUTelemetry("gpu-001", now, "invalid-time")
	if err == nil {
		t.Errorf("expected error for invalid end_time, got nil")
	}

	if response != nil {
		t.Errorf("expected nil response on error, got %v", response)
	}
}

// TestGetGPUTelemetry_NoGPUID tests with empty GPU ID
func TestGetGPUTelemetry_NoGPUID(t *testing.T) {
	mockRepo := &MockTelemetryRepository{}
	svc := service.NewTelemetryService(mockRepo)

	response, err := svc.GetGPUTelemetry("", "", "")
	if err == nil {
		t.Errorf("expected error for empty gpu_id, got nil")
	}

	if response != nil {
		t.Errorf("expected nil response on error, got %v", response)
	}
}

// TestGetGPUTelemetry_FieldsAvailable tests that available fields are populated
func TestGetGPUTelemetry_FieldsAvailable(t *testing.T) {
	now := time.Now()
	mockRepo := &MockTelemetryRepository{
		records: []models.TelemetryRecord{
			{
				ID:        1,
				GPUID:     "gpu-001",
				Timestamp: now,
				Data:      []byte(`{"temp": 45, "power": 100, "memory": 4096}`),
			},
		},
	}
	svc := service.NewTelemetryService(mockRepo)

	response, err := svc.GetGPUTelemetry("gpu-001", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(response.FieldsAvailable) != 3 {
		t.Errorf("expected 3 fields, got %d", len(response.FieldsAvailable))
	}
}

// TestGetGPUTelemetry_NoData tests with no telemetry data
func TestGetGPUTelemetry_NoData(t *testing.T) {
	mockRepo := &MockTelemetryRepository{
		records: []models.TelemetryRecord{},
	}
	svc := service.NewTelemetryService(mockRepo)

	response, err := svc.GetGPUTelemetry("gpu-001", "", "")
	if err == nil {
		t.Errorf("expected error for no data, got nil")
	}

	if response != nil {
		t.Errorf("expected nil response on error, got %v", response)
	}

	if !strings.Contains(err.Error(), "no telemetry data found") {
		t.Errorf("expected 'no telemetry data found' error, got %q", err.Error())
	}
}

// TestGetGPUTelemetry_RepositoryError tests when repository returns error
func TestGetGPUTelemetry_RepositoryError(t *testing.T) {
	mockRepo := &MockTelemetryRepository{
		err:      fmt.Errorf("database error"),
		errCount: 1,
	}
	svc := service.NewTelemetryService(mockRepo)

	response, err := svc.GetGPUTelemetry("gpu-001", "", "")
	if err == nil {
		t.Errorf("expected error from repository, got nil")
	}

	if response != nil {
		t.Errorf("expected nil response on error, got %v", response)
	}

	if !strings.Contains(err.Error(), "failed to get gpu telemetry") {
		t.Errorf("expected 'failed to get gpu telemetry' error, got %q", err.Error())
	}
}

// TestBuildTelemetryResponse_InvalidJSON tests handling of invalid JSON data
func TestBuildTelemetryResponse_InvalidJSON(t *testing.T) {
	now := time.Now()
	// Create a record with invalid JSON data
	mockRepo := &MockTelemetryRepository{
		records: []models.TelemetryRecord{
			{
				ID:        1,
				GPUID:     "gpu-001",
				Timestamp: now,
				Data:      []byte(`invalid json`),
			},
		},
	}
	svc := service.NewTelemetryService(mockRepo)

	response, err := svc.GetGPUTelemetry("gpu-001", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if response == nil {
		t.Fatalf("expected response, got nil")
	}

	// Should have one record with raw data
	if len(response.Records) != 1 {
		t.Errorf("expected 1 record, got %d", len(response.Records))
	}

	// Check that raw data was captured
	if record := response.Records[0]; record.Data != nil {
		if rawVal, exists := record.Data["raw"]; !exists {
			t.Errorf("expected 'raw' field in data for invalid JSON")
		} else {
			if rawStr, ok := rawVal.(string); !ok || rawStr != "invalid json" {
				t.Errorf("expected raw data to be 'invalid json', got %v", rawVal)
			}
		}
	}
}
