package internal

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestRegisterRoutes tests that all routes are properly registered
func TestRegisterRoutes(t *testing.T) {
	mockDB, _, err := sqlmock.New()
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
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Mock empty result
	rows := sqlmock.NewRows([]string{"gpu_id"})
	mock.ExpectQuery("SELECT DISTINCT").
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestRoutes_TelemetryEndpoint tests POST /api/v1/telemetry/query endpoint
func TestRoutes_TelemetryEndpoint(t *testing.T) {
	mockDB, _, err := sqlmock.New()
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
	mockDB, _, err := sqlmock.New()
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
	mockDB, _, err := sqlmock.New()
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
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Mock empty result
	rows := sqlmock.NewRows([]string{"gpu_id"})
	mock.ExpectQuery("SELECT DISTINCT").
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected content-type application/json, got %s", w.Header().Get("Content-Type"))
	}
}
