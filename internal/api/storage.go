package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/disk"
)

func RegisterStorageRoutes(rg *gin.RouterGroup) {
	storage := rg.Group("/storage")
	{
		storage.GET("/disks", getDisks)
		storage.GET("/usage", getFilesystemUsage)
		storage.GET("/browse", browseDirectory)
		storage.POST("/mkdir", createDirectory)
		storage.DELETE("/delete", deleteItem)
	}
}

// StorageConfig represents the storage configuration
type StorageConfig struct {
	Volumes  []Volume `json:"volumes"`
	Settings struct {
		DefaultPath string `json:"defaultPath"`
	} `json:"settings"`
}

const configPath = "/etc/projecthorizon/config/storage.json"

func loadStorageConfig() (*StorageConfig, error) {
	// Try system config first
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Fallback to local config for development
		data, err = os.ReadFile("config/storage.json")
		if err != nil {
			// Return default config
			return &StorageConfig{
				Volumes: []Volume{},
				Settings: struct {
					DefaultPath string `json:"defaultPath"`
				}{DefaultPath: "/media"},
			}, nil
		}
	}

	var config StorageConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func getDisks(c *gin.Context) {
	partitions, err := disk.Partitions(true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type DiskPartition struct {
		Device     string `json:"device"`
		Mountpoint string `json:"mountpoint"`
		Fstype     string `json:"fstype"`
		Opts       string `json:"opts"`
	}

	var result []DiskPartition
	for _, p := range partitions {
		result = append(result, DiskPartition{
			Device:     p.Device,
			Mountpoint: p.Mountpoint,
			Fstype:     p.Fstype,
			Opts:       p.Opts[0],
		})
	}

	c.JSON(http.StatusOK, result)
}

func getFilesystemUsage(c *gin.Context) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type FsUsage struct {
		Filesystem  string  `json:"filesystem"`
		Mount       string  `json:"mount"`
		Type        string  `json:"type"`
		Total       uint64  `json:"total"`
		Used        uint64  `json:"used"`
		Available   uint64  `json:"available"`
		UsedPercent float64 `json:"usedPercent"`
	}

	var result []FsUsage
	for _, p := range partitions {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		result = append(result, FsUsage{
			Filesystem:  p.Device,
			Mount:       p.Mountpoint,
			Type:        p.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Available:   usage.Free,
			UsedPercent: usage.UsedPercent,
		})
	}

	c.JSON(http.StatusOK, result)
}

// FileItem represents a file or directory
type FileItem struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	IsDirectory bool   `json:"isDirectory"`
	Size        int64  `json:"size"`
	Modified    int64  `json:"modified"`
	Permissions string `json:"permissions"`
}

type BrowseResponse struct {
	CurrentPath string     `json:"currentPath"`
	Parent      string     `json:"parent"`
	Items       []FileItem `json:"items"`
}

func browseDirectory(c *gin.Context) {
	config, _ := loadStorageConfig()
	requestedPath := c.Query("path")
	if requestedPath == "" {
		requestedPath = config.Settings.DefaultPath
	}

	// Check if path exists
	info, err := os.Stat(requestedPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Path not found"})
		return
	}

	if !info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is not a directory"})
		return
	}

	entries, err := os.ReadDir(requestedPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var items []FileItem
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		items = append(items, FileItem{
			Name:        entry.Name(),
			Path:        filepath.Join(requestedPath, entry.Name()),
			IsDirectory: entry.IsDir(),
			Size:        info.Size(),
			Modified:    info.ModTime().Unix(),
			Permissions: info.Mode().String(),
		})
	}

	// Sort: directories first, then by name
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDirectory != items[j].IsDirectory {
			return items[i].IsDirectory
		}
		return items[i].Name < items[j].Name
	})

	c.JSON(http.StatusOK, BrowseResponse{
		CurrentPath: requestedPath,
		Parent:      filepath.Dir(requestedPath),
		Items:       items,
	})
}

type MkdirRequest struct {
	Path string `json:"path"`
}

func createDirectory(c *gin.Context) {
	var req MkdirRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := os.MkdirAll(req.Path, fs.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "path": req.Path})
}

type DeleteRequest struct {
	Path string `json:"path"`
}

func deleteItem(c *gin.Context) {
	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := os.RemoveAll(req.Path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "deleted": req.Path})
}
