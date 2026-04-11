package repository_test

import (
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"gpu-pipeline/api-gateway/pkg/repository"
)

// TestGetGPUIDs tests retrieving unique GPU IDs
func TestGetGPUIDs(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatalf("failed to setup mock db: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := repository.NewTelemetryRepository(db)

	// Test successful retrieval
	rows := sqlmock.NewRows([]string{"gpu_id"}).
		AddRow("gpu-001").
		AddRow("gpu-002").
		AddRow("gpu-003")

	mock.ExpectQuery(`SELECT DISTINCT gpu_id FROM "telemetry" ORDER BY gpu_id ASC`).
		WillReturnRows(rows)

	gpuIDs, err := repo.GetGPUIDs()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(gpuIDs) != 3 {
		t.Errorf("expected 3 GPUs, got %d", len(gpuIDs))
	}

	if gpuIDs[0] != "gpu-001" {
		t.Errorf("expected first GPU to be gpu-001, got %s", gpuIDs[0])
	}
}

// TestGetTelemetryByGPU tests retrieving telemetry for a specific GPU
func TestGetTelemetryByGPU(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatalf("failed to setup mock db: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := repository.NewTelemetryRepository(db)

	// Test successful retrieval
	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "gpu_id", "timestamp", "data", "created_at"}).
		AddRow(1, "gpu-001", now, []byte(`{"temp": 45}`), now).
		AddRow(2, "gpu-001", now.Add(1*time.Second), []byte(`{"temp": 46}`), now.Add(1*time.Second))

	mock.ExpectQuery(`SELECT \* FROM "telemetry" WHERE gpu_id = \$1 ORDER BY timestamp ASC`).
		WithArgs("gpu-001").
		WillReturnRows(rows)

	records, err := repo.GetTelemetryByGPU("gpu-001")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
}

// TestGetTelemetryByGPUAndTimeRange tests retrieving telemetry with time range
func TestGetTelemetryByGPUAndTimeRange(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatalf("failed to setup mock db: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := repository.NewTelemetryRepository(db)

	// Test with time range
	now := time.Now()
	startTime := now
	endTime := now.Add(2 * time.Second)

	rows := sqlmock.NewRows([]string{"id", "gpu_id", "timestamp", "data", "created_at"}).
		AddRow(1, "gpu-001", now.Add(1*time.Second), []byte(`{"temp": 45}`), now)

	mock.ExpectQuery(`SELECT \* FROM "telemetry" WHERE gpu_id = \$1 AND timestamp >= \$2 AND timestamp <= \$3 ORDER BY timestamp ASC`).
		WithArgs("gpu-001", startTime, endTime).
		WillReturnRows(rows)

	records, err := repo.GetTelemetryByGPUAndTimeRange("gpu-001", startTime, endTime)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
}

// TestGetTelemetryByGPU_EmptyResult tests retrieving telemetry with no results
func TestGetTelemetryByGPU_EmptyResult(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatalf("failed to setup mock db: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := repository.NewTelemetryRepository(db)

	// Test with no results
	rows := sqlmock.NewRows([]string{"id", "gpu_id", "timestamp", "data", "created_at"})

	mock.ExpectQuery(`SELECT \* FROM "telemetry" WHERE gpu_id = \$1 ORDER BY timestamp ASC`).
		WithArgs("gpu-invalid").
		WillReturnRows(rows)

	records, err := repo.GetTelemetryByGPU("gpu-invalid")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

// TestGetTelemetryByGPU_NilDB tests error handling with nil database
func TestGetTelemetryByGPU_NilDB(t *testing.T) {
	repo := repository.NewTelemetryRepository(nil)

	records, err := repo.GetTelemetryByGPU("gpu-001")
	if err == nil {
		t.Errorf("expected error with nil database, got nil")
	}

	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

// TestGetGPUIDs_NilDB tests error handling when database is nil
func TestGetGPUIDs_NilDB(t *testing.T) {
	repo := repository.NewTelemetryRepository(nil)

	gpuIDs, err := repo.GetGPUIDs()
	if err == nil {
		t.Errorf("expected error with nil database, got nil")
	}

	if len(gpuIDs) != 0 {
		t.Errorf("expected 0 GPUs, got %d", len(gpuIDs))
	}
}

// TestGetGPUIDs_EmptyResult tests retrieving GPU IDs with no results
func TestGetGPUIDs_EmptyResult(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatalf("failed to setup mock db: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := repository.NewTelemetryRepository(db)

	// Test with no results
	rows := sqlmock.NewRows([]string{"gpu_id"})

	mock.ExpectQuery(`SELECT DISTINCT gpu_id FROM "telemetry" ORDER BY gpu_id ASC`).
		WillReturnRows(rows)

	gpuIDs, err := repo.GetGPUIDs()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(gpuIDs) != 0 {
		t.Errorf("expected 0 GPUs, got %d", len(gpuIDs))
	}
}

// TestGetTelemetryByGPUAndTimeRange_NilDB tests error handling with nil database
func TestGetTelemetryByGPUAndTimeRange_NilDB(t *testing.T) {
	repo := repository.NewTelemetryRepository(nil)
	now := time.Now()

	records, err := repo.GetTelemetryByGPUAndTimeRange("gpu-001", now, now.Add(1*time.Hour))
	if err == nil {
		t.Errorf("expected error with nil database, got nil")
	}

	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

// TestGetTelemetryByGPUAndTimeRange_EmptyResult tests retrieving telemetry with no results
func TestGetTelemetryByGPUAndTimeRange_EmptyResult(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatalf("failed to setup mock db: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := repository.NewTelemetryRepository(db)

	// Test with no results in time range
	now := time.Now()
	startTime := now
	endTime := now.Add(2 * time.Second)

	rows := sqlmock.NewRows([]string{"id", "gpu_id", "timestamp", "data", "created_at"})

	mock.ExpectQuery(`SELECT \* FROM "telemetry" WHERE gpu_id = \$1 AND timestamp >= \$2 AND timestamp <= \$3 ORDER BY timestamp ASC`).
		WithArgs("gpu-invalid", startTime, endTime).
		WillReturnRows(rows)

	records, err := repo.GetTelemetryByGPUAndTimeRange("gpu-invalid", startTime, endTime)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

// TestGetGPUIDs_QueryError tests GetGPUIDs with query error
func TestGetGPUIDs_QueryError(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatalf("failed to setup mock db: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := repository.NewTelemetryRepository(db)

	// Expect query to return error
	mock.ExpectQuery(`SELECT DISTINCT gpu_id FROM "telemetry" ORDER BY gpu_id ASC`).
		WillReturnError(gorm.ErrInvalidDB)

	gpuIDs, err := repo.GetGPUIDs()
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if gpuIDs != nil {
		t.Errorf("expected nil gpuIDs, got %v", gpuIDs)
	}
}

// TestGetTelemetryByGPU_QueryError tests GetTelemetryByGPU with query error
func TestGetTelemetryByGPU_QueryError(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatalf("failed to setup mock db: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := repository.NewTelemetryRepository(db)

	// Expect query to return error
	mock.ExpectQuery(`SELECT \* FROM "telemetry" WHERE gpu_id = \$1 ORDER BY timestamp ASC`).
		WithArgs("gpu-001").
		WillReturnError(gorm.ErrInvalidDB)

	records, err := repo.GetTelemetryByGPU("gpu-001")
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if records != nil {
		t.Errorf("expected nil records, got %v", records)
	}
}

// TestGetTelemetryByGPUAndTimeRange_QueryError tests GetTelemetryByGPUAndTimeRange with query error
func TestGetTelemetryByGPUAndTimeRange_QueryError(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatalf("failed to setup mock db: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := repository.NewTelemetryRepository(db)
	now := time.Now()
	startTime := now
	endTime := now.Add(2 * time.Second)

	// Expect query to return error
	mock.ExpectQuery(`SELECT \* FROM "telemetry" WHERE gpu_id = \$1 AND timestamp >= \$2 AND timestamp <= \$3 ORDER BY timestamp ASC`).
		WithArgs("gpu-001", startTime, endTime).
		WillReturnError(gorm.ErrInvalidDB)

	records, err := repo.GetTelemetryByGPUAndTimeRange("gpu-001", startTime, endTime)
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	if records != nil {
		t.Errorf("expected nil records, got %v", records)
	}
}

// TestGetGPUIDs_ReturnsEmptyArray tests that empty results return array not nil
func TestGetGPUIDs_ReturnsEmptyArray(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatalf("failed to setup mock db: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := repository.NewTelemetryRepository(db)

	// Return nil rows to simulate empty result
	rows := sqlmock.NewRows([]string{"gpu_id"})
	mock.ExpectQuery(`SELECT DISTINCT gpu_id FROM "telemetry" ORDER BY gpu_id ASC`).
		WillReturnRows(rows)

	gpuIDs, err := repo.GetGPUIDs()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Should be empty array, not nil
	if gpuIDs == nil {
		t.Errorf("expected empty array, got nil")
	}

	if len(gpuIDs) != 0 {
		t.Errorf("expected 0 GPUs, got %d", len(gpuIDs))
	}
}

// TestGetGPUIDs_ReturnsNonNilOnZeroResults tests nil to empty array conversion
func TestGetGPUIDs_ReturnsNonNilOnZeroResults(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatalf("failed to setup mock db: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := repository.NewTelemetryRepository(db)

	// Return completely empty row set - this can result in nil slice from Pluck
	rows := sqlmock.NewRows([]string{"gpu_id"})

	mock.ExpectQuery(`SELECT DISTINCT gpu_id FROM "telemetry" ORDER BY gpu_id ASC`).
		WillReturnRows(rows)

	gpuIDs, err := repo.GetGPUIDs()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// The key test: result should never be nil, should be empty array
	if gpuIDs == nil {
		t.Errorf("GetGPUIDs should return empty array, not nil")
	}

	if len(gpuIDs) != 0 {
		t.Errorf("expected 0 GPUs in result, got %d", len(gpuIDs))
	}

	// Verify the slice is initialized (not nil)
	if cap(gpuIDs) == 0 && len(gpuIDs) == 0 {
		// Even an initialized empty slice should work
		t.Logf("Result is empty initialized slice: len=%d cap=%d", len(gpuIDs), cap(gpuIDs))
	}
}

// TestGetTelemetryByGPU_EmptyGPUID tests with empty GPU ID
func TestGetTelemetryByGPU_EmptyGPUID(t *testing.T) {
	db, _, err := setupMockDB(t)
	if err != nil {
		t.Fatalf("failed to setup mock db: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := repository.NewTelemetryRepository(db)

	records, err := repo.GetTelemetryByGPU("")
	if err == nil {
		t.Errorf("expected error for empty gpu_id, got nil")
	}

	if records != nil {
		t.Errorf("expected nil records, got %v", records)
	}

	if !strings.Contains(err.Error(), "gpu_id is required") {
		t.Errorf("expected 'gpu_id is required' error, got %q", err.Error())
	}
}

// TestGetTelemetryByGPUAndTimeRange_EmptyGPUID tests with empty GPU ID
func TestGetTelemetryByGPUAndTimeRange_EmptyGPUID(t *testing.T) {
	db, _, err := setupMockDB(t)
	if err != nil {
		t.Fatalf("failed to setup mock db: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	repo := repository.NewTelemetryRepository(db)
	now := time.Now()

	records, err := repo.GetTelemetryByGPUAndTimeRange("", now, now.Add(1*time.Hour))
	if err == nil {
		t.Errorf("expected error for empty gpu_id, got nil")
	}

	if records != nil {
		t.Errorf("expected nil records, got %v", records)
	}

	if !strings.Contains(err.Error(), "gpu_id is required") {
		t.Errorf("expected 'gpu_id is required' error, got %q", err.Error())
	}
}

// setupMockDB creates a mock database connection for testing
func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, error) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, err
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	return gormDB, mock, nil
}
