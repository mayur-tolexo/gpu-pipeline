package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"gpu-pipeline/api-gateway/internal"
	"gpu-pipeline/api-gateway/pkg/db"
	"gpu-pipeline/api-gateway/pkg/repository"
	"gpu-pipeline/api-gateway/pkg/service"
)

// @title GPU Pipeline API Gateway
// @version 1.0
// @description Query GPU telemetry data from the central database
// @termsOfService http://swagger.io/terms/
// @host localhost:8000
// @BasePath /

// getEnv retrieves environment variable with a default value
func getEnv(k, d string) string {
	if v, ok := os.LookupEnv(k); ok {
		return v
	}
	return d
}

// getDSN returns the database connection string from flags or environment
func getDSN(dsn string) string {
	if dsn == "" {
		dsn = "user=user password=pass host=localhost port=5432 dbname=telemetry sslmode=disable"
	}
	return dsn
}

// initializeDatabase connects to the database and returns the connection
func initializeDatabase(dsn string) error {
	_, err := db.Connect(dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	log.Printf("Connected to database")
	return nil
}

// setupLayers initializes the dependency injection layers
func setupLayers() (*internal.GPUHandler, error) {
	// Get database connection
	dbConn, err := db.GetConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// Initialize repository layer
	telemetryRepo := repository.NewTelemetryRepository(dbConn)

	// Initialize service layer
	telemetryService := service.NewTelemetryService(telemetryRepo)

	// Create handler
	handler := internal.NewGPUHandler(telemetryService)

	return handler, nil
}

// startServer starts the HTTP server on the specified port with the given handler
func startServer(handler *internal.GPUHandler, port string) error {
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting API Gateway on %s", addr)

	return http.ListenAndServe(addr, mux)
}

func main() {
	dsn := flag.String("dsn", getEnv("DATABASE_URL", ""), "PostgreSQL connection string")
	port := flag.String("port", getEnv("PORT", "8000"), "Port to listen on")
	flag.Parse()

	// Get database connection string
	*dsn = getDSN(*dsn)

	// Initialize database
	if err := initializeDatabase(*dsn); err != nil {
		log.Fatalf("%v", err)
	}
	defer db.Close()

	// Setup dependency injection layers
	handler, err := setupLayers()
	if err != nil {
		log.Fatalf("%v", err)
	}

	// Start server
	if err := startServer(handler, *port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
