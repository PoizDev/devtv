package middlewares

import (
	"devtv/config"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// IPRateLimiter - Her IP için ayrı rate limiter
type IPRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	r        rate.Limit // İstek/saniye
	b        int        // Burst (ani yoğunluk)
}

// NewIPRateLimiter - Yeni rate limiter oluştur
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

// GetLimiter - IP için limiter al veya oluştur
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

// RateLimitMiddleware - Rate limiting middleware
func RateLimitMiddleware(limiter *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		// IP için limiter al
		ipLimiter := limiter.GetLimiter(ip)

		// İstek yapılabilir mi kontrol et
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

// CleanupMiddleware - Eski IP'leri temizle (opsiyonel)
func (i *IPRateLimiter) Cleanup() {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Limiters map'ini temizle (memory leak önleme)
	// Not: Production'da daha sofistike bir cleanup gerekebilir
	if len(i.limiters) > 10000 {
		i.limiters = make(map[string]*rate.Limiter)
		config.Log.Info("Rate limiter cache temizlendi")
	}
}
