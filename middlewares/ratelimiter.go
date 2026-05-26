package middlewares

import (
	"devtv/config"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	r        rate.Limit
	b        int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.limiters[ip] = limiter
	}

	return limiter
}

func RateLimitMiddleware(limiter *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		ipLimiter := limiter.GetLimiter(ip)

		if !ipLimiter.Allow() {
			config.Log.Warn("Rate limit aşıldı", zap.String("ip", ip), zap.String("path", c.Request.URL.Path))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Çok fazla istek",
				"message":     "Lütfen bir süre bekleyip tekrar deneyin",
				"retry_after": "1 saniye",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (i *IPRateLimiter) Cleanup() {
	i.mu.Lock()
	defer i.mu.Unlock()

	if len(i.limiters) > 10000 {
		i.limiters = make(map[string]*rate.Limiter)
		config.Log.Info("Rate limiter cache temizlendi")
	}
}

func (i *IPRateLimiter) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			i.Cleanup()
		}
	}()
}
