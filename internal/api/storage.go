package api

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/disk"
)

func RegisterStorageRoutes(rg *gin.RouterGroup) {
	storage := rg.Group("/storage")
	{
		storage.GET("/disks", getDisks)
		storage.GET("/usage", getFilesystemUsage)
		storage.GET("/browse", browseDirectory)
		storage.GET("/properties", getFileProperties)
		storage.GET("/file", serveFile)
		storage.POST("/mkdir", createDirectory)
		storage.POST("/mkfile", createFile)
		storage.POST("/copy", copyItem)
		storage.POST("/move", moveItem)
		storage.POST("/rename", renameItem)
		storage.DELETE("/delete", deleteItem)
		storage.POST("/trash", moveToTrash)
		storage.GET("/trash", listTrash)
		storage.POST("/trash/restore", restoreFromTrash)
		storage.DELETE("/trash/empty", emptyTrash)
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

// File properties response
type FileProperties struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	IsDirectory bool   `json:"isDirectory"`
	Size        int64  `json:"size"`
	Modified    int64  `json:"modified"`
	Created     int64  `json:"created"`
	Permissions string `json:"permissions"`
	MimeType    string `json:"mimeType"`
	Extension   string `json:"extension"`
}

func getFileProperties(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path required"})
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	ext := filepath.Ext(path)
	mimeType := getMimeType(ext)

	c.JSON(http.StatusOK, FileProperties{
		Name:        info.Name(),
		Path:        path,
		IsDirectory: info.IsDir(),
		Size:        info.Size(),
		Modified:    info.ModTime().Unix(),
		Created:     info.ModTime().Unix(), // Go doesn't have cross-platform birth time
		Permissions: info.Mode().String(),
		MimeType:    mimeType,
		Extension:   ext,
	})
}

func getMimeType(ext string) string {
	mimeTypes := map[string]string{
		".txt":  "text/plain",
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".webp": "image/webp",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".ogg":  "audio/ogg",
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".mkv":  "video/x-matroska",
		".avi":  "video/x-msvideo",
		".zip":  "application/zip",
		".rar":  "application/x-rar-compressed",
		".7z":   "application/x-7z-compressed",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
	}
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

func serveFile(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path required"})
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	if info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot serve directory"})
		return
	}

	c.File(path)
}

func createFile(c *gin.Context) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := os.WriteFile(req.Path, []byte(req.Content), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "path": req.Path})
}

func copyItem(c *gin.Context) {
	var req struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	info, err := os.Stat(req.Source)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Source not found"})
		return
	}

	if info.IsDir() {
		err = copyDir(req.Source, req.Destination)
	} else {
		err = copyFile(req.Source, req.Destination)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

func moveItem(c *gin.Context) {
	var req struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := os.Rename(req.Source, req.Destination); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func renameItem(c *gin.Context) {
	var req struct {
		Path    string `json:"path"`
		NewName string `json:"newName"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	dir := filepath.Dir(req.Path)
	newPath := filepath.Join(dir, req.NewName)

	if err := os.Rename(req.Path, newPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "newPath": newPath})
}

// Recycle bin functions
var trashDir = "config/recycle-bin"

type TrashItem struct {
	ID           string `json:"id"`
	OriginalPath string `json:"originalPath"`
	Name         string `json:"name"`
	IsDirectory  bool   `json:"isDirectory"`
	Size         int64  `json:"size"`
	DeletedAt    int64  `json:"deletedAt"`
}

func moveToTrash(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	info, err := os.Stat(req.Path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Create trash directory if needed
	if err := os.MkdirAll(trashDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generate unique ID
	id := filepath.Base(req.Path) + "_" + fmt.Sprintf("%d", time.Now().UnixNano())
	trashPath := filepath.Join(trashDir, id)

	// Move to trash
	if err := os.Rename(req.Path, trashPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Save metadata
	meta := TrashItem{
		ID:           id,
		OriginalPath: req.Path,
		Name:         info.Name(),
		IsDirectory:  info.IsDir(),
		Size:         info.Size(),
		DeletedAt:    time.Now().Unix(),
	}
	metaData, _ := json.Marshal(meta)
	os.WriteFile(filepath.Join(trashDir, id+".meta"), metaData, 0644)

	c.JSON(http.StatusOK, gin.H{"success": true, "id": id})
}

func listTrash(c *gin.Context) {
	entries, err := os.ReadDir(trashDir)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, []TrashItem{})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var items []TrashItem
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".meta" {
			data, err := os.ReadFile(filepath.Join(trashDir, entry.Name()))
			if err != nil {
				continue
			}
			var item TrashItem
			if json.Unmarshal(data, &item) == nil {
				items = append(items, item)
			}
		}
	}

	c.JSON(http.StatusOK, items)
}

func restoreFromTrash(c *gin.Context) {
	var req struct {
		ID string `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	metaPath := filepath.Join(trashDir, req.ID+".meta")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found in trash"})
		return
	}

	var item TrashItem
	if err := json.Unmarshal(data, &item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid metadata"})
		return
	}

	trashPath := filepath.Join(trashDir, req.ID)

	// Restore to original location
	if err := os.Rename(trashPath, item.OriginalPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Remove metadata
	os.Remove(metaPath)

	c.JSON(http.StatusOK, gin.H{"success": true, "restored": item.OriginalPath})
}

func emptyTrash(c *gin.Context) {
	if err := os.RemoveAll(trashDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Recreate empty directory
	os.MkdirAll(trashDir, 0755)

	c.JSON(http.StatusOK, gin.H{"success": true})
}
