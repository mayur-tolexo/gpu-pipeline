package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gpu-pipeline/api-gateway/pkg/interfaces"
	"gpu-pipeline/api-gateway/pkg/models"
)

// GPUHandler handles HTTP requests for GPU telemetry data
type GPUHandler struct {
	service interfaces.TelemetryService
}

// NewGPUHandler creates a new GPU handler
func NewGPUHandler(service interfaces.TelemetryService) *GPUHandler {
	return &GPUHandler{service: service}
}

// sendJSONError sends a JSON error response
func sendJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(models.ErrorResponse{Error: message})
}

// ListGPUs godoc
// @Summary List all unique GPUs
// @Description Get a list of all unique GPU IDs that have telemetry data
// @Tags GPUs
// @Produce json
// @Success 200 {object} models.ListGPUsResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/gpus [get]
func (h *GPUHandler) ListGPUs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	response, err := h.service.ListGPUs()
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetGPUTelemetry godoc
// @Summary Get telemetry for a specific GPU
// @Description Get all telemetry entries for a specific GPU, ordered by time
// @Description Supports optional time window filters with start_time and end_time query parameters (RFC3339 format)
// @Tags Telemetry
// @Produce json
// @Param id path string true "GPU ID" example(gpu-001)
// @Param start_time query string false "Start time (RFC3339 format)" example(2026-04-11T00:00:00Z)
// @Param end_time query string false "End time (RFC3339 format)" example(2026-04-12T23:59:59Z)
// @Success 200 {object} models.QueryTelemetryResponse "Telemetry data for GPU"
// @Failure 400 {object} models.ErrorResponse "Invalid time format or missing GPU ID"
// @Failure 404 {object} models.ErrorResponse "GPU not found"
// @Failure 500 {object} models.ErrorResponse "Database query error"
// @Router /api/v1/gpus/{id}/telemetry [get]
// extractGPUIDFromPath extracts GPU ID from the URL path
func extractGPUIDFromPath(path string) (string, error) {
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 5 {
		return "", fmt.Errorf("invalid path")
	}
	gpuID := pathParts[4]
	if gpuID == "" {
		return "", fmt.Errorf("gpu_id is required")
	}
	return gpuID, nil
}

// determineErrorStatus determines the appropriate HTTP status code based on error message
func determineErrorStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "invalid") && strings.Contains(errMsg, "format") {
		return http.StatusBadRequest
	}
	if strings.Contains(errMsg, "no telemetry data found") {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

// GetGPUTelemetry godoc
// @Summary Get telemetry for a specific GPU
// @Description Get all telemetry entries for a specific GPU, ordered by time
// @Description Supports optional time window filters with start_time and end_time query parameters (RFC3339 format)
// @Tags Telemetry
// @Produce json
// @Param id path string true "GPU ID" example(gpu-001)
// @Param start_time query string false "Start time (RFC3339 format)" example(2026-04-11T00:00:00Z)
// @Param end_time query string false "End time (RFC3339 format)" example(2026-04-12T23:59:59Z)
// @Success 200 {object} models.QueryTelemetryResponse "Telemetry data for GPU"
// @Failure 400 {object} models.ErrorResponse "Invalid time format or missing GPU ID"
// @Failure 404 {object} models.ErrorResponse "GPU not found"
// @Failure 500 {object} models.ErrorResponse "Database query error"
// @Router /api/v1/gpus/{id}/telemetry [get]
func (h *GPUHandler) GetGPUTelemetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract GPU ID from path
	gpuID, err := extractGPUIDFromPath(r.URL.Path)
	if err != nil {
		sendJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Parse query parameters for time range
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")

	response, err := h.service.GetGPUTelemetry(gpuID, startTimeStr, endTimeStr)
	if err != nil {
		statusCode := determineErrorStatus(err)
		sendJSONError(w, statusCode, err.Error())
		return
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
		sendJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
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
