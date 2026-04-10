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

	mq "gpu-pipeline/mq"
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

	// simple topic
	topic := mq.NewTopic("default", *parts)
	_ = mq.NewProducer(topic)
	// create a simple HTTP server exposing minimal endpoints
	h := http.NewServeMux()
	h.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); w.Write([]byte("ok")) })
	h.HandleFunc("/produce", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		msg := internalmq.Message{ID: fmt.Sprintf("m-%d", time.Now().UnixNano()), Key: key, Payload: []byte(r.URL.Query().Get("payload")), Timestamp: time.Now()}
		idx, off := topic.Produce(msg)
		w.WriteHeader(200)
		w.Write([]byte(fmt.Sprintf("%d:%d", idx, off)))
	})

	srv := &http.Server{Addr: *addr, Handler: h}

	go func() {
		log.Printf("starting server on %s (partitions=%d)", *addr, *parts)
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
