package middlewares

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/jeanphorn/log4go"
	"github.com/redis/go-redis/v9"
)

type bufferedWriter struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	committed  bool
}

func (w *bufferedWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *bufferedWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *bufferedWriter) Status() int {
	return w.statusCode
}

func (w *bufferedWriter) flush() {
	if w.committed {
		return
	}
	w.committed = true
	w.ResponseWriter.WriteHeader(w.statusCode)
	w.ResponseWriter.Write(w.body.Bytes())
}
func RedisFallbackCache(rdb *redis.Client, ttl time.Duration) gin.HandlerFunc {
	const dbTimeout = 3 * time.Second
	const fallbackTTL = 1 * time.Hour

	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		uri := c.Request.URL.RequestURI()
		cacheKey := "devtv:cache:" + uri
		fallbackKey := "devtv:fallback:" + uri

		redisCtx, redisCancel := context.WithTimeout(context.Background(), 2*time.Second)
		cachedData, err := rdb.Get(redisCtx, cacheKey).Bytes()
		redisCancel()

		if err == nil {
			log.Info("⚡ Cache HIT: %s", cacheKey)
			c.Data(http.StatusOK, "application/json; charset=utf-8", cachedData)
			c.Abort()
			return
		}

		originalCtx := c.Request.Context()
		dbCtx, dbCancel := context.WithTimeout(originalCtx, dbTimeout)
		defer dbCancel()

		c.Request = c.Request.WithContext(dbCtx)

		bw := &bufferedWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
			statusCode:     http.StatusOK,
		}
		c.Writer = bw

		c.Next()

		c.Request = c.Request.WithContext(originalCtx)

		if bw.statusCode == http.StatusOK && bw.body.Len() > 0 {
			bw.flush()

			responseBytes := bw.body.Bytes()

			writeCtx, writeCancel := context.WithTimeout(context.Background(), 1*time.Second)
			pipe := rdb.Pipeline()
			pipe.Set(writeCtx, cacheKey, responseBytes, ttl)
			pipe.Set(writeCtx, fallbackKey, responseBytes, fallbackTTL)
			_, errPipe := pipe.Exec(writeCtx)
			writeCancel()

			if errPipe != nil {
				log.Error("Redis'e yazılırken hata oluştu: %v", errPipe)
			} else {
				log.Info("Cache MISS → Yeni Veri Redis'e Kaydedildi: %s", cacheKey)
			}
			return
		}

		log.Warn("Controller hata döndü (status=%d) veya DB timeout, Redis fallback deneniyor: %s", bw.statusCode, cacheKey)

		fallbackCtx, fallbackCancel := context.WithTimeout(context.Background(), 2*time.Second)
		staleData, staleErr := rdb.Get(fallbackCtx, fallbackKey).Bytes()
		fallbackCancel()

		if staleErr == nil && len(staleData) > 0 {
			log.Warn("Redis FALLBACK aktif — stale veri servis ediliyor: %s", fallbackKey)
			bw.body.Reset()
			bw.statusCode = http.StatusOK
			bw.body.Write(staleData)
			bw.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
			bw.ResponseWriter.Header().Set("X-Cache-Fallback", "true")
			bw.ResponseWriter.Header().Set("X-Cache-Source", "redis-stale")
			bw.flush()
			return
		}

		log.Error("Redis'te de veri yok, controller hatası istemciye iletiliyor: %s", cacheKey)
		bw.flush()
	}
}
