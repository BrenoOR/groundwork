package guardrails

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// Default limits applied to the MCP server.
const (
	DefaultMaxFiles        = 10_000
	DefaultMaxProjectBytes = 100 * 1024 * 1024 // 100 MB
	DefaultMaxFileBytes    = 512 * 1024         // 512 KB
	DefaultRatePerSec      = 10.0
	DefaultBurst           = 20.0
)

// SizeLimits holds project scan size limits.
type SizeLimits struct {
	MaxFiles        int
	MaxProjectBytes int64
	MaxFileBytes    int64
}

// DefaultSizeLimits returns limits using the package constants.
func DefaultSizeLimits() SizeLimits {
	return SizeLimits{
		MaxFiles:        DefaultMaxFiles,
		MaxProjectBytes: DefaultMaxProjectBytes,
		MaxFileBytes:    DefaultMaxFileBytes,
	}
}

// RateLimiter is a thread-safe token-bucket rate limiter.
type RateLimiter struct {
	mu       sync.Mutex
	tokens   float64
	maxBurst float64
	rate     float64 // tokens per second
	lastAt   time.Time
}

// NewRateLimiter creates a rate limiter allowing ratePerSec calls/s with burst capacity.
func NewRateLimiter(ratePerSec, burst float64) *RateLimiter {
	return &RateLimiter{
		tokens:   burst,
		maxBurst: burst,
		rate:     ratePerSec,
		lastAt:   time.Now(),
	}
}

// Allow returns true if the call is within rate limits, consuming one token.
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(r.lastAt).Seconds()
	r.lastAt = now
	r.tokens = math.Min(r.maxBurst, r.tokens+elapsed*r.rate)
	if r.tokens >= 1 {
		r.tokens--
		return true
	}
	return false
}

// ErrRateLimited returns the standard rate-limit error message.
func ErrRateLimited() error {
	return fmt.Errorf("rate limit exceeded — try again in a moment")
}
