package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
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
	Listen              string                     `json:"listen"`
	Partitions          int                        `json:"partitions"`
	PartitionCapacity   int                        `json:"partition_capacity,omitempty"`
	CompactionEnabled   bool                       `json:"compaction_enabled,omitempty"`
	CompactionInterval  string                     `json:"compaction_interval,omitempty"` // e.g., "5m", "30s"
	CompactionThreshold int                        `json:"compaction_threshold,omitempty"` // message count threshold
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
	var cfg *Config
	partitionCapacity := 0 // default: unlimited capacity, managed by compaction
	if cfgPath != "" {
		if c, err := loadConfig(cfgPath); err == nil {
			cfg = c
			if c.Listen != "" {
				*addr = c.Listen
			}
			if c.Partitions > 0 {
				*parts = c.Partitions
			}
			partitionCapacity = c.PartitionCapacity // 0 = unlimited (default)
		} else {
			log.Printf("warning: failed to load config %s: %v", cfgPath, err)
		}
	}

	// initialize queue with configured partition capacity (0 = unlimited, managed by compaction)
	q := internalmq.NewQueue(partitionCapacity)

	// create a sample default topic for demo
	if err := q.CreateTopic("default", *parts, 0); err != nil {
		log.Printf("warning: create default topic: %v", err)
	}

	// setup automatic compaction scheduler
	compactorCfg := internalmq.DefaultCompactorConfig()
	if cfg != nil {
		if !cfg.CompactionEnabled {
			compactorCfg.Enabled = false
		}
		if cfg.CompactionInterval != "" {
			if dur, err := time.ParseDuration(cfg.CompactionInterval); err == nil {
				compactorCfg.Interval = dur
			} else {
				log.Printf("warning: invalid compaction interval %q: %v", cfg.CompactionInterval, err)
			}
		}
		if cfg.CompactionThreshold > 0 {
			compactorCfg.ThresholdMessages = cfg.CompactionThreshold
		}
	}
	compactor := internalmq.NewCompactor(q, compactorCfg)
	compactor.Start()

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
		capDisplay := "unlimited"
		if partitionCapacity > 0 {
			capDisplay = fmt.Sprintf("%d", partitionCapacity)
		}
		log.Printf("starting server on %s (partitions=%d, partition_capacity=%s, compaction=%v)", *addr, *parts, capDisplay, compactorCfg.Enabled)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")
	compactor.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
