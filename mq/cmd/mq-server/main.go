package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	apimux "gpu-pipeline/mq/api"
	internalmq "gpu-pipeline/mq/internal"
)

var (
	addr   = flag.String("listen", ":8080", "HTTP listen address")
	parts  = flag.Int("partitions", 3, "number of partitions")
	config = flag.String("config", "", "path to JSON config file")
)

// Config represents runtime configuration for the server (can be provided via file).
type Config struct {
	Listen     string `json:"listen"`
	Partitions int    `json:"partitions"`
}

func loadConfig(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func main() {
	flag.Parse()
	// load config if provided via flag or env
	cfgPath := *config
	if cfgPath == "" {
		if ev := os.Getenv("CONFIG_PATH"); ev != "" {
			cfgPath = ev
		}
	}
	if cfgPath != "" {
		if c, err := loadConfig(cfgPath); err == nil {
			if c.Listen != "" {
				*addr = c.Listen
			}
			if c.Partitions > 0 {
				*parts = c.Partitions
			}
		} else {
			log.Printf("warning: failed to load config %s: %v", cfgPath, err)
		}
	}

	// initialize queue with default partition capacity
	q := internalmq.NewQueue(*parts)

	// create a sample default topic for demo
	if err := q.CreateTopic("default", *parts, 0); err != nil {
		log.Printf("warning: create default topic: %v", err)
	}

	// setup API handler and routes
	apiHandler := apimux.NewHandler(q)
	mux := http.NewServeMux()
	apiHandler.RegisterRoutes(mux)

	srv := &http.Server{
		Addr:         *addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("starting server on %s (default partitions=%d)", *addr, *parts)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
