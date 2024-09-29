package ratelimiter

import (
	"log"
	"math"
	"sync"
	"time"
)

const maxErrBan = 10

// ClientAction represents an action to take for a client.
type ClientAction int

const (
	ActionAllow ClientAction = iota
	ActionIncreaseDifficulty
	ActionBan
)

// ClientInfo stores the request count and last seen time for a client.
type ClientInfo struct {
	RequestCount float64
	ErrorCount   float64
	LastSeen     time.Time
	LastError    time.Time
	BannedUntil  time.Time // New field for ban expiration
}

// RateLimiter manages rate limiting for clients.
type RateLimiter struct {
	clients   map[string]*ClientInfo
	mu        sync.Mutex
	decayRate float64
}

// NewRateLimiter creates a new RateLimiter with the specified half-life in seconds.
func NewRateLimiter(halfLifeSeconds float64) *RateLimiter {
	decayRate := math.Ln2 / halfLifeSeconds
	rl := &RateLimiter{
		clients:   make(map[string]*ClientInfo),
		decayRate: decayRate,
	}
	go rl.decayErrorCounts()
	return rl
}

// UpdateRequestRate updates the request count for a client and returns the current count.
func (rl *RateLimiter) UpdateRequestRate(clientID string) float64 {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	clientInfo, exists := rl.clients[clientID]
	if !exists {
		clientInfo = &ClientInfo{
			RequestCount: 0.0,
			LastSeen:     now,
		}
		rl.clients[clientID] = clientInfo
	}

	deltaTime := now.Sub(clientInfo.LastSeen).Seconds()
	decayFactor := math.Exp(-rl.decayRate * deltaTime)
	clientInfo.RequestCount = clientInfo.RequestCount*decayFactor + 1.0
	clientInfo.LastSeen = now
	return clientInfo.RequestCount
}

// Cleanup removes clients that have not been seen for a specified duration.
func (rl *RateLimiter) Cleanup(inactiveDuration time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	for clientID, info := range rl.clients {
		if now.Sub(info.LastSeen) > inactiveDuration && now.After(info.BannedUntil) {
			delete(rl.clients, clientID)
		}
	}
}

func (rl *RateLimiter) GetClientAction(clientID string) ClientAction {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	clientInfo, exists := rl.clients[clientID]
	if !exists {
		return ActionAllow
	}

	now := time.Now()
	if clientInfo.BannedUntil.After(now) {
		return ActionBan
	}

	errorCount := clientInfo.ErrorCount

	if errorCount > maxErrBan {
		// Ban the client for a fixed duration if not already banned
		if clientInfo.BannedUntil.IsZero() || clientInfo.BannedUntil.Before(now) {
			clientInfo.BannedUntil = now.Add(1 * time.Minute)
			log.Printf("Client %s is banned until %v", clientID, clientInfo.BannedUntil)
		}
		return ActionBan
	} else if errorCount > maxErrBan/2 {
		return ActionIncreaseDifficulty
	} else {
		return ActionAllow
	}
}

func (rl *RateLimiter) UpdateErrorCount(clientID string) float64 {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	clientInfo, exists := rl.clients[clientID]
	if !exists {
		clientInfo = &ClientInfo{
			RequestCount: 0.0,
			ErrorCount:   1.0, // Start with 1.0 due to this error
			LastSeen:     now,
			LastError:    now,
		}
		rl.clients[clientID] = clientInfo
		return clientInfo.ErrorCount
	}

	deltaTime := now.Sub(clientInfo.LastError).Seconds()
	decayFactor := math.Exp(-rl.decayRate * deltaTime)
	clientInfo.ErrorCount = clientInfo.ErrorCount*decayFactor + 1.0
	clientInfo.LastError = now
	return clientInfo.ErrorCount
}

func (rl *RateLimiter) decayErrorCounts() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		<-ticker.C
		rl.mu.Lock()
		now := time.Now()
		for _, clientInfo := range rl.clients {
			deltaTime := now.Sub(clientInfo.LastError).Seconds()
			decayFactor := math.Exp(-rl.decayRate * deltaTime)
			clientInfo.ErrorCount *= decayFactor
			clientInfo.LastError = now
		}
		rl.mu.Unlock()
	}
}
