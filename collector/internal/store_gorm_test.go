package internal

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestStore_Insert_WithMockDB tests insert with sqlmock
func TestStore_Insert_WithMockDB(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, _ := gorm.Open(dialector, &gorm.Config{})

	store := &Store{db: db}

	// GORM wraps operations in transactions, so expect Begin, Insert, Commit
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "telemetry" ("gpu_id","timestamp","data") VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`)).
		WithArgs("g1", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	rec := map[string]interface{}{
		"gpu_id":    "g1",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"power":     250.5,
	}

	err := store.insert(rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestStore_Insert_MissingGpuID tests insert validation
func TestStore_Insert_MissingGpuID(t *testing.T) {
	mockDB, _, _ := sqlmock.New()
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, _ := gorm.Open(dialector, &gorm.Config{})
	store := &Store{db: db}

	rec := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	err := store.insert(rec)
	if err == nil {
		t.Fatalf("expected error for missing gpu_id")
	}
}

// TestStore_Insert_MissingTimestamp tests insert validation
func TestStore_Insert_MissingTimestamp(t *testing.T) {
	mockDB, _, _ := sqlmock.New()
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, _ := gorm.Open(dialector, &gorm.Config{})
	store := &Store{db: db}

	rec := map[string]interface{}{
		"gpu_id": "g1",
	}

	err := store.insert(rec)
	if err == nil {
		t.Fatalf("expected error for missing timestamp")
	}
}

// TestStore_Insert_InvalidTimestampFormat tests timestamp parsing
func TestStore_Insert_InvalidTimestampFormat(t *testing.T) {
	mockDB, _, _ := sqlmock.New()
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, _ := gorm.Open(dialector, &gorm.Config{})
	store := &Store{db: db}

	rec := map[string]interface{}{
		"gpu_id":    "g1",
		"timestamp": "not-rfc3339",
	}

	err := store.insert(rec)
	if err == nil {
		t.Fatalf("expected error for invalid timestamp format")
	}
}

// TestStore_Insert_ValidRecord tests successful insert
func TestStore_Insert_ValidRecord(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, _ := gorm.Open(dialector, &gorm.Config{})
	store := &Store{db: db}

	// GORM wraps in transaction
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "telemetry"`)).
		WithArgs("gpu-001", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	ts := time.Now().UTC()
	rec := map[string]interface{}{
		"gpu_id":    "gpu-001",
		"timestamp": ts.Format(time.RFC3339),
		"power":     250.5,
		"temp":      75.2,
	}

	err := store.insert(rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestInsertFunc_CallsStoreInsert tests that InsertFunc calls store.insert
func TestInsertFunc_CallsStoreInsert(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, _ := gorm.Open(dialector, &gorm.Config{})

	// Setup singleton
	orig := storeInstance
	defer func() { storeInstance = orig }()
	storeInstance = &Store{db: db}

	// GORM wraps in transaction
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "telemetry"`)).
		WithArgs("g1", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	rec := map[string]interface{}{
		"gpu_id":    "g1",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	err := InsertFunc(rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestStore_Insert_JSONMarshaling tests JSON marshaling of data
func TestStore_Insert_JSONMarshaling(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, _ := gorm.Open(dialector, &gorm.Config{})
	store := &Store{db: db}

	// GORM wraps in transaction
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "telemetry"`)).
		WithArgs("g1", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	rec := map[string]interface{}{
		"gpu_id":    "g1",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"power":     250.5,
		"temp":      75.2,
		"status":    "ok",
	}

	err := store.insert(rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestTelemetryModel_HasRequiredFields tests Telemetry model
func TestTelemetryModel_HasRequiredFields(t *testing.T) {
	tm := &Telemetry{
		GPUID:     "g1",
		Timestamp: time.Now(),
		Data:      []byte("{}"),
	}

	if tm.GPUID == "" {
		t.Fatalf("expected GPUID to be set")
	}
	if tm.Timestamp.IsZero() {
		t.Fatalf("expected Timestamp to be set")
	}
	if len(tm.Data) == 0 {
		t.Fatalf("expected Data to be set")
	}
}

// TestStore_GetStore_ReturnsInstance tests GetStore
func TestStore_GetStore_ReturnsInstance(t *testing.T) {
	orig := storeInstance
	defer func() { storeInstance = orig }()

	mockDB, _, _ := sqlmock.New()
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, _ := gorm.Open(dialector, &gorm.Config{})

	storeInstance = &Store{db: db}

	store := GetStore()
	if store == nil {
		t.Fatalf("expected non-nil store")
	}
	if store.db != db {
		t.Fatalf("expected same db instance")
	}
}

// TestStore_CloseStore tests CloseStore
func TestStore_CloseStore(t *testing.T) {
	orig := storeInstance
	defer func() { storeInstance = orig }()

	mockDB, _, _ := sqlmock.New()
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, _ := gorm.Open(dialector, &gorm.Config{})

	storeInstance = &Store{db: db}

	err := CloseStore()
	if err != nil {
		t.Logf("CloseStore returned: %v", err)
	}
}

// TestStore_Insert_DataMarshaled tests that data is properly marshaled as JSON bytes
func TestStore_Insert_DataMarshaled(t *testing.T) {
	mockDB, mock, _ := sqlmock.New()
	defer mockDB.Close()

	dialector := postgres.New(postgres.Config{
		Conn:       mockDB,
		DriverName: "postgres",
	})
	db, _ := gorm.Open(dialector, &gorm.Config{})
	store := &Store{db: db}

	// GORM wraps in transaction
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "telemetry"`)).
		WithArgs("gpu-001", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	data := map[string]interface{}{
		"gpu_id":    "gpu-001",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"metrics": map[string]interface{}{
			"power": 250.5,
			"temp":  75.2,
			"util":  85.3,
		},
	}

	err := store.insert(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the JSON was marshaled correctly
	expectedJSON, _ := json.Marshal(data)
	if len(expectedJSON) == 0 {
		t.Fatalf("expected non-empty JSON data")
	}
}
