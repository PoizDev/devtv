package middlewares

import (
	"devtv/in"
	healthpb "devtv/middlewares/proto"
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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var AppStartTime = time.Now()
var ActiveWebSockets int32 = 0

var (
	healthCacheMu    sync.RWMutex
	lastUpdateTime   time.Time
	cachedProtoBytes []byte
	cachedJSONBytes  []byte
)

func IncreaseWS() {
	atomic.AddInt32(&ActiveWebSockets, 1)
}

func DecreaseWS() {
	atomic.AddInt32(&ActiveWebSockets, -1)
}

func safeGet(arr []float64) float64 {
	if len(arr) == 0 {
		return 0
	}
	return arr[0]
}

func StartHealthCollector() {
	UpdateProtoCache()

	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for range ticker.C {
			UpdateProtoCache()
		}
	}()

	log.Info("Health collector başlatıldı - Güncelleme: 1 saniye")
}

func collectSystemMetricsProto() *healthpb.SystemMetricsResponse {
	uptime, _ := host.Uptime()
	cpuPercent, _ := cpu.Percent(0, false)
	vm, _ := mem.VirtualMemory()

	httpMetrics := GetMetrics()

	getInt64 := func(key string) int64 {
		if val, ok := httpMetrics[key]; ok {
			switch v := val.(type) {
			case int:
				return int64(v)
			case int64:
				return v
			case float64:
				return int64(v)
			}
		}
		return 0
	}

	getFloat64 := func(key string) float64 {
		if val, ok := httpMetrics[key]; ok {
			switch v := val.(type) {
			case float64:
				return v
			case int:
				return float64(v)
			}
		}
		return 0
	}

	getString := func(key string) string {
		if val, ok := httpMetrics[key].(string); ok {
			return val
		}
		return "0"
	}

	protoMethodStats := make(map[string]int64)
	if rbm, ok := httpMetrics["requests_by_method"].(map[string]int); ok {
		for method, count := range rbm {
			protoMethodStats[method] = int64(count)
		}
	} else if rbm, ok := httpMetrics["requests_by_method"].(map[string]interface{}); ok {
		for method, count := range rbm {
			switch v := count.(type) {
			case int:
				protoMethodStats[method] = int64(v)
			case float64:
				protoMethodStats[method] = int64(v)
			}
		}
	}

	netStats, _ := net.IOCounters(false)
	var bytesRecv, bytesSent uint64
	if len(netStats) > 0 {
		bytesRecv = netStats[0].BytesRecv
		bytesSent = netStats[0].BytesSent
	}

	paths := []string{"/", "C:\\"}
	var diskUsages []*healthpb.DiskUsage
	for _, p := range paths {
		du, err := disk.Usage(p)
		if err == nil {
			diskUsages = append(diskUsages, &healthpb.DiskUsage{
				Path:         p,
				TotalMb:      du.Total / (1024 * 1024),
				UsedMb:       du.Used / (1024 * 1024),
				UsagePercent: du.UsedPercent,
			})
		}
	}

	sqlDB, _ := in.DB.DB()
	var dbStats *healthpb.DBStats
	if sqlDB != nil {
		stats := sqlDB.Stats()
		dbStats = &healthpb.DBStats{
			MaxOpenConns: int32(stats.MaxOpenConnections),
			OpenConns:    int32(stats.OpenConnections),
			InUse:        int32(stats.InUse),
			Idle:         int32(stats.Idle),
		}
	}

	return &healthpb.SystemMetricsResponse{
		SystemUptimeSecs:      uptime,
		CpuUsagePercent:       safeGet(cpuPercent),
		RamTotalMb:            vm.Total / (1024 * 1024),
		RamUsedMb:             vm.Used / (1024 * 1024),
		RamUsagePercent:       vm.UsedPercent,
		NetBytesReceivedTotal: bytesRecv,
		NetBytesSentTotal:     bytesSent,
		DiskUsages:            diskUsages,
		ActiveWebsockets:      atomic.LoadInt32(&ActiveWebSockets),
		GoroutineCount:        int32(runtime.NumGoroutine()),
		DbStats:               dbStats,
		AppUptime:             FormatUptime(time.Since(AppStartTime)),
		Timestamp:             timestamppb.Now(),
		CacheAge:              durationpb.New(time.Since(lastUpdateTime)),
		ApiMetrics: &healthpb.ApiMetrics{
			TotalRequests:      getInt64("total_requests"),
			TotalErrors:        getInt64("total_errors"),
			ErrorRatePercent:   getString("error_rate_percent"),
			SuccessRatePercent: getString("success_rate_percent"),
			AvgResponseTimeMs:  getFloat64("avg_response_time_ms"),
			RequestsByMethod:   protoMethodStats,
		},
	}
}

func UpdateProtoCache() {
	resp := collectSystemMetricsProto()

	data, err := proto.Marshal(resp)
	if err != nil {
		log.Error("Proto health marshal hatası: %s", err)
		return
	}

	jsonData, err := protojson.Marshal(resp)
	if err != nil {
		log.Error("JSON health marshal hatası: %s", err)
		return
	}

	healthCacheMu.Lock()
	cachedProtoBytes = data
	cachedJSONBytes = jsonData
	lastUpdateTime = time.Now()
	healthCacheMu.Unlock()
}

func GetHealthProto() []byte {
	healthCacheMu.RLock()
	defer healthCacheMu.RUnlock()
	return cachedProtoBytes
}

func GetHealthJSON() []byte {
	healthCacheMu.RLock()
	defer healthCacheMu.RUnlock()
	return cachedJSONBytes
}

func CheckHealthProto() *healthpb.HealthCheckResponse {
	sqlDB, err := in.DB.DB()
	if err != nil || sqlDB.Ping() != nil {
		return &healthpb.HealthCheckResponse{
			Status: healthpb.HealthCheckResponse_NOT_SERVING,
		}
	}

	return &healthpb.HealthCheckResponse{
		Status: healthpb.HealthCheckResponse_SERVING,
	}
}

func ProtoHealthHandler(c *gin.Context) {
	if c.Query("format") == "json" {
		jsonBytes := GetHealthJSON()
		if jsonBytes == nil {
			c.Status(http.StatusServiceUnavailable)
			return
		}
		c.Data(http.StatusOK, "application/json", jsonBytes)
		return
	}

	data := GetHealthProto()
	if data == nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}
	c.Data(http.StatusOK, "application/x-protobuf", data)
}

func ProtoHealthCheckHandler(c *gin.Context) {
	resp := CheckHealthProto()

	if c.Query("format") == "json" {
		jsonBytes, err := protojson.MarshalOptions{Indent: "  "}.Marshal(resp)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "JSON marshal hatası"})
			return
		}
		c.Data(http.StatusOK, "application/json", jsonBytes)
		return
	}

	data, err := proto.Marshal(resp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Proto marshal hatası"})
		return
	}
	c.Data(http.StatusOK, "application/x-protobuf", data)
}
