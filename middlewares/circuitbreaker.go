package middlewares

import (
	"devtv/config"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

//! Fallback Cache Mekanizması Eklenecek Prod'a çıkmadan kesin ve net! (eklenemedi ama bu sene eklencek.)

type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	maxFailures  int                 // Kaç hata sonra açılsın
	timeout      time.Duration       // Kaç süre sonra tekrar denesin
	state        CircuitBreakerState // Mevcut durum
	failures     int                 // Hata sayacı
	lastFailTime time.Time           // Son hata zamanı
	mu           sync.RWMutex        // Thread-safe
}

func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures: maxFailures,
		timeout:     timeout,
		state:       StateClosed,
		failures:    0,
	}
}

// CircuitBreakerMiddleware - Gin middleware
func CircuitBreakerMiddleware(cb *CircuitBreaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Devre açık mı kontrol et
		if !cb.AllowRequest() {
			config.Log.Warn("Circuit Breaker AÇIK - İstek reddedildi", zap.String("path", c.Request.URL.Path))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "Servis geçici olarak kullanılamıyor",
				"message": "Lütfen birkaç saniye sonra tekrar deneyin",
			})
			c.Abort()
			return
		}

		// İsteği işle
		c.Next()

		// Response status'e göre başarı/başarısızlık kaydı
		if c.Writer.Status() >= http.StatusBadGateway {
			cb.RecordFailure()
		} else {
			cb.RecordSuccess()
		}
	}
}

// AllowRequest - İstek yapılabilir mi?
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// Normal çalışma, izin ver
		return true

	case StateOpen:
		// Timeout geçti mi? Geçtiyse half-open'a geç
		if time.Since(cb.lastFailTime) > cb.timeout {
			cb.state = StateHalfOpen
			config.Log.Info("Circuit Breaker HALF-OPEN'a geçti (test modu)")
			return true
		}
		// Hala timeout içinde, reddet
		return false

	case StateHalfOpen:
		// Test modu, tek bir istek dene
		return true

	default:
		return false
	}
}

// RecordFailure - Hata kaydı
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailTime = time.Now()

	switch cb.state {
	case StateClosed:
		// Eşik aşıldı mı?
		if cb.failures >= cb.maxFailures {
			cb.state = StateOpen
			config.Log.Warn("Circuit Breaker AÇILDI", zap.Int("failures", cb.failures))
		}

	case StateHalfOpen:
		// Test başarısız, tekrar aç
		cb.state = StateOpen
		config.Log.Warn("Circuit Breaker tekrar AÇILDI - Test başarısız")
	}
}

// RecordSuccess - Başarı kaydı
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateHalfOpen:
		// Test başarılı, kapat
		cb.state = StateClosed
		cb.failures = 0
		config.Log.Info("Circuit Breaker KAPANDI - Servis normale döndü")

	case StateClosed:
		// Normal çalışma, hata sayacını sıfırla
		if cb.failures > 0 {
			cb.failures = 0
		}
	}
}

// GetState - Mevcut durumu al (monitoring için)
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetFailures - Hata sayısını al (monitoring için)
func (cb *CircuitBreaker) GetFailures() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}
