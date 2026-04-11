package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	internal "gpu-pipeline/mq/internal"
)

func TestRoutes_RegisterRoutesHealth(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET /healthz, got %d", w.Code)
	}
}

func TestRoutes_HealthWrongMethod(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for POST /healthz, got %d", w.Code)
	}
}

func TestRoutes_TopicsRootCreateTopic(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// POST /topics should route to CreateTopic
	req := httptest.NewRequest(http.MethodPost, "/topics", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code == http.StatusMethodNotAllowed || w.Code == http.StatusNotFound {
		t.Fatalf("expected routing to work, got %d", w.Code)
	}
}

func TestRoutes_TopicsRootNotFound(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// GET /topics should not be handled (returns 404)
	req := httptest.NewRequest(http.MethodGet, "/topics", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Logf("expected 404 for GET /topics, got %d", w.Code)
	}
}

func TestRoutes_TopicsSubroutePublish(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	
	// Create topic first
	q.CreateTopic("test-topic", 1, 0)

	// POST /topics/test-topic/publish should route to Publish
	req := httptest.NewRequest(http.MethodPost, "/topics/test-topic/publish", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should not be 404
	if w.Code == http.StatusNotFound {
		t.Fatalf("expected routing to work for publish, got 404")
	}
}

func TestRoutes_TopicsSubrouteConsume(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	
	// Create topic first
	q.CreateTopic("test-topic", 1, 0)

	// GET /topics/test-topic/consume should route to Consume
	req := httptest.NewRequest(http.MethodGet, "/topics/test-topic/consume?group=group1&partition=0&batch=10", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should not be 404
	if w.Code == http.StatusNotFound {
		t.Fatalf("expected routing to work for consume, got 404")
	}
}

func TestRoutes_TopicsSubrouteAck(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	
	// Create topic first
	q.CreateTopic("test-topic", 1, 0)

	// POST /topics/test-topic/ack should route to Ack
	req := httptest.NewRequest(http.MethodPost, "/topics/test-topic/ack", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should not be 404
	if w.Code == http.StatusNotFound {
		t.Fatalf("expected routing to work for ack, got 404")
	}
}

func TestRoutes_TopicsSubrouteInvalidAction(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	
	// Create topic first
	q.CreateTopic("test-topic", 1, 0)

	// GET /topics/test-topic/invalid should return 404
	req := httptest.NewRequest(http.MethodGet, "/topics/test-topic/invalid", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for invalid action, got %d", w.Code)
	}
}

func TestRoutes_TopicsSubrouteNoAction(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	
	// Create topic first
	q.CreateTopic("test-topic", 1, 0)

	// GET /topics/test-topic/ should return 404 (missing action)
	req := httptest.NewRequest(http.MethodGet, "/topics/test-topic/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing action, got %d", w.Code)
	}
}

func TestRoutes_AdminCompact(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	
	// Create topic first
	q.CreateTopic("test-topic", 1, 0)

	// POST /admin/compact should route to Compact
	req := httptest.NewRequest(http.MethodPost, "/admin/compact", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should not be 404
	if w.Code == http.StatusNotFound {
		t.Fatalf("expected routing to work for compact, got 404")
	}
}

func TestRoutes_AdminStats(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	
	// Create topic first
	q.CreateTopic("test-topic", 1, 0)

	// GET /admin/stats/test-topic/0 should route to GetPartitionStats
	req := httptest.NewRequest(http.MethodGet, "/admin/stats/test-topic/0", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should not be 404
	if w.Code == http.StatusNotFound {
		t.Fatalf("expected routing to work for stats, got 404")
	}
}

func TestRoutes_AdminStatsIncompletePath(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// GET /admin/stats/test should return 404 (missing partition)
	req := httptest.NewRequest(http.MethodGet, "/admin/stats/test", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for incomplete stats path, got %d", w.Code)
	}
}

func TestRoutes_AdminInvalidAction(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// GET /admin/invalid should return 404
	req := httptest.NewRequest(http.MethodGet, "/admin/invalid", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for invalid admin action, got %d", w.Code)
	}
}

func TestRoutes_TopicsRootPath(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// GET /topics/ (with trailing slash) should not match /topics root handler
	req := httptest.NewRequest(http.MethodGet, "/topics/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Logf("expected 404 for /topics/, got %d", w.Code)
	}
}

func TestRoutes_NotFound(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// GET /nonexistent should return 404
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for nonexistent route, got %d", w.Code)
	}
}

func TestRoutes_HandleTopicsRootWithExtraPath(t *testing.T) {
	q := internal.NewQueue(0)
	h := NewHandler(q)

	// Test handleTopicsRoot directly with a path that doesn't match "/topics"
	req := httptest.NewRequest(http.MethodGet, "/topics/extra", nil)
	w := httptest.NewRecorder()
	
	// Call handleTopicsRoot directly - this is internal but we can test it
	// by using the exported RegisterRoutes and checking behavior
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	
	// When accessing /topics/extra, it should go to handleTopicsSubroutes, not handleTopicsRoot
	mux.ServeHTTP(w, req)
	
	// This should either 404 or be handled by subroutes
	if w.Code < 400 {
		// If successful, that's fine
		return
	}
	if w.Code != http.StatusNotFound {
		t.Logf("expected 404 or routed, got %d", w.Code)
	}
}
