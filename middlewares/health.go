package middlewares

import (
	"devtv/in"
	"devtv/models"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/jeanphorn/log4go"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

var AppStartTime = time.Now()
var ActiveWebSockets int32 = 0

// Cache için global değişkenler
var (
	cachedHealthData models.HealthData
	healthCacheMu    sync.RWMutex
	lastUpdateTime   time.Time
)

// Bu fonksiyon WebSocket açılırken çağrılacak
func IncreaseWS() {
	atomic.AddInt32(&ActiveWebSockets, 1)
}

// Bu fonksiyon WebSocket kapanırken çağrılacak
func DecreaseWS() {
	atomic.AddInt32(&ActiveWebSockets, -1)
}

// StartHealthCollector - Arka planda health data'yı topla (30 saniyede bir)
func StartHealthCollector() {
	// İlk veriyi hemen topla
	updateHealthData()

	// Her 30 saniyede bir güncelle
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			updateHealthData()
		}
	}()

	log.Info("Health collector başlatıldı - Güncelleme: 30 saniye")
}

// updateHealthData - Sistem metriklerini topla ve cache'le
func updateHealthData() {
	// ---- SYSTEM UPTIME ----
	uptime, _ := host.Uptime()

	// ---- CPU ----
	cpuPercent, _ := cpu.Percent(time.Second, false)

	// ---- RAM ----
	vm, _ := mem.VirtualMemory()

	// ---- NETWORK ----
	netStats, _ := net.IOCounters(false)
	var bytesRecv, bytesSent uint64
	if len(netStats) > 0 {
		bytesRecv = netStats[0].BytesRecv
		bytesSent = netStats[0].BytesSent
	}

	// ---- DISK ----
	paths := []string{"/", "C:\\"}
	var allDisks []models.DiskUsage

	for _, p := range paths {
		du, err := disk.Usage(p)
		if err == nil {
			allDisks = append(allDisks, models.DiskUsage{
				Path:         p,
				TotalMB:      du.Total / (1024 * 1024),
				UsedMB:       du.Used / (1024 * 1024),
				UsagePercent: du.UsedPercent,
			})
		}
	}

	// ---- DB POOL ----
	sqlDB, _ := in.DB.DB()
	dbStats := sqlDB.Stats()
	db := models.DBStats{
		MaxOpenConns: dbStats.MaxOpenConnections,
		OpenConns:    dbStats.OpenConnections,
		InUse:        dbStats.InUse,
		Idle:         dbStats.Idle,
	}

	// ---- GOROUTINES ----
	goroutines := runtime.NumGoroutine()

	// ---- APP UPTIME ----
	appUptime := time.Since(AppStartTime).String()

	// Cache'i güncelle (thread-safe)
	healthCacheMu.Lock()
	cachedHealthData = models.HealthData{
		Uptime:          uptime,
		CPUUsagePercent: safeGet(cpuPercent),
		RAMTotalMB:      vm.Total / (1024 * 1024),
		RAMUsedMB:       vm.Used / (1024 * 1024),
		RAMUsagePercent: vm.UsedPercent,
		NetBytesRecv:    bytesRecv,
		NetBytesSent:    bytesSent,
		DiskUsages:      allDisks,

		ActiveWebSockets:  int(atomic.LoadInt32(&ActiveWebSockets)),
		GoRoutinesCount:   goroutines,
		DBConnectionStats: db,
		AppUptime:         appUptime,
	}
	lastUpdateTime = time.Now()
	healthCacheMu.Unlock()

	log.Fine("Health data güncellendi")
}

// GetCachedHealthData - Cache'den health data'yı al (çok hızlı!)
func GetCachedHealthData() models.HealthData {
	healthCacheMu.RLock()
	defer healthCacheMu.RUnlock()

	// Real-time güncellenebilecek değerler
	data := cachedHealthData
	data.ActiveWebSockets = int(atomic.LoadInt32(&ActiveWebSockets))
	data.GoRoutinesCount = runtime.NumGoroutine()
	data.AppUptime = time.Since(AppStartTime).String()

	// DB stats da real-time olabilir (çok hızlı)
	sqlDB, _ := in.DB.DB()
	if sqlDB != nil {
		dbStats := sqlDB.Stats()
		data.DBConnectionStats = models.DBStats{
			MaxOpenConns: dbStats.MaxOpenConnections,
			OpenConns:    dbStats.OpenConnections,
			InUse:        dbStats.InUse,
			Idle:         dbStats.Idle,
		}
	}

	return data
}

// HealthMiddleware - Artık sadece cache'den okur (çok hızlı!)
func HealthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Cache'den oku (< 1ms)
		health := GetCachedHealthData()
		c.Set("health_data", health)
		c.Next()
	}
}

func safeGet(arr []float64) float64 {
	if len(arr) == 0 {
		return 0
	}
	return arr[0]
}

func GetHealth(c *gin.Context) {
	data, exists := c.Get("health_data")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Health data bulunamadı",
		})
		return
	}

	healthData := data.(models.HealthData)

	// Metadata ekle
	response := gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
		"cache_age": time.Since(lastUpdateTime).String(),
		"data":      healthData,
	}

	c.JSON(http.StatusOK, response)
}
