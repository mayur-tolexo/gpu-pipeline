package internal

import (
	"net/http"

	_ "gpu-pipeline/api-gateway/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

// RegisterRoutes registers all API Gateway routes on the given mux.
func (h *GPUHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/gpus", h.ListGPUs)
	mux.HandleFunc("POST /api/v1/telemetry/query", h.QueryTelemetry)
	mux.HandleFunc("GET /api/v1/health", h.Health)
	mux.Handle("/swagger/", httpSwagger.WrapHandler)
}
