package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"
)

// GPUHandler handles requests related to GPU data
type GPUHandler struct {
	db *gorm.DB
}

// NewGPUHandler creates a new GPU handler
func NewGPUHandler(db *gorm.DB) *GPUHandler {
	return &GPUHandler{db: db}
}

// TelemetryRecord represents a telemetry record from the database
type TelemetryRecord struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	GPUID     string    `json:"gpu_id"`
	Timestamp time.Time `json:"timestamp"`
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

// ListGPUs godoc
// @Summary List all unique GPUs
// @Description Get a list of all unique GPU IDs that have telemetry data
// @Tags GPUs
// @Produce json
// @Success 200 {object} ListGPUsResponse
// @Failure 500 {string} string
// @Router /api/v1/gpus [get]
func (h *GPUHandler) ListGPUs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var gpuIDs []string
	result := h.db.Model(&TelemetryRecord{}).
		Distinct("gpu_id").
		Order("gpu_id ASC").
		Pluck("gpu_id", &gpuIDs)

	if result.Error != nil {
		http.Error(w, fmt.Sprintf("query error: %v", result.Error), http.StatusInternalServerError)
		return
	}

	if gpuIDs == nil {
		gpuIDs = []string{} // Return empty array instead of null
	}

	response := ListGPUsResponse{
		GPUs:  gpuIDs,
		Count: len(gpuIDs),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// QueryTelemetry godoc
// @Summary Query telemetry data for a GPU
// @Description Get telemetry data for a specific GPU with optional time-range filtering
// @Tags Telemetry
// @Accept json
// @Produce json
// @Param request body QueryTelemetryRequest true "Query parameters"
// @Success 200 {object} QueryTelemetryResponse
// @Failure 400 {string} string
// @Failure 404 {string} string
// @Failure 500 {string} string
// @Router /api/v1/telemetry/query [post]
func (h *GPUHandler) QueryTelemetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req QueryTelemetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.GPUID == "" {
		http.Error(w, "gpu_id is required", http.StatusBadRequest)
		return
	}

	query := h.db.Where("gpu_id = ?", req.GPUID)

	// Apply time range filters if provided
	if !req.StartTime.IsZero() {
		query = query.Where("timestamp >= ?", req.StartTime)
	}
	if !req.EndTime.IsZero() {
		query = query.Where("timestamp <= ?", req.EndTime)
	}

	var records []TelemetryRecord
	result := query.Order("timestamp DESC").Find(&records)

	if result.Error != nil {
		http.Error(w, fmt.Sprintf("query error: %v", result.Error), http.StatusInternalServerError)
		return
	}

	if result.RowsAffected == 0 {
		http.Error(w, fmt.Sprintf("no telemetry data found for gpu_id: %s", req.GPUID), http.StatusNotFound)
		return
	}

	// Convert records to response format
	telemetryResponses := make([]TelemetryResponse, len(records))
	fieldsSet := make(map[string]bool)

	for i, record := range records {
		var data map[string]interface{}
		if err := json.Unmarshal(record.Data, &data); err != nil {
			// If unmarshal fails, use raw data
			data = map[string]interface{}{"raw": string(record.Data)}
		}

		telemetryResponses[i] = TelemetryResponse{
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

	// Set default time range if not provided
	startTime := req.StartTime
	endTime := req.EndTime
	if startTime.IsZero() && len(records) > 0 {
		startTime = records[len(records)-1].Timestamp // Oldest record
	}
	if endTime.IsZero() && len(records) > 0 {
		endTime = records[0].Timestamp // Newest record
	}

	response := QueryTelemetryResponse{
		GPUID:           req.GPUID,
		Records:         telemetryResponses,
		Count:           len(records),
		StartTime:       startTime,
		EndTime:         endTime,
		FieldsAvailable: fieldsAvailable,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Health godoc
// @Summary Health check
// @Description Check if the API gateway is running
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/v1/health [get]
func (h *GPUHandler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
