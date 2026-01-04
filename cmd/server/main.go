package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/dontdude/goxec/internal/domain"
	"github.com/dontdude/goxec/internal/platform/queue"
	"github.com/dontdude/goxec/internal/platform/web"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Global Hub to manage active WebSocket connections.
// Map key: JobID -> Value: WebSocket Connection
var (
	clientHub = make(map[string]*websocket.Conn)
	hubMu	  sync.RWMutex
)

func main() {
	// 1. Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Initialize Redis Queue (as a dependency)
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisQ := queue.NewRedisQueue(redisAddr, "goxec:jobs", "goxec:workers")

	// 3. Start Log Broadcaster (Background goroutine)
	go broadcastLogs(redisQ)

	// 4. Setup Rate Limiter
	// Rate: 0.5 tokens/sec (1 request every 2s), Capacity: 5 (Burst)
	limiter := web.NewRateLimiter(0.5, 5.0)

	// 5. Setup Router (Standard Lib)
	mux := http.NewServeMux()

	// 6. Register Handlers
	// Post /api/run -> Enqueues Job (Wrapped with RateLimit)
	mux.HandleFunc("POST /api/run", limiter.RateLimitMiddleware(handleSubmit(redisQ)))

	// Get /api/ws -> WebSocket Updgrade
	mux.HandleFunc("GET /api/ws", handleWS())

	// 7. Middleware (CORS)
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

// handleWS upgrades the connection to WebSocket and registers it to hub.
func handleWS() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract JobID from Query Params
		jobID := r.URL.Query().Get("job_id")
		if jobID == "" {
			http.Error(w, "job_id is required", http.StatusBadRequest)
			return
		}

		// 2. Upgrade to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("WebSocket upgrade failed", "error", err)
			return
		}

		// 3. Register to Hub
		slog.Info("Client connected via WebSocket", "remoteAddr", conn.RemoteAddr())
		hubMu.Lock()
		clientHub[jobID] = conn
		hubMu.Unlock()

		// 4. Clean up on disconnect
		defer func() {
			slog.Info("Client Disconnected", "jobID", jobID)
			hubMu.Lock()
			delete(clientHub, jobID)
			hubMu.Unlock()
			conn.Close()
		}()

		// 5. Stub Loop: Keep connection alive until client disconnects
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
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

// broadcastLogs listens to the Redis Pub/Sub channel and forwards messages to connected clients.
func broadcastLogs(q domain.JobQueue) {
	slog.Info("Starting Log Broadcaster...")

	// Subscribe to all logs
	logsCh, err := q.SubscribeLogs(context.Background())
	if err != nil {
		slog.Error("Failed to subscribe to logs", "error", err)
		os.Exit(1)
	}

	for msg := range logsCh {
		// 1. Check if we have a client connected for this JobID
		hubMu.RLock()
		conn, exists := clientHub[msg.JobID]
		hubMu.RUnlock()

		if exists {
			// 2. Forward the message to the WebSocket
			// Forwarding the domain.JobResult directly as JSON
			err := conn.WriteJSON(msg)

			if err != nil {
				slog.Error("Failed to write to websocket", "jobID", msg.JobID, "error", err)
				// If write fails, we should probably close and remove the connection, 
				// but just logging it for now
			}
		}
	}
}