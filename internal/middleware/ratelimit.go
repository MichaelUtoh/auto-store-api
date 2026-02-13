package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter provides per-key rate limiting
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rpm      int
	burst    int
}

func NewRateLimiter(rpm, burst int) *RateLimiter {
	if burst <= 0 {
		burst = rpm
	}
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rpm:      rpm,
		burst:    burst,
	}
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if lim, ok := rl.limiters[key]; ok {
		return lim
	}
	lim := rate.NewLimiter(rate.Every(time.Minute/time.Duration(rl.rpm)), rl.burst)
	rl.limiters[key] = lim
	return lim
}

// RateLimit returns a middleware that limits requests per client IP
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.ClientIP()
		lim := rl.getLimiter(key)
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if !lim.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		_ = ctx
		c.Next()
	}
}
