package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"gpu-pipeline/api-gateway/pkg/interfaces"
	"gpu-pipeline/api-gateway/pkg/models"
)

// TelemetryService implements the TelemetryService interface
type TelemetryService struct {
	repository interfaces.TelemetryRepository
}

// NewTelemetryService creates a new service instance
func NewTelemetryService(repo interfaces.TelemetryRepository) *TelemetryService {
	return &TelemetryService{repository: repo}
}

// ListGPUs returns a list of all unique GPU IDs
func (s *TelemetryService) ListGPUs() (*models.ListGPUsResponse, error) {
	gpuIDs, err := s.repository.GetGPUIDs()
	if err != nil {
		return nil, fmt.Errorf("failed to list gpus: %w", err)
	}

	return &models.ListGPUsResponse{
		GPUs:  gpuIDs,
		Count: len(gpuIDs),
	}, nil
}

// QueryTelemetry retrieves telemetry data based on the query request
func (s *TelemetryService) QueryTelemetry(req *models.QueryTelemetryRequest) (*models.QueryTelemetryResponse, error) {
	if req.GPUID == "" {
		return nil, fmt.Errorf("gpu_id is required")
	}

	records, err := s.repository.GetTelemetryByGPUAndTimeRange(req.GPUID, req.StartTime, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query telemetry: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no telemetry data found for gpu_id: %s", req.GPUID)
	}

	return s.buildTelemetryResponse(req.GPUID, records, req.StartTime, req.EndTime)
}

// GetGPUTelemetry retrieves telemetry for a specific GPU with optional time range filtering
func (s *TelemetryService) GetGPUTelemetry(gpuID string, startTimeStr, endTimeStr string) (*models.QueryTelemetryResponse, error) {
	if gpuID == "" {
		return nil, fmt.Errorf("gpu_id is required")
	}

	// Parse time parameters
	var startTime, endTime time.Time
	var err error

	if startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid start_time format: %w", err)
		}
	}

	if endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid end_time format: %w", err)
		}
	}

	records, err := s.repository.GetTelemetryByGPUAndTimeRange(gpuID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get gpu telemetry: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no telemetry data found for gpu_id: %s", gpuID)
	}

	return s.buildTelemetryResponse(gpuID, records, startTime, endTime)
}

// buildTelemetryResponse builds the response from telemetry records
func (s *TelemetryService) buildTelemetryResponse(gpuID string, records []models.TelemetryRecord, startTime, endTime time.Time) (*models.QueryTelemetryResponse, error) {
	// Convert records to response format
	telemetryResponses := make([]models.TelemetryResponse, len(records))
	fieldsSet := make(map[string]bool)

	for i, record := range records {
		var data map[string]interface{}
		if err := json.Unmarshal(record.Data, &data); err != nil {
			// If unmarshal fails, use raw data
			data = map[string]interface{}{"raw": string(record.Data)}
		}

		telemetryResponses[i] = models.TelemetryResponse{
			GPUID:     record.GPUID,
			Timestamp: record.Timestamp,
			Data:      data,
		}

		// Track available fields
		for key := range data {
			fieldsSet[key] = true
		}
	}

	// Convert fields set to sorted list
	fieldsAvailable := make([]string, 0, len(fieldsSet))
	for field := range fieldsSet {
		fieldsAvailable = append(fieldsAvailable, field)
	}
	sort.Strings(fieldsAvailable)

	// Set actual time range from results
	if len(records) > 0 {
		if startTime.IsZero() {
			startTime = records[0].Timestamp
		}
		if endTime.IsZero() {
			endTime = records[len(records)-1].Timestamp
		}
	}

	return &models.QueryTelemetryResponse{
		GPUID:           gpuID,
		Records:         telemetryResponses,
		Count:           len(records),
		StartTime:       startTime,
		EndTime:         endTime,
		FieldsAvailable: fieldsAvailable,
	}, nil
}
