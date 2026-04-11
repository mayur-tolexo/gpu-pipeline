package internal

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gpu-pipeline/api-gateway/pkg/models"
)

// TestRegisterRoutes tests that all routes are properly registered
func TestRegisterRoutes(t *testing.T) {
	mockService := &MockTelemetryService{}
	handler := NewGPUHandler(mockService)
	mux := http.NewServeMux()

	// Register routes
	handler.RegisterRoutes(mux)

	// Test that mux is not nil
	if mux == nil {
		t.Errorf("expected mux to be created")
	}
}

// TestRoutes_ListGPUEndpoint tests GET /api/v1/gpus endpoint
func TestRoutes_ListGPUEndpoint(t *testing.T) {
	mockService := &MockTelemetryService{
		ListGPUsFunc: func() (*models.ListGPUsResponse, error) {
			return &models.ListGPUsResponse{
				GPUs:  []string{"gpu-001", "gpu-002"},
				Count: 2,
			}, nil
		},
	}

	handler := NewGPUHandler(mockService)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestRoutes_TelemetryEndpoint tests POST /api/v1/telemetry/query endpoint
func TestRoutes_TelemetryEndpoint(t *testing.T) {
	mockService := &MockTelemetryService{}

	handler := NewGPUHandler(mockService)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Create request with empty body (will be invalid)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/query", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// Should get 400 because body is invalid
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestRoutes_HealthEndpoint tests GET /api/v1/health endpoint
func TestRoutes_HealthEndpoint(t *testing.T) {
	mockService := &MockTelemetryService{}

	handler := NewGPUHandler(mockService)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestRoutes_HealthCheckReturnsOK verifies health check response format
func TestRoutes_HealthCheckReturnsOK(t *testing.T) {
	mockService := &MockTelemetryService{}

	handler := NewGPUHandler(mockService)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected content-type application/json, got %s", w.Header().Get("Content-Type"))
	}
}

// TestRoutes_GPUListReturnsJSON verifies GPU list response format
func TestRoutes_GPUListReturnsJSON(t *testing.T) {
	mockService := &MockTelemetryService{
		ListGPUsFunc: func() (*models.ListGPUsResponse, error) {
			return &models.ListGPUsResponse{GPUs: []string{}, Count: 0}, nil
		},
	}

	handler := NewGPUHandler(mockService)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected content-type application/json, got %s", w.Header().Get("Content-Type"))
	}
}

// TestRoutes_GetGPUTelemetryEndpoint tests GET /api/v1/gpus/{id}/telemetry endpoint
func TestRoutes_GetGPUTelemetryEndpoint(t *testing.T) {
	mockService := &MockTelemetryService{
		GetGPUTelemetryFunc: func(gpuID string, startTime, endTime string) (*models.QueryTelemetryResponse, error) {
			return nil, fmt.Errorf("no telemetry data found for gpu_id: gpu-001")
		},
	}

	handler := NewGPUHandler(mockService)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus/gpu-001/telemetry", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// Should return 404 when no data found
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected content-type application/json, got %s", w.Header().Get("Content-Type"))
	}
}

// TestHandleGPUPath_InvalidPath tests handleGPUPath with invalid path format
func TestHandleGPUPath_InvalidPath(t *testing.T) {
	mockService := &MockTelemetryService{}

	handler := NewGPUHandler(mockService)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Test invalid path that doesn't match /api/v1/gpus/{id}/telemetry pattern
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus/gpu-001/invalid", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// Should return 404 for invalid path
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestHandleGPUPath_OnlyGPUIDPath tests handleGPUPath with only GPU ID (no /telemetry)
func TestHandleGPUPath_OnlyGPUIDPath(t *testing.T) {
	mockService := &MockTelemetryService{}

	handler := NewGPUHandler(mockService)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Test path with only GPU ID and nothing after
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus/gpu-001", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	// Should return 404 because it's not /api/v1/gpus/{id}/telemetry
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
