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

// internal insert implementation on *Store
func (s *Store) insert(record map[string]interface{}) error {
	if s == nil || s.db == nil {
		return errors.New("store not initialized")
	}
	gpuID, _ := record["gpu_id"].(string)
	tsStr, _ := record["timestamp"].(string)
	if gpuID == "" || tsStr == "" {
		return fmt.Errorf("missing required fields: gpu_id or timestamp")
	}
	ts, err := time.Parse(time.RFC3339, tsStr)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}

	t := Telemetry{
		GPUID:     gpuID,
		Timestamp: ts,
		Data:      data,
	}

	// Use upsert (ON CONFLICT DO NOTHING) for idempotency
	result := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&t)
	if result.Error != nil {
		return fmt.Errorf("insert error: %w", result.Error)
	}
	return nil
}

// InitStore initializes the singleton Store using provided DSN.
func InitStore(dsn string) error {
	once.Do(func() {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			initErr = fmt.Errorf("gorm open: %w", err)
			return
		}

		// Auto migrate schema
		if err := db.AutoMigrate(&Telemetry{}); err != nil {
			initErr = fmt.Errorf("auto migrate: %w", err)
			return
		}

		// Create unique index for idempotency (gpu_id + timestamp)
		if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS telemetry_gpu_ts_unique ON telemetry (gpu_id, timestamp)`).Error; err != nil {
			initErr = fmt.Errorf("create index: %w", err)
			return
		}

		storeInstance = &Store{db: db}
	})
	return initErr
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
