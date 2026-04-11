package models

import (
	"time"
)

// TelemetryRecord represents a telemetry record from the database
type TelemetryRecord struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	GPUID     string    `json:"gpu_id" gorm:"index"`
	Timestamp time.Time `json:"timestamp" gorm:"index"`
	Data      []byte    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName specifies the table name for Telemetry
func (TelemetryRecord) TableName() string {
	return "telemetry"
}

// ListGPUsResponse contains unique GPU IDs
type ListGPUsResponse struct {
	GPUs  []string `json:"gpus"`
	Count int      `json:"count"`
}

// QueryTelemetryRequest represents query parameters for telemetry data
type QueryTelemetryRequest struct {
	GPUID     string    `json:"gpu_id" binding:"required"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// TelemetryResponse represents a single telemetry entry
type TelemetryResponse struct {
	GPUID     string                 `json:"gpu_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// QueryTelemetryResponse contains telemetry data query results
type QueryTelemetryResponse struct {
	GPUID           string              `json:"gpu_id"`
	Records         []TelemetryResponse `json:"records"`
	Count           int                 `json:"count"`
	StartTime       time.Time           `json:"start_time"`
	EndTime         time.Time           `json:"end_time"`
	FieldsAvailable []string            `json:"fields_available"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}
