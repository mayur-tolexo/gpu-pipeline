package interfaces

import (
	"time"

	"gpu-pipeline/api-gateway/pkg/models"
)

// TelemetryRepository defines the interface for telemetry data access
type TelemetryRepository interface {
	// GetGPUIDs retrieves all unique GPU IDs
	GetGPUIDs() ([]string, error)

	// GetTelemetryByGPUAndTimeRange retrieves telemetry records for a specific GPU within a time range
	GetTelemetryByGPUAndTimeRange(gpuID string, startTime, endTime time.Time) ([]models.TelemetryRecord, error)

	// GetTelemetryByGPU retrieves all telemetry records for a specific GPU
	GetTelemetryByGPU(gpuID string) ([]models.TelemetryRecord, error)
}
