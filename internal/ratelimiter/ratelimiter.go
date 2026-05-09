package ratelimiter

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	// AuthenticatedRate is the request rate for authenticated GitHub API (30 req/min)
	AuthenticatedRate = 30.0 / 60.0 // 0.5 req/sec
	// UnauthenticatedRate is the request rate for unauthenticated API (10 req/min)
	UnauthenticatedRate = 10.0 / 60.0 // ~0.167 req/sec
	// TokenCapacity is the burst capacity of the token bucket
	TokenCapacity = 5.0
)

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	tokens    float64
	capacity  float64
	rate      float64 // tokens per second
	mu        sync.Mutex
	lastRefill time.Time
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(rate float64) *TokenBucket {
	capacity := TokenCapacity
	if capacity > rate*10 {
		capacity = rate * 10
	}
	return &TokenBucket{
		tokens:     capacity,
		capacity:   capacity,
		rate:       rate,
		lastRefill: time.Now(),
	}
}

// Wait blocks until a token is available
func (tb *TokenBucket) Wait(ctx context.Context) error {
	for {
		tb.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(tb.lastRefill).Seconds()
		tb.tokens += elapsed * tb.rate
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastRefill = now

		if tb.tokens >= 1.0 {
			tb.tokens--
			tb.mu.Unlock()
			return nil
		}

		// Calculate wait time for next token
		waitTime := time.Duration((1.0 - tb.tokens) / tb.rate * float64(time.Second))
		tb.mu.Unlock()

		select {
		case <-time.After(waitTime):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// SetRate updates the rate dynamically (for fallback scenarios)
func (tb *TokenBucket) SetRate(rate float64) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	oldRate := tb.rate
	tb.rate = rate
	log.Printf("[RateLimiter] Rate changed: %.3f req/sec → %.3f req/sec", oldRate, rate)
}

// Rate returns the current rate
func (tb *TokenBucket) Rate() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.rate
}

// RateLimiter coordinates rate limiting across multiple consumers
type RateLimiter struct {
	bucket      *TokenBucket
	isAuth      bool
	mu          sync.RWMutex
	fallbackMsg sync.Once
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(isAuthenticated bool) *RateLimiter {
	rate := UnauthenticatedRate
	if isAuthenticated {
		rate = AuthenticatedRate
	}

	return &RateLimiter{
		bucket: NewTokenBucket(rate),
		isAuth: isAuthenticated,
	}
}

// Wait blocks until a token is available
func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.bucket.Wait(ctx)
}

// IsAuthenticated returns true if currently using authenticated rate limits
func (rl *RateLimiter) IsAuthenticated() bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.isAuth
}

// FallbackToUnauthenticated switches to unauthenticated rate limits
func (rl *RateLimiter) FallbackToUnauthenticated() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if !rl.isAuth {
		return // Already unauthenticated
	}

	rl.isAuth = false
	rl.bucket.SetRate(UnauthenticatedRate)
	rl.fallbackMsg.Do(func() {
		log.Printf("[RateLimiter] ⚠️  WARNING: Authentication failed or rejected. " +
			"Falling back to unauthenticated rate limit (%.0f req/min). " +
			"Consider checking your API token.", UnauthenticatedRate*60)
	})
}

// String returns a human-readable description of the current rate limit
func (rl *RateLimiter) String() string {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	authStatus := "unauthenticated"
	if rl.isAuth {
		authStatus = "authenticated"
	}

	rate := rl.bucket.Rate()
	return fmt.Sprintf("%s (%.0f requests/minute)", authStatus, rate*60)
}
