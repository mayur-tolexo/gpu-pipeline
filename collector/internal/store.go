package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Store is a wrapper around *gorm.DB to persist telemetry records.
// Implemented as a singleton to be shared across the collector.
type Store struct {
	db *gorm.DB
}

var (
	storeInstance *Store
	once          sync.Once
	initErr       error
)

// InsertFunc is the function used by the collector to persist records. Tests may
// override this to avoid real DB access.
var InsertFunc = func(record map[string]interface{}) error {
	s := GetStore()
	if s == nil {
		return errors.New("store not initialized")
	}
	return s.insert(record)
}

// validateRecord checks if a record has required fields
func validateRecord(record map[string]interface{}) error {
	gpuID, _ := record["gpu_id"].(string)
	tsStr, _ := record["timestamp"].(string)
	
	if gpuID == "" || tsStr == "" {
		return fmt.Errorf("missing required fields: gpu_id or timestamp")
	}
	
	if _, err := time.Parse(time.RFC3339, tsStr); err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}
	
	return nil
}

// marshallRecord converts a record map to JSON bytes
func marshallRecord(record map[string]interface{}) ([]byte, error) {
	data, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("marshal record: %w", err)
	}
	return data, nil
}

// parseTimestamp parses RFC3339 timestamp string
func parseTimestamp(tsStr string) (time.Time, error) {
	ts, err := time.Parse(time.RFC3339, tsStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp: %w", err)
	}
	return ts, nil
}

// createTelemetry creates a Telemetry model from record
func createTelemetry(record map[string]interface{}, data []byte) (*Telemetry, error) {
	gpuID, _ := record["gpu_id"].(string)
	tsStr, _ := record["timestamp"].(string)
	
	ts, err := parseTimestamp(tsStr)
	if err != nil {
		return nil, err
	}
	
	return &Telemetry{
		GPUID:     gpuID,
		Timestamp: ts,
		Data:      data,
	}, nil
}

// internal insert implementation on *Store
func (s *Store) insert(record map[string]interface{}) error {
	if s == nil || s.db == nil {
		return errors.New("store not initialized")
	}
	
	if err := validateRecord(record); err != nil {
		return err
	}
	
	data, err := marshallRecord(record)
	if err != nil {
		return err
	}
	
	telemetry, err := createTelemetry(record, data)
	if err != nil {
		return err
	}

	// Use upsert (ON CONFLICT DO NOTHING) for idempotency
	result := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(telemetry)
	if result.Error != nil {
		return fmt.Errorf("insert error: %w", result.Error)
	}
	return nil
}

// InitStore initializes the singleton Store using provided DSN.
func InitStore(dsn string) error {
	once.Do(func() {
		initErr = initializeStore(dsn)
	})
	return initErr
}

// initializeStore performs the actual store initialization (extracted for testing)
func initializeStore(dsn string) error {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("gorm open: %w", err)
	}

	// Auto migrate schema
	if err := db.AutoMigrate(&Telemetry{}); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	// Create unique index for idempotency (gpu_id + timestamp)
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS telemetry_gpu_ts_unique ON telemetry (gpu_id, timestamp)`).Error; err != nil {
		return fmt.Errorf("create index: %w", err)
	}

	storeInstance = &Store{db: db}
	return nil
}

// GetStore returns the initialized Store instance. Call InitStore first.
func GetStore() *Store {
	return storeInstance
}

// CloseStore closes the underlying database connection.
func CloseStore() error {
	if storeInstance == nil || storeInstance.db == nil {
		return nil
	}
	sqlDB, err := storeInstance.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

