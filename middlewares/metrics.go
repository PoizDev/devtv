package middlewares

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/jeanphorn/log4go"
)

// Metrics - Global metrikler (atomic operasyonlar ile thread-safe)
var (
	TotalRequests       int64                     // Toplam istek sayısı
	TotalErrors         int64                     // Toplam hata sayısı (4xx, 5xx)
	TotalResponseTimeMs int64                     // Toplam response time (ms)
	RequestsByMethod    = make(map[string]*int64) // Method'a göre istek sayısı
)

func init() {
	// HTTP metodları için counter'ları initialize et
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	for _, method := range methods {
		var counter int64 = 0
		RequestsByMethod[method] = &counter
	}
}

// MetricsMiddleware - Her isteği say ve response time'ı kaydet
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// İstek başlangıcı
		start := time.Now()
		path := c.Request.URL.Path

		if strings.HasPrefix(path, "/health") || strings.HasPrefix(path, "/health/check") {
			c.Next()
			return
		}

		// İstek sayısını artır (atomic - thread-safe)
		atomic.AddInt64(&TotalRequests, 1)

		// Method'a göre istek sayısını artır
		if counter, exists := RequestsByMethod[c.Request.Method]; exists {
			atomic.AddInt64(counter, 1)
		}

		// İsteği işle
		c.Next()

		// Response time hesapla
		duration := time.Since(start)
		durationMs := duration.Milliseconds()

		// Toplam response time'a ekle
		atomic.AddInt64(&TotalResponseTimeMs, durationMs)

		// Hata sayısını artır (4xx, 5xx)
		if c.Writer.Status() >= 400 {
			atomic.AddInt64(&TotalErrors, 1)
		}

		// Yavaş istekleri logla (> 500ms)
		if duration > 500*time.Millisecond {
			log.Warn(fmt.Sprintf(
				"SLOW REQUEST: %s %s | Duration: %dms | Status: %d",
				c.Request.Method,
				c.Request.URL.Path,
				durationMs,
				c.Writer.Status(),
			))
		}
	}
}

// GetMetrics - Metrikleri al (atomic read - çok hızlı!)
func GetMetrics() map[string]interface{} {
	totalReqs := atomic.LoadInt64(&TotalRequests)
	totalErrs := atomic.LoadInt64(&TotalErrors)
	totalTime := atomic.LoadInt64(&TotalResponseTimeMs)

	// Ortalama response time hesapla (number olarak)
	var avgResponseTime float64
	if totalReqs > 0 {
		avgResponseTime = float64(totalTime) / float64(totalReqs)
	}

	// Method'lara göre istek sayıları (sadece 0'dan büyük olanlar)
	methodStats := make(map[string]int64)
	for method, counter := range RequestsByMethod {
		count := atomic.LoadInt64(counter)
		if count > 0 { // Sadece istek yapılan metodları göster
			methodStats[method] = count
		}
	}

	// Hata oranı hesapla
	errorRate := 0.0
	if totalReqs > 0 {
		errorRate = float64(totalErrs) / float64(totalReqs) * 100
	}

	// Başarı oranı
	successRate := 100.0 - errorRate

	return map[string]interface{}{
		"total_requests":       totalReqs,
		"total_errors":         totalErrs,
		"error_rate_percent":   fmt.Sprintf("%.2f%%", errorRate),
		"success_rate_percent": fmt.Sprintf("%.2f%%", successRate),
		"avg_response_time_ms": avgResponseTime, // Number olarak (string değil)
		"requests_by_method":   methodStats,
	}
}

// ResetMetrics - Metrikleri sıfırla (test için)
func ResetMetrics() {
	atomic.StoreInt64(&TotalRequests, 0)
	atomic.StoreInt64(&TotalErrors, 0)
	atomic.StoreInt64(&TotalResponseTimeMs, 0)

	for _, counter := range RequestsByMethod {
		atomic.StoreInt64(counter, 0)
	}

	log.Info("Metrikler sıfırlandı")
}

// GetMetricsHandler - Metrics endpoint handler
func GetMetricsHandler(c *gin.Context) {
	metrics := GetMetrics()

	c.JSON(200, gin.H{
		"status":    "ok",
		"timestamp": time.Now(),
		"metrics":   metrics,
	})
}
