package repository

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"gpu-pipeline/api-gateway/pkg/models"
)

// TelemetryRepository implements the TelemetryRepository interface
type TelemetryRepository struct {
	db *gorm.DB
}

// NewTelemetryRepository creates a new repository instance
func NewTelemetryRepository(db *gorm.DB) *TelemetryRepository {
	return &TelemetryRepository{db: db}
}

// GetGPUIDs retrieves all unique GPU IDs from the database
func (r *TelemetryRepository) GetGPUIDs() ([]string, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	gpuIDs := []string{} // Initialize as empty array to ensure non-nil result
	result := r.db.Model(&models.TelemetryRecord{}).
		Distinct("gpu_id").
		Order("gpu_id ASC").
		Pluck("gpu_id", &gpuIDs)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to query gpu_ids: %w", result.Error)
	}

	return gpuIDs, nil
}

// GetTelemetryByGPU retrieves all telemetry records for a specific GPU
func (r *TelemetryRepository) GetTelemetryByGPU(gpuID string) ([]models.TelemetryRecord, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	if gpuID == "" {
		return nil, fmt.Errorf("gpu_id is required")
	}

	var records []models.TelemetryRecord
	result := r.db.Where("gpu_id = ?", gpuID).
		Order("timestamp ASC").
		Find(&records)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to query telemetry records: %w", result.Error)
	}

	return records, nil
}

// GetTelemetryByGPUAndTimeRange retrieves telemetry records for a specific GPU within a time range
func (r *TelemetryRepository) GetTelemetryByGPUAndTimeRange(gpuID string, startTime, endTime time.Time) ([]models.TelemetryRecord, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	if gpuID == "" {
		return nil, fmt.Errorf("gpu_id is required")
	}

	query := r.db.Where("gpu_id = ?", gpuID)

	// Apply time range filters if provided
	if !startTime.IsZero() {
		query = query.Where("timestamp >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("timestamp <= ?", endTime)
	}

	var records []models.TelemetryRecord
	result := query.Order("timestamp ASC").Find(&records)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to query telemetry records: %w", result.Error)
	}

	return records, nil
}
