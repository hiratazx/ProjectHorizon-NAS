package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ShareType string

const (
	ShareTypeSMB ShareType = "smb"
	ShareTypeNFS ShareType = "nfs"
)

type Share struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Type       ShareType `json:"type"`
	ReadOnly   bool      `json:"readOnly"`
	GuestOk    bool      `json:"guestOk"`    // SMB only
	AllowedIPs string    `json:"allowedIPs"` // NFS only
	CreatedAt  time.Time `json:"createdAt"`
}

var (
	sharesFile = "config/shares.json"
	sharesLock sync.RWMutex
	shares     []Share
)

func RegisterSharesRoutes(rg *gin.RouterGroup) {
	group := rg.Group("/shares")
	group.Use(AuthMiddleware())
	{
		group.GET("", listShares)
		group.POST("", AdminOnly(), createShare)
		group.DELETE("/:id", AdminOnly(), deleteShare)
	}

	loadShares()
}

func loadShares() {
	sharesLock.Lock()
	defer sharesLock.Unlock()

	data, err := os.ReadFile(sharesFile)
	if err != nil {
		if os.IsNotExist(err) {
			shares = []Share{}
			return
		}
		return // Ignore errors for now
	}

	json.Unmarshal(data, &shares)
}

func saveShares() error {
	sharesLock.Lock()
	defer sharesLock.Unlock()

	data, err := json.MarshalIndent(shares, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(sharesFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(sharesFile, data, 0644)
}

func listShares(c *gin.Context) {
	sharesLock.RLock()
	defer sharesLock.RUnlock()
	c.JSON(http.StatusOK, shares)
}

func createShare(c *gin.Context) {
	var input struct {
		Name       string    `json:"name"`
		Path       string    `json:"path"`
		Type       ShareType `json:"type"`
		ReadOnly   bool      `json:"readOnly"`
		GuestOk    bool      `json:"guestOk"`
		AllowedIPs string    `json:"allowedIPs"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Name == "" || input.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name and Path are required"})
		return
	}

	share := Share{
		ID:         uuid.New().String(),
		Name:       input.Name,
		Path:       input.Path,
		Type:       input.Type,
		ReadOnly:   input.ReadOnly,
		GuestOk:    input.GuestOk,
		AllowedIPs: input.AllowedIPs,
		CreatedAt:  time.Now(),
	}

	sharesLock.Lock()
	shares = append(shares, share)
	sharesLock.Unlock()

	if err := saveShares(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save share"})
		return
	}

	c.JSON(http.StatusCreated, share)
}

func deleteShare(c *gin.Context) {
	id := c.Param("id")

	sharesLock.Lock()
	found := false
	newShares := make([]Share, 0)
	for _, s := range shares {
		if s.ID == id {
			found = true
		} else {
			newShares = append(newShares, s)
		}
	}
	shares = newShares
	sharesLock.Unlock()

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Share not found"})
		return
	}

	if err := saveShares(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save changes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
