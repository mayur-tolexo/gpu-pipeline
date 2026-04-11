package db

import (
	"fmt"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// singleton instance of database connection
var (
	instance *gorm.DB
	once     sync.Once
	mu       sync.RWMutex
)

// Connect initializes the database connection (singleton pattern)
// Should be called once during application startup
func Connect(dsn string) (*gorm.DB, error) {
	var err error
	once.Do(func() {
		instance, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	})
	return instance, err
}

// GetConnection returns the singleton database connection
func GetConnection() (*gorm.DB, error) {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}
	return instance, nil
}

// SetConnection sets the database connection (useful for testing)
func SetConnection(db *gorm.DB) {
	mu.Lock()
	defer mu.Unlock()
	instance = db
}

// Close closes the database connection
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if instance == nil {
		return nil
	}

	sqlDB, err := instance.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

// Reset resets the singleton (useful for testing)
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	instance = nil
	once = sync.Once{}
}
