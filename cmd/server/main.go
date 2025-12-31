package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

    "github.com/dontdude/goxec/internal/domain"
	"github.com/dontdude/goxec/internal/platform/queue"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func main() {
	// 1. Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Initialize Redis Queue (as a dependency)
	redisQ := queue.NewRedisQueue("localhost:6379", "goxec:jobs", "goxec:workers")

	// 3. Setup Router (Standard Lib)
	mux := http.NewServeMux()

	// 4. Register Handlers
	// Post /submit -> Enqueues Job
	mux.HandleFunc("POST /submit", handleSubmit(redisQ))
	// Get /ws -> WebSocket Updgrade
	mux.HandleFunc("GET /ws", handleWS())

	// 4. Middleware (CORS)
	handler := enableCORS(mux)

	slog.Info("API Server starting on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

// handleSubmit creates a closure to inject the Queue dependency.
func handleSubmit(q domain.JobQueue) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Define Request Payload
		var req struct {
			Code	 string `json:"code"`
			Language string `json:"language"`
		}

		// Decode JSON
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate (Basic)
		if req.Code == "" || req.Language == "" {
			http.Error(w, "Code and Language are required", http.StatusBadRequest)
			return
		}

		// Create Job with UUID
		jobID := uuid.New().String()
		job := domain.Job{
			ID: 	  jobID,
			Code: 	  req.Code,
			Language: req.Language,
		}

		// Enqueue to Redis
		slog.Info("Received submission", "jobID", jobID)
		if err := q.Publish(r.Context(), job); err != nil {
			slog.Error("Failed to publish job", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Return JSON Response
		w.Header().Set("Content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"job_id": jobID,
			"status": "queued",
		})
	}
}

// WebSocket Upgrader (Gorilla)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {return true}, // Allow all origins for dev
}

// handleWS upgrades the connection to WebSocket.
func handleWS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("WebSocket upgrade failed", "error", err)
			return
		}
		defer conn.Close()

		slog.Info("Client connected via WebSocket", "remoteAddr", conn.RemoteAddr())

		// Stub Loop: Keep connection alive until client disconnects
		for {
			// Read message (ignore content for now)
			_, _, err := conn.ReadMessage()
			if err != nil {
				slog.Info("Client Disconnected", "error", err)
				break
			}
		}
	}
}

// enableCORS adds headers to allow requests from the Frontend. 
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow Localhost Frontend
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle Preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}