package mq

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestPartition_BasicOperations(t *testing.T) {
	p := NewPartition("p0")
	if p.ID() != "p0" {
		t.Fatalf("unexpected id: %s", p.ID())
	}
	if p.Len() != 0 {
		t.Fatalf("expected empty partition")
	}
	if p.TailOffset() != 0 {
		t.Fatalf("expected tail 0")
	}

	m1 := Message{ID: "m1", Key: "k", Payload: []byte("a"), Timestamp: time.Now()}
	off1 := p.Append(m1)
	if off1 != 0 {
		t.Fatalf("expected offset 0, got %d", off1)
	}
	if p.Len() != 1 {
		t.Fatalf("expected len 1")
	}

	m, err := p.Get(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(m.Payload) != "a" {
		t.Fatalf("unexpected payload: %s", string(m.Payload))
	}

	// out of range
	if _, err := p.Get(1); err == nil {
		t.Fatalf("expected error for Get out of range")
	}
}

func TestPartition_ReadFrom(t *testing.T) {
	p := NewPartition("p1")
	for i := 0; i < 5; i++ {
		p.Append(Message{ID: strconv.Itoa(i), Payload: []byte{byte(i)}})
	}

	// read subset
	msgs, err := p.ReadFrom(1, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}

	// read all from offset
	all, err := p.ReadFrom(0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(all))
	}

	// offset == tail
	empty, err := p.ReadFrom(p.TailOffset(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty slice for offset==tail")
	}

	// offset > tail
	if _, err := p.ReadFrom(p.TailOffset()+1, 1); err == nil {
		t.Fatalf("expected error for offset > tail")
	}
}

func TestPartition_CommitAndOffset(t *testing.T) {
	p := NewPartition("p2")
	for i := 0; i < 3; i++ {
		p.Append(Message{ID: fmt.Sprintf("%c", rune('a')+rune(i))})
	}
	// default offset
	if p.Offset("cg") != 0 {
		t.Fatalf("expected default offset 0")
	}
	// valid commit
	if err := p.Commit("cg", 2); err != nil {
		t.Fatalf("unexpected commit error: %v", err)
	}
	if p.Offset("cg") != 2 {
		t.Fatalf("expected committed offset 2, got %d", p.Offset("cg"))
	}
	// commit beyond tail should fail
	if err := p.Commit("cg", p.TailOffset()+1); err == nil {
		t.Fatalf("expected error when committing beyond tail")
	}
}

func TestPartition_Concurrency(t *testing.T) {
	p := NewPartition("p3")
	wg := sync.WaitGroup{}
	n := 200
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			p.Append(Message{ID: fmt.Sprintf("%c", rune('A')+rune(i%26))})
		}(i)
	}
	wg.Wait()

	if p.Len() != n {
		t.Fatalf("expected len %d, got %d", n, p.Len())
	}

	// concurrent reads and commits
	wg = sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = p.ReadFrom(0, 10)
	}()
	go func() {
		defer wg.Done()
		_ = p.Commit("cg2", p.TailOffset())
	}()
	wg.Wait()

	if p.Offset("cg2") != int64(n) {
		t.Fatalf("expected committed offset %d, got %d", n, p.Offset("cg2"))
	}
}

func TestPartition_Errors(t *testing.T) {
	p := NewPartition("p4")
	// negative get
	if _, err := p.Get(-1); err == nil {
		t.Fatalf("expected error for negative offset")
	}
	// negative read
	if _, err := p.ReadFrom(-1, 1); err == nil {
		t.Fatalf("expected error for negative read offset")
	}
	// negative commit
	if err := p.Commit("c", -1); err == nil {
		t.Fatalf("expected error for negative commit")
	}
}
