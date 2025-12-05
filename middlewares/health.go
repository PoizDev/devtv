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

var (
	cachedHealthData models.HealthData
	healthCacheMu    sync.RWMutex
	lastUpdateTime   time.Time
)

func IncreaseWS() {
	atomic.AddInt32(&ActiveWebSockets, 1)
}

func DecreaseWS() {
	atomic.AddInt32(&ActiveWebSockets, -1)
}

func StartHealthCollector() {
	updateHealthData()

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			updateHealthData()
		}
	}()

	log.Info("Health collector başlatıldı - Güncelleme: 30 saniye")
}

func updateHealthData() {
	uptime, _ := host.Uptime()

	cpuPercent, _ := cpu.Percent(time.Second, false)

	vm, _ := mem.VirtualMemory()

	netStats, _ := net.IOCounters(false)
	var bytesRecv, bytesSent uint64
	if len(netStats) > 0 {
		bytesRecv = netStats[0].BytesRecv
		bytesSent = netStats[0].BytesSent
	}

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

	sqlDB, _ := in.DB.DB()
	dbStats := sqlDB.Stats()
	db := models.DBStats{
		MaxOpenConns: dbStats.MaxOpenConnections,
		OpenConns:    dbStats.OpenConnections,
		InUse:        dbStats.InUse,
		Idle:         dbStats.Idle,
	}

	goroutines := runtime.NumGoroutine()

	appUptime := FormatUptime(time.Since(AppStartTime))

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

func GetCachedHealthData() models.HealthData {
	healthCacheMu.RLock()
	defer healthCacheMu.RUnlock()

	data := cachedHealthData
	data.ActiveWebSockets = int(atomic.LoadInt32(&ActiveWebSockets))
	data.GoRoutinesCount = runtime.NumGoroutine()
	data.AppUptime = FormatUptime(time.Since(AppStartTime))

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
	metrics := GetMetrics()

	// Metadata ekle
	response := gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
		"cache_age": time.Since(lastUpdateTime).String(),
		"data":      healthData,
		"metrics":   metrics,
	}

	c.JSON(http.StatusOK, response)
}
