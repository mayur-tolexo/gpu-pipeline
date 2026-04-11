package internal

import (
	"net/http"
	"strings"

	_ "gpu-pipeline/api-gateway/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

// RegisterRoutes registers all API Gateway routes on the given mux.
func (h *GPUHandler) RegisterRoutes(mux *http.ServeMux) {
	// Health and GPUs list
	mux.HandleFunc("GET /api/v1/health", h.Health)
	mux.HandleFunc("GET /api/v1/gpus", h.ListGPUs)

	// Telemetry endpoints - handle /api/v1/gpus/{id}/telemetry first (more specific)
	mux.HandleFunc("GET /api/v1/gpus/", h.handleGPUPath)

	// Query telemetry via POST
	mux.HandleFunc("POST /api/v1/telemetry/query", h.QueryTelemetry)

	// Swagger UI
	mux.Handle("/swagger/", httpSwagger.WrapHandler)
}

// handleGPUPath routes requests to /api/v1/gpus/{id}/telemetry
func (h *GPUHandler) handleGPUPath(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// Path format: /api/v1/gpus/{id}/telemetry

	pathParts := strings.Split(strings.TrimPrefix(path, "/api/v1/gpus/"), "/")
	if len(pathParts) >= 2 && pathParts[1] == "telemetry" {
		// This is /api/v1/gpus/{id}/telemetry
		h.GetGPUTelemetry(w, r)
	} else {
		http.Error(w, "not found", http.StatusNotFound)
	}
}
