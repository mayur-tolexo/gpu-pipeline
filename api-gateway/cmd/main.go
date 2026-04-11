package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"gpu-pipeline/api-gateway/internal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// @title GPU Pipeline API Gateway
// @version 1.0
// @description Query GPU telemetry data from the central database
// @termsOfService http://swagger.io/terms/
// @host localhost:8081
// @BasePath /
func main() {
	dsn := flag.String("dsn", os.Getenv("DATABASE_URL"), "PostgreSQL connection string")
	port := flag.String("port", "8081", "Port to listen on")
	flag.Parse()

	if *dsn == "" {
		*dsn = "user=user password=pass host=localhost port=5432 dbname=telemetry sslmode=disable"
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(*dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	log.Printf("Connected to database")

	// Create handler
	handler := internal.NewGPUHandler(db)

	// Register routes
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Start server
	addr := fmt.Sprintf(":%s", *port)
	log.Printf("Starting API Gateway on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
