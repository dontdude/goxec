package web

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Default cleanup intervals.
const (
	cleanupInterval = 1 * time.Minute
	visitorTimeout  = 3 * time.Minute
)

// Client represents a single visitor (IP) and their token bucket state.
type Client struct {
	// mu protects the individual client's state (tokens, lastRefill).
	// This allows concurrent updates to different clients without contention.
	mu 		   sync.Mutex
	tokens     float64
	lastRefill time.Time
}

// RateLimiter manages rate limiting for multiple clients using a Token Bucket algorithm.
type RateLimiter struct {
	// clients maps IP addresses to their Client state.
	clients map[string]*Client
	// mu protects the global map (adding/removing clients).
	// It uses an RWMutex so multiple readers (getting clients) can run in parallel.
	mu sync.RWMutex

	// rate is the number of tokens added per second.
	rate float64
	// capacity is the max burst size.
	capacity float64
}

// NewRateLimiter creates a RateLimiter and starts the background cleanup.
func NewRateLimiter(rate, capacity float64) *RateLimiter {
	rl := &RateLimiter{
		clients:  make(map[string]*Client),
		rate:	  rate,
		capacity: capacity,
	}

	// Start background cleanup goroutine
	go rl.cleanupVisitors()

	return rl
}

// getClient retrieves or creates a client for the give IP.
func (rl *RateLimiter) getClient(ip string) *Client {
	// 1. Fast Path: Read Lock
	rl.mu.RLock()
	c, exists := rl.clients[ip]
	rl.mu.RUnlock()

	if exists {
		return c
	}

	// 2. Slow Path: Write Lock (Create new client)
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check check
	if c, exists = rl.clients[ip]; !exists {
		c = &Client{
			tokens:     rl.capacity, // Start full
			lastRefill: time.Now(),
		}
		rl.clients[ip] = c
	}

	return c
}

// Allow checks if the request is allowed for the give IP.
// Implements the "Lazy Refill" algorithm.
func (rl *RateLimiter) Allow(ip string) bool {
	c := rl.getClient(ip)

	// Lock only this specific client
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// 1. Refill tokens based on elapsed time (Lazy Refill)
	elapsed := now.Sub(c.lastRefill).Seconds()
	tokensToAdd := elapsed * rl.rate

	if tokensToAdd > 0 {
		c.tokens += tokensToAdd
		if c.tokens > rl.capacity {
			c.tokens = rl.capacity
		}
		c.lastRefill = now
	}

	// 2. Consume token
	if c.tokens >= 1.0 {
		c.tokens--
		return true
	}

	return false
}

// cleanupVisitors removes inactive clients to prevent memory leaks. 
func (rl *RateLimiter) cleanupVisitors() {
	for {
		time.Sleep(cleanupInterval)

		rl.mu.Lock()
		for ip, c := range rl.clients {
			c.mu.Lock()
			if time.Since(c.lastRefill) > visitorTimeout {
				delete(rl.clients, ip)
			}
			c.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware wraps an http.Handler to enforce rate limits.
func (rl *RateLimiter) RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract IP (Basic implementation)
		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = fwd
		} else if strings.Contains(ip, ":") {
			ip = strings.Split(ip, ":")[0]
		}

		if !rl.Allow(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "Too Many Requests"})
			return
		}

		next(w, r)
	}
}