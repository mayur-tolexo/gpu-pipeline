package api

import (
	"net/http"
	"strings"
)

// RegisterRoutes registers all MQ API routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/topics", h.handleTopicsRoot)
	mux.HandleFunc("/topics/", h.handleTopicsSubroutes)
	mux.HandleFunc("/healthz", h.handleHealth)
	mux.HandleFunc("/swagger", h.HandleSwagger)
}

// handleTopicsRoot routes POST /topics (create topic)
func (h *Handler) handleTopicsRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/topics" {
		h.HandleCreateTopic(w, r)
		return
	}
	http.NotFound(w, r)
}

// handleTopicsSubroutes routes /topics/{topic}/...
func (h *Handler) handleTopicsSubroutes(w http.ResponseWriter, r *http.Request) {
	// path: /topics/{topic}/action
	path := r.URL.Path
	if !strings.HasPrefix(path, "/topics/") {
		http.NotFound(w, r)
		return
	}
	// trim /topics/ prefix
	rest := path[len("/topics/"):]
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	topic := parts[0]
	action := parts[1]

	switch action {
	case "publish":
		h.HandlePublish(w, r, topic)
	case "consume":
		h.HandleConsume(w, r, topic)
	case "ack":
		h.HandleAck(w, r, topic)
	default:
		http.NotFound(w, r)
	}
}

// handleHealth handles GET /healthz
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
