package models

type DiskUsage struct {
	Path         string  `json:"path"`
	TotalMB      uint64  `json:"total_mb"`
	UsedMB       uint64  `json:"used_mb"`
	UsagePercent float64 `json:"usage_percent"`
}

type DBStats struct {
	MaxOpenConns int `json:"max_open_conns"`
	OpenConns    int `json:"open_conns"`
	InUse        int `json:"in_use"`
	Idle         int `json:"idle"`
}

type HealthData struct {
	Uptime          uint64      `json:"system_uptime_seconds"`
	CPUUsagePercent float64     `json:"cpu_usage_percent"`
	RAMTotalMB      uint64      `json:"ram_total_mb"`
	RAMUsedMB       uint64      `json:"ram_used_mb"`
	RAMUsagePercent float64     `json:"ram_usage_percent"`
	NetBytesRecv    uint64      `json:"net_bytes_received_total"`
	NetBytesSent    uint64      `json:"net_bytes_sent_total"`
	DiskUsages      []DiskUsage `json:"disk_usages"`

	ActiveWebSockets  int     `json:"active_websockets"`
	GoRoutinesCount   int     `json:"goroutine_count"`
	DBConnectionStats DBStats `json:"db_connection_stats"`
	AppUptime         string  `json:"app_uptime"`
}
