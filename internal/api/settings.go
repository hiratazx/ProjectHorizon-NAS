package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// Volume represents a storage volume mapping
type Volume struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	HostPath  string `json:"hostPath"`
	MountPath string `json:"mountPath"`
	Mode      string `json:"mode"`
}

// StorageSettings represents the storage configuration
type StorageSettings struct {
	Volumes  []Volume `json:"volumes"`
	Settings struct {
		DefaultPath string `json:"defaultPath"`
	} `json:"settings"`
}

func RegisterSettingsRoutes(rg *gin.RouterGroup) {
	settings := rg.Group("/settings")
	{
		settings.GET("/volumes", getVolumes)
		settings.POST("/volumes", addVolume)
		settings.PUT("/volumes/:id", updateVolume)
		settings.DELETE("/volumes/:id", deleteVolume)
	}
}

func getConfigPath() string {
	paths := []string{
		"/etc/projecthorizon/config/storage.json",
		"config/storage.json",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "config/storage.json"
}

func loadSettings() (*StorageSettings, error) {
	configPath := getConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &StorageSettings{
				Volumes: []Volume{},
				Settings: struct {
					DefaultPath string `json:"defaultPath"`
				}{DefaultPath: "/media"},
			}, nil
		}
		return nil, err
	}

	var settings StorageSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

func saveSettings(settings *StorageSettings) error {
	configPath := getConfigPath()
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func generateVolumeID() string {
	return "vol-" + generateRandomString(8)
}

func generateRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}

func getVolumes(c *gin.Context) {
	settings, err := loadSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, settings.Volumes)
}

// AddVolumeRequest for adding a new volume
type AddVolumeRequest struct {
	Name      string `json:"name" binding:"required"`
	HostPath  string `json:"hostPath" binding:"required"`
	MountPath string `json:"mountPath" binding:"required"`
	Mode      string `json:"mode"` // "rw" or "ro", defaults to "rw"
}

func addVolume(c *gin.Context) {
	var req AddVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate paths
	if req.HostPath == "" || req.MountPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Host path and mount path are required"})
		return
	}

	// Check if host path exists
	if _, err := os.Stat(req.HostPath); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Host path does not exist"})
		return
	}

	if req.Mode == "" {
		req.Mode = "rw"
	}

	settings, err := loadSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check for duplicate mount path
	for _, v := range settings.Volumes {
		if v.MountPath == req.MountPath {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mount path already in use"})
			return
		}
	}

	volume := Volume{
		ID:        generateVolumeID(),
		Name:      req.Name,
		HostPath:  req.HostPath,
		MountPath: req.MountPath,
		Mode:      req.Mode,
	}

	settings.Volumes = append(settings.Volumes, volume)

	if err := saveSettings(settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, volume)
}

func updateVolume(c *gin.Context) {
	id := c.Param("id")

	var req AddVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settings, err := loadSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	found := false
	for i, v := range settings.Volumes {
		if v.ID == id {
			settings.Volumes[i].Name = req.Name
			settings.Volumes[i].HostPath = req.HostPath
			settings.Volumes[i].MountPath = req.MountPath
			if req.Mode != "" {
				settings.Volumes[i].Mode = req.Mode
			}
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Volume not found"})
		return
	}

	if err := saveSettings(settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Volume updated"})
}

func deleteVolume(c *gin.Context) {
	id := c.Param("id")

	settings, err := loadSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	found := false
	newVolumes := make([]Volume, 0)
	for _, v := range settings.Volumes {
		if v.ID == id {
			found = true
		} else {
			newVolumes = append(newVolumes, v)
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Volume not found"})
		return
	}

	settings.Volumes = newVolumes

	if err := saveSettings(settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Volume deleted"})
}
