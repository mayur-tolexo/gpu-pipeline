package interfaces

import (
	"gpu-pipeline/api-gateway/pkg/models"
)

// TelemetryService defines the business logic interface for telemetry operations
type TelemetryService interface {
	// ListGPUs returns a list of all unique GPU IDs
	ListGPUs() (*models.ListGPUsResponse, error)

	// QueryTelemetry retrieves telemetry data based on the query request
	QueryTelemetry(req *models.QueryTelemetryRequest) (*models.QueryTelemetryResponse, error)

	// GetGPUTelemetry retrieves telemetry for a specific GPU with optional time range filtering
	GetGPUTelemetry(gpuID string, startTime, endTime string) (*models.QueryTelemetryResponse, error)
}
