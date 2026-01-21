package api

import (
	"net/http"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

func RegisterServicesRoutes(rg *gin.RouterGroup) {
	services := rg.Group("/services")
	services.Use(AuthMiddleware())
	{
		services.GET("", listServices)
		services.GET("/:name/status", getServiceStatus)
		services.POST("/:name/start", AdminOnly(), startService)
		services.POST("/:name/stop", AdminOnly(), stopService)
		services.POST("/:name/restart", AdminOnly(), restartService)
	}
}

// ServiceInfo represents a systemd service
type ServiceInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	LoadState   string `json:"loadState"`
	ActiveState string `json:"activeState"`
	SubState    string `json:"subState"`
	UnitFile    string `json:"unitFile"`
}

func listServices(c *gin.Context) {
	// Get list of services using systemctl
	cmd := exec.Command("systemctl", "list-units", "--type=service", "--all", "--no-pager", "--plain", "--no-legend")
	output, err := cmd.Output()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list services"})
		return
	}

	var services []ServiceInfo
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			name := strings.TrimSuffix(fields[0], ".service")
			description := ""
			if len(fields) > 4 {
				description = strings.Join(fields[4:], " ")
			}
			
			services = append(services, ServiceInfo{
				Name:        name,
				LoadState:   fields[1],
				ActiveState: fields[2],
				SubState:    fields[3],
				Description: description,
			})
		}
	}

	c.JSON(http.StatusOK, services)
}

func getServiceStatus(c *gin.Context) {
	name := c.Param("name") + ".service"
	
	// Get service status
	cmd := exec.Command("systemctl", "show", name, "--no-pager", 
		"--property=LoadState,ActiveState,SubState,Description,MainPID,MemoryCurrent")
	output, err := cmd.Output()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get service status"})
		return
	}

	status := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			status[parts[0]] = parts[1]
		}
	}

	c.JSON(http.StatusOK, status)
}

func startService(c *gin.Context) {
	name := c.Param("name") + ".service"
	
	cmd := exec.Command("sudo", "systemctl", "start", name)
	if err := cmd.Run(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start service: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Service started",
		"service": name,
	})
}

func stopService(c *gin.Context) {
	name := c.Param("name") + ".service"
	
	cmd := exec.Command("sudo", "systemctl", "stop", name)
	if err := cmd.Run(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop service: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Service stopped",
		"service": name,
	})
}

func restartService(c *gin.Context) {
	name := c.Param("name") + ".service"
	
	cmd := exec.Command("sudo", "systemctl", "restart", name)
	if err := cmd.Run(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restart service: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Service restarted",
		"service": name,
	})
}
