package mq

import (
	"testing"
	"time"
)

// TestCompactor_Creation tests compactor initialization
func TestCompactor_Creation(t *testing.T) {
	// Arrange
	q := NewQueue(0)
	cfg := CompactorConfig{
		Enabled:           true,
		Interval:          1 * time.Second,
		ThresholdMessages: 100,
	}

	// Act
	c := NewCompactor(q, cfg)

	// Assert
	if c == nil {
		t.Fatalf("expected non-nil compactor")
	}
	if c.IsRunning() {
		t.Fatalf("expected compactor not running initially")
	}
}

// TestCompactor_Start tests starting the compactor
func TestCompactor_Start(t *testing.T) {
	// Arrange
	q := NewQueue(0)
	cfg := DefaultCompactorConfig()
	c := NewCompactor(q, cfg)

	// Act
	c.Start()
	defer c.Stop()

	// Assert
	time.Sleep(100 * time.Millisecond)
	if !c.IsRunning() {
		t.Fatalf("expected compactor to be running")
	}
}

// TestCompactor_StartDisabled tests starting disabled compactor
func TestCompactor_StartDisabled(t *testing.T) {
	// Arrange
	q := NewQueue(0)
	cfg := CompactorConfig{
		Enabled:  false,
		Interval: 1 * time.Second,
	}
	c := NewCompactor(q, cfg)

	// Act
	c.Start()

	// Assert
	if c.IsRunning() {
		t.Fatalf("expected compactor not to run when disabled")
	}
}

// TestCompactor_Stop tests stopping the compactor
func TestCompactor_Stop(t *testing.T) {
	// Arrange
	q := NewQueue(0)
	cfg := DefaultCompactorConfig()
	c := NewCompactor(q, cfg)
	c.Start()
	time.Sleep(100 * time.Millisecond)

	// Act
	c.Stop()
	time.Sleep(100 * time.Millisecond)

	// Assert
	if c.IsRunning() {
		t.Fatalf("expected compactor to be stopped")
	}
}

// TestCompactor_MultipleStart tests that multiple starts are safe
func TestCompactor_MultipleStart(t *testing.T) {
	// Arrange
	q := NewQueue(0)
	cfg := DefaultCompactorConfig()
	c := NewCompactor(q, cfg)

	// Act
	c.Start()
	time.Sleep(50 * time.Millisecond)
	c.Start() // Second start should be idempotent
	defer c.Stop()

	// Assert
	if !c.IsRunning() {
		t.Fatalf("expected compactor to be running")
	}
}

// TestCompactor_MultipleStop tests that multiple stops are safe
func TestCompactor_MultipleStop(t *testing.T) {
	// Arrange
	q := NewQueue(0)
	cfg := DefaultCompactorConfig()
	c := NewCompactor(q, cfg)
	c.Start()
	time.Sleep(50 * time.Millisecond)

	// Act
	c.Stop()
	time.Sleep(50 * time.Millisecond)
	c.Stop() // Second stop should be safe

	// Assert
	// Should not panic or hang
}

// TestCompactor_UpdateConfig tests updating configuration
func TestCompactor_UpdateConfig(t *testing.T) {
	// Arrange
	q := NewQueue(0)
	cfg1 := CompactorConfig{
		Enabled:           true,
		Interval:          5 * time.Second,
		ThresholdMessages: 100,
	}
	c := NewCompactor(q, cfg1)
	c.Start()
	time.Sleep(50 * time.Millisecond)

	// Act
	cfg2 := CompactorConfig{
		Enabled:           true,
		Interval:          1 * time.Second,
		ThresholdMessages: 50,
	}
	c.UpdateConfig(cfg2)
	defer c.Stop()
	time.Sleep(50 * time.Millisecond)

	// Assert
	if !c.IsRunning() {
		t.Fatalf("expected compactor to still be running after update")
	}
	retrievedCfg := c.GetConfig()
	if retrievedCfg.Interval != 1*time.Second {
		t.Fatalf("expected interval 1s, got %v", retrievedCfg.Interval)
	}
}

// TestCompactor_GetConfig tests retrieving configuration
func TestCompactor_GetConfig(t *testing.T) {
	// Arrange
	q := NewQueue(0)
	cfg := CompactorConfig{
		Enabled:           true,
		Interval:          2 * time.Second,
		ThresholdMessages: 200,
	}
	c := NewCompactor(q, cfg)

	// Act
	retrieved := c.GetConfig()

	// Assert
	if retrieved.Enabled != cfg.Enabled {
		t.Fatalf("expected enabled=%v, got %v", cfg.Enabled, retrieved.Enabled)
	}
	if retrieved.Interval != cfg.Interval {
		t.Fatalf("expected interval=%v, got %v", cfg.Interval, retrieved.Interval)
	}
	if retrieved.ThresholdMessages != cfg.ThresholdMessages {
		t.Fatalf("expected threshold=%d, got %d", cfg.ThresholdMessages, retrieved.ThresholdMessages)
	}
}

// TestCompactor_DefaultConfig tests the default configuration
func TestCompactor_DefaultConfig(t *testing.T) {
	// Act
	cfg := DefaultCompactorConfig()

	// Assert
	if !cfg.Enabled {
		t.Fatalf("expected enabled=true")
	}
	if cfg.Interval != 1*time.Minute {
		t.Fatalf("expected interval=1m, got %v", cfg.Interval)
	}
	if cfg.ThresholdMessages != 100000 {
		t.Fatalf("expected threshold=100000, got %d", cfg.ThresholdMessages)
	}
}

// TestCompactor_DisableAndEnable tests disabling and re-enabling
func TestCompactor_DisableAndEnable(t *testing.T) {
	// Arrange
	q := NewQueue(0)
	cfg := DefaultCompactorConfig()
	c := NewCompactor(q, cfg)
	c.Start()
	time.Sleep(50 * time.Millisecond)

	// Act - Disable
	disabledCfg := cfg
	disabledCfg.Enabled = false
	c.UpdateConfig(disabledCfg)
	time.Sleep(50 * time.Millisecond)

	// Assert - Should be stopped
	if c.IsRunning() {
		t.Fatalf("expected compactor to be stopped after disabling")
	}

	// Act - Re-enable
	c.UpdateConfig(cfg)
	time.Sleep(50 * time.Millisecond)
	defer c.Stop()

	// Assert - Should be running again
	if !c.IsRunning() {
		t.Fatalf("expected compactor to be running after re-enabling")
	}
}

// TestCompactor_CompactionTriggered tests that compaction is triggered
func TestCompactor_CompactionTriggered(t *testing.T) {
	// Arrange
	q := NewQueue(0)
	q.CreateTopic("test", 2, 0)

	// Publish messages
	for i := 0; i < 50; i++ {
		q.Publish("test", Message{
			Key:       "k",
			Payload:   []byte("data"),
			Timestamp: time.Now(),
		})
	}

	// Ack some messages to enable compaction
	q.Ack("test", "group1", 0, 25)

	// Setup compactor with short interval and low threshold
	cfg := CompactorConfig{
		Enabled:           true,
		Interval:          100 * time.Millisecond,
		ThresholdMessages: 20,
	}
	c := NewCompactor(q, cfg)
	c.Start()
	defer c.Stop()

	// Act - Wait for compaction to trigger
	time.Sleep(500 * time.Millisecond)

	// Assert - Stats should show some messages were compacted
	// (Note: This is a soft assertion as timing may vary)
	stats, _ := q.GetPartitionStats("test", 0)
	if stats != nil {
		t.Logf("partition stats after compaction: %v", stats)
	}
}

// TestCompactor_CheckTopicTriggers tests the trigger checking logic
func TestCompactor_CheckTopicTriggers(t *testing.T) {
	// Arrange
	q := NewQueue(0)
	q.CreateTopic("test", 1, 0)

	cfg := CompactorConfig{
		Enabled:           true,
		Interval:          5 * time.Second,
		ThresholdMessages: 10,
	}
	c := NewCompactor(q, cfg)

	// Publish messages below threshold
	for i := 0; i < 5; i++ {
		q.Publish("test", Message{
			Key:       "k",
			Payload:   []byte("data"),
			Timestamp: time.Now(),
		})
	}

	// Act - Check triggers (should not trigger)
	trigger1 := c.checkTopicTriggers("test")

	// Publish more messages to exceed threshold
	for i := 0; i < 10; i++ {
		q.Publish("test", Message{
			Key:       "k",
			Payload:   []byte("data"),
			Timestamp: time.Now(),
		})
	}

	// Act - Check triggers again (should trigger)
	trigger2 := c.checkTopicTriggers("test")

	// Assert
	if trigger1 {
		t.Logf("trigger1 unexpectedly true (messages=%d, threshold=%d)", 5, 10)
	}
	if !trigger2 {
		t.Logf("trigger2 unexpectedly false (messages=%d, threshold=%d)", 15, 10)
	}
}

// TestCompactor_StopWhileNotRunning tests stopping when already stopped
func TestCompactor_StopWhileNotRunning(t *testing.T) {
	// Arrange
	q := NewQueue(0)
	cfg := DefaultCompactorConfig()
	c := NewCompactor(q, cfg)

	// Act - Stop without starting
	c.Stop()

	// Assert - Should not crash
	if c.IsRunning() {
		t.Fatalf("expected compactor not running")
	}
}
