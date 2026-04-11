package mq

import (
	"log"
	"sync"
	"time"
)

// CompactorConfig defines the configuration for the automatic compaction scheduler.
type CompactorConfig struct {
	// Enabled controls whether automatic compaction is enabled.
	Enabled bool `json:"enabled"`
	// Interval is the duration between compaction runs.
	Interval time.Duration `json:"interval"`
	// ThresholdBytes triggers compaction if any partition exceeds this size.
	// 0 means no size-based trigger (only time-based).
	ThresholdBytes int64 `json:"threshold_bytes,omitempty"`
	// ThresholdMessages triggers compaction if any partition exceeds this message count.
	// 0 means no message-based trigger (only time-based).
	ThresholdMessages int `json:"threshold_messages,omitempty"`
}

// DefaultCompactorConfig returns a sensible default configuration.
// - Enabled: true
// - Interval: 1 minute (suitable for development, adjust for production)
// - ThresholdMessages: 100,000
func DefaultCompactorConfig() CompactorConfig {
	return CompactorConfig{
		Enabled:           true,
		Interval:          1 * time.Minute,
		ThresholdMessages: 100000,
	}
}

// Compactor manages automatic watermark-based compaction for the queue.
// It runs in a background goroutine and periodically triggers compaction based on
// time interval, message count threshold, or size threshold.
type Compactor struct {
	mu       sync.RWMutex
	queue    *Queue
	config   CompactorConfig
	done     chan struct{}
	running  bool
	stopOnce sync.Once
}

// NewCompactor creates a new compactor with the given queue and configuration.
func NewCompactor(q *Queue, cfg CompactorConfig) *Compactor {
	return &Compactor{
		queue:  q,
		config: cfg,
		done:   make(chan struct{}),
		running: false,
	}
}

// Start begins the background compaction routine. Safe to call multiple times.
func (c *Compactor) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.running {
		return // already running
	}
	
	if !c.config.Enabled {
		log.Printf("compaction is disabled in configuration")
		return
	}
	
	c.running = true
	go c.compactionLoop()
	log.Printf("compaction scheduler started (interval: %v)", c.config.Interval)
}

// Stop gracefully stops the background compaction routine. Safe to call multiple times.
func (c *Compactor) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	c.mu.Unlock()
	
	c.stopOnce.Do(func() {
		close(c.done)
		log.Printf("compaction scheduler stopped")
	})
}

// IsRunning returns whether the compactor is currently running.
func (c *Compactor) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// UpdateConfig updates the compactor configuration. Stop and restart if needed.
func (c *Compactor) UpdateConfig(cfg CompactorConfig) {
	c.mu.Lock()
	wasRunning := c.running
	c.running = false
	c.config = cfg
	c.mu.Unlock()
	
	// Stop the current instance if running
	if wasRunning {
		c.stopOnce.Do(func() {
			close(c.done)
		})
	}
	
	// Restart if new config is enabled
	if cfg.Enabled {
		// Create a new done channel for restart
		c.done = make(chan struct{})
		c.stopOnce = sync.Once{}
		c.Start()
	}
}

// GetConfig returns a copy of the current configuration.
func (c *Compactor) GetConfig() CompactorConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// compactionLoop is the main loop that runs in a background goroutine.
// It checks for compaction triggers (time, size, message count) and executes compaction.
func (c *Compactor) compactionLoop() {
	ticker := time.NewTicker(c.config.Interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.mu.RLock()
			shouldCompact := c.running && c.config.Enabled
			c.mu.RUnlock()
			
			if shouldCompact {
				c.checkAndCompact()
			}
		}
	}
}

// checkAndCompact checks all topics and partitions for compaction triggers.
// It performs compaction if any trigger threshold is exceeded.
func (c *Compactor) checkAndCompact() {
	topics := c.queue.ListTopics()
	if len(topics) == 0 {
		log.Printf("🔄 [COMPACTOR] CHECK: no topics to compact")
		return
	}
	
	log.Printf("🔄 [COMPACTOR] CHECKING: %d topics for compaction triggers", len(topics))
	
	compactionsPerformed := 0
	for _, topicName := range topics {
		if shouldTrigger := c.checkTopicTriggers(topicName); shouldTrigger {
			log.Printf("🔄 [COMPACTOR] TRIGGER DETECTED: topic=%s, performing compaction", topicName)
			c.compactTopicWithLogging(topicName)
			compactionsPerformed++
		}
	}
	
	if compactionsPerformed == 0 {
		log.Printf("🔄 [COMPACTOR] CHECK COMPLETE: no thresholds exceeded")
	}
}

// checkTopicTriggers checks if any partition in a topic exceeds compaction thresholds.
func (c *Compactor) checkTopicTriggers(topicName string) bool {
	topic, err := c.queue.GetTopic(topicName)
	if err != nil {
		log.Printf("❌ [COMPACTOR] TRIGGER_CHECK FAILED: topic not found (topic=%s, error=%v)", topicName, err)
		return false
	}
	
	c.mu.RLock()
	thresholdMessages := c.config.ThresholdMessages
	c.mu.RUnlock()
	
	// Check each partition for trigger thresholds
	for i := 0; i < topic.Partitions(); i++ {
		stats, err := c.queue.GetPartitionStats(topicName, i)
		if err != nil {
			log.Printf("⚠️  [COMPACTOR] TRIGGER_CHECK: failed to get stats (topic=%s, partition=%d, error=%v)", 
				topicName, i, err)
			continue
		}
		
		// Check message count threshold
		if thresholdMessages > 0 {
			msgCount := stats["message_count"].(int)
			if msgCount > thresholdMessages {
				log.Printf("⚠️  [COMPACTOR] THRESHOLD_EXCEEDED: message count (topic=%s, partition=%d, count=%d, threshold=%d)", 
					topicName, i, msgCount, thresholdMessages)
				return true
			}
		}
		
		// Check compactable message count (if close to being able to free significant space)
		compactableCount := stats["compactable_count"].(int64)
		if compactableCount > int64(thresholdMessages/2) {
			log.Printf("⚠️  [COMPACTOR] THRESHOLD_EXCEEDED: compactable messages (topic=%s, partition=%d, compactable=%d, threshold=%d)", 
				topicName, i, compactableCount, thresholdMessages/2)
			return true
		}
	}
	
	return false
}

// compactTopicWithLogging compacts a topic and logs the results.
func (c *Compactor) compactTopicWithLogging(topicName string) {
	log.Printf("🗑️  [COMPACTOR] STARTING: topic=%s", topicName)
	
	results, err := c.queue.CompactTopic(topicName)
	if err != nil {
		log.Printf("❌ [COMPACTOR] ERROR: compaction failed (topic=%s, error=%v)", topicName, err)
		return
	}
	
	totalCompacted := int64(0)
	for partID, count := range results {
		totalCompacted += count
		if count > 0 {
			log.Printf("   📊 [COMPACTOR] PARTITION_RESULT: partition=%s, messages_removed=%d", partID, count)
		}
	}
	
	if totalCompacted > 0 {
		log.Printf("✅ [COMPACTOR] COMPLETED: topic=%s, total_messages_removed=%d, partitions_compacted=%d", 
			topicName, totalCompacted, len(results))
	} else {
		log.Printf("🔄 [COMPACTOR] COMPLETED: topic=%s, no messages to compact", topicName)
	}
}
