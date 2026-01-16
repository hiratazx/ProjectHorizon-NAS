package api

import (
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

func RegisterSystemRoutes(rg *gin.RouterGroup) {
	system := rg.Group("/system")
	{
		system.GET("/info", getSystemInfo)
		system.GET("/cpu", getCPUInfo)
		system.GET("/memory", getMemoryInfo)
		system.GET("/network", getNetworkInfo)
		system.GET("/disks", getDiskInfo)
	}
}

// SystemInfo represents overall system information
type SystemInfo struct {
	Hostname    string  `json:"hostname"`
	Platform    string  `json:"platform"`
	OS          string  `json:"os"`
	Arch        string  `json:"arch"`
	KernelVer   string  `json:"kernelVersion"`
	Uptime      uint64  `json:"uptime"`
	BootTime    uint64  `json:"bootTime"`
	CPUUsage    float64 `json:"cpuUsage"`
	MemoryUsage float64 `json:"memoryUsage"`
	CPUCores    int     `json:"cpuCores"`
	MemTotal    uint64  `json:"memTotal"`
	MemUsed     uint64  `json:"memUsed"`
}

func getSystemInfo(c *gin.Context) {
	hostInfo, err := host.Info()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cpuPercent, _ := cpu.Percent(time.Second, false)
	memInfo, _ := mem.VirtualMemory()

	hostname, _ := os.Hostname()

	info := SystemInfo{
		Hostname:    hostname,
		Platform:    hostInfo.Platform,
		OS:          hostInfo.OS,
		Arch:        runtime.GOARCH,
		KernelVer:   hostInfo.KernelVersion,
		Uptime:      hostInfo.Uptime,
		BootTime:    hostInfo.BootTime,
		CPUCores:    runtime.NumCPU(),
		CPUUsage:    0,
		MemoryUsage: memInfo.UsedPercent,
		MemTotal:    memInfo.Total,
		MemUsed:     memInfo.Used,
	}

	if len(cpuPercent) > 0 {
		info.CPUUsage = cpuPercent[0]
	}

	c.JSON(http.StatusOK, info)
}

// CPUInfo represents CPU information
type CPUInfo struct {
	Cores       int       `json:"cores"`
	ModelName   string    `json:"modelName"`
	Usage       float64   `json:"usage"`
	PerCoreUsage []float64 `json:"perCoreUsage"`
}

func getCPUInfo(c *gin.Context) {
	cpuInfo, err := cpu.Info()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cpuPercent, _ := cpu.Percent(time.Second, false)
	perCore, _ := cpu.Percent(time.Second, true)

	info := CPUInfo{
		Cores:       runtime.NumCPU(),
		PerCoreUsage: perCore,
	}

	if len(cpuInfo) > 0 {
		info.ModelName = cpuInfo[0].ModelName
	}
	if len(cpuPercent) > 0 {
		info.Usage = cpuPercent[0]
	}

	c.JSON(http.StatusOK, info)
}

// MemoryInfo represents memory information
type MemoryInfo struct {
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"usedPercent"`
	SwapTotal   uint64  `json:"swapTotal"`
	SwapUsed    uint64  `json:"swapUsed"`
}

func getMemoryInfo(c *gin.Context) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	swapInfo, _ := mem.SwapMemory()

	info := MemoryInfo{
		Total:       memInfo.Total,
		Available:   memInfo.Available,
		Used:        memInfo.Used,
		UsedPercent: memInfo.UsedPercent,
	}

	if swapInfo != nil {
		info.SwapTotal = swapInfo.Total
		info.SwapUsed = swapInfo.Used
	}

	c.JSON(http.StatusOK, info)
}

// NetworkInterface represents a network interface
type NetworkInterface struct {
	Name        string   `json:"name"`
	MTU         int      `json:"mtu"`
	HardwareAddr string  `json:"mac"`
	Addrs       []string `json:"addresses"`
	BytesSent   uint64   `json:"bytesSent"`
	BytesRecv   uint64   `json:"bytesRecv"`
}

func getNetworkInfo(c *gin.Context) {
	interfaces, err := net.Interfaces()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	stats, _ := net.IOCounters(true)
	statsMap := make(map[string]net.IOCountersStat)
	for _, s := range stats {
		statsMap[s.Name] = s
	}

	var result []NetworkInterface
	for _, iface := range interfaces {
		addrs := make([]string, 0)
		for _, addr := range iface.Addrs {
			addrs = append(addrs, addr.Addr)
		}

		ni := NetworkInterface{
			Name:         iface.Name,
			MTU:          iface.MTU,
			HardwareAddr: iface.HardwareAddr,
			Addrs:        addrs,
		}

		if stat, ok := statsMap[iface.Name]; ok {
			ni.BytesSent = stat.BytesSent
			ni.BytesRecv = stat.BytesRecv
		}

		result = append(result, ni)
	}

	c.JSON(http.StatusOK, result)
}

// DiskInfo represents disk information
type DiskInfo struct {
	Device      string  `json:"device"`
	Mountpoint  string  `json:"mountpoint"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"usedPercent"`
}

func getDiskInfo(c *gin.Context) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var result []DiskInfo
	for _, p := range partitions {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		result = append(result, DiskInfo{
			Device:      p.Device,
			Mountpoint:  p.Mountpoint,
			Fstype:      p.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
		})
	}

	c.JSON(http.StatusOK, result)
}
