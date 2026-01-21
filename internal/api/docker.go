package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

func RegisterDockerRoutes(rg *gin.RouterGroup) {
	docker := rg.Group("/docker")
	{
		docker.GET("/info", getDockerInfo)
		docker.GET("/containers", listContainers)
		docker.GET("/containers/:id", getContainer)
		docker.GET("/containers/:id/logs", getContainerLogs)
		docker.GET("/containers/:id/stats", getContainerStats)
		docker.POST("/containers/:id/:action", containerAction)
		docker.DELETE("/containers/:id", deleteContainer)
		docker.GET("/images", listImages)
		docker.DELETE("/images/:id", deleteImage)
	}
}

func getDockerClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

// DockerInfo represents Docker system information
type DockerInfo struct {
	Containers        int    `json:"containers"`
	ContainersRunning int    `json:"containersRunning"`
	ContainersPaused  int    `json:"containersPaused"`
	ContainersStopped int    `json:"containersStopped"`
	Images            int    `json:"images"`
	ServerVersion     string `json:"serverVersion"`
	MemTotal          int64  `json:"memTotal"`
	CPUs              int    `json:"cpus"`
}

func getDockerInfo(c *gin.Context) {
	cli, err := getDockerClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker not available: " + err.Error()})
		return
	}
	defer cli.Close()

	info, err := cli.Info(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, DockerInfo{
		Containers:        info.Containers,
		ContainersRunning: info.ContainersRunning,
		ContainersPaused:  info.ContainersPaused,
		ContainersStopped: info.ContainersStopped,
		Images:            info.Images,
		ServerVersion:     info.ServerVersion,
		MemTotal:          info.MemTotal,
		CPUs:              info.NCPU,
	})
}

// ContainerInfo represents container information
type ContainerInfo struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	State   string            `json:"state"`
	Status  string            `json:"status"`
	Ports   []ContainerPort   `json:"ports"`
	Created int64             `json:"created"`
}

type ContainerPort struct {
	Private uint16 `json:"private"`
	Public  uint16 `json:"public"`
	Type    string `json:"type"`
}

func listContainers(c *gin.Context) {
	cli, err := getDockerClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker not available"})
		return
	}
	defer cli.Close()

	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var result []ContainerInfo
	for _, cont := range containers {
		name := ""
		if len(cont.Names) > 0 {
			name = cont.Names[0][1:] // Remove leading slash
		}

		ports := make([]ContainerPort, 0)
		for _, p := range cont.Ports {
			ports = append(ports, ContainerPort{
				Private: p.PrivatePort,
				Public:  p.PublicPort,
				Type:    p.Type,
			})
		}

		result = append(result, ContainerInfo{
			ID:      cont.ID[:12],
			Name:    name,
			Image:   cont.Image,
			State:   cont.State,
			Status:  cont.Status,
			Ports:   ports,
			Created: cont.Created,
		})
	}

	c.JSON(http.StatusOK, result)
}

func getContainer(c *gin.Context) {
	id := c.Param("id")

	cli, err := getDockerClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker not available"})
		return
	}
	defer cli.Close()

	container, err := cli.ContainerInspect(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Container not found"})
		return
	}

	c.JSON(http.StatusOK, container)
}

func getContainerLogs(c *gin.Context) {
	id := c.Param("id")

	cli, err := getDockerClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker not available"})
		return
	}
	defer cli.Close()

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "100",
		Timestamps: true,
	}

	logs, err := cli.ContainerLogs(context.Background(), id, options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer logs.Close()

	c.DataFromReader(http.StatusOK, -1, "text/plain", logs, nil)
}

func containerAction(c *gin.Context) {
	id := c.Param("id")
	action := c.Param("action")

	cli, err := getDockerClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker not available"})
		return
	}
	defer cli.Close()

	ctx := context.Background()
	timeout := 10

	switch action {
	case "start":
		err = cli.ContainerStart(ctx, id, container.StartOptions{})
	case "stop":
		err = cli.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout})
	case "restart":
		err = cli.ContainerRestart(ctx, id, container.StopOptions{Timeout: &timeout})
	case "pause":
		err = cli.ContainerPause(ctx, id)
	case "unpause":
		err = cli.ContainerUnpause(ctx, id)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"action":      action,
		"containerId": id,
	})
}

// ImageInfo represents Docker image information
type ImageInfo struct {
	ID      string   `json:"id"`
	Tags    []string `json:"tags"`
	Size    int64    `json:"size"`
	Created int64    `json:"created"`
}

func listImages(c *gin.Context) {
	cli, err := getDockerClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker not available"})
		return
	}
	defer cli.Close()

	images, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var result []ImageInfo
	for _, img := range images {
		id := img.ID
		if len(id) > 19 {
			id = id[7:19] // Remove sha256: prefix and truncate
		}

		result = append(result, ImageInfo{
			ID:      id,
			Tags:    img.RepoTags,
			Size:    img.Size,
			Created: img.Created,
		})
	}

	c.JSON(http.StatusOK, result)
}

// ContainerStats represents container resource usage
type ContainerStats struct {
	CPUPercent    float64 `json:"cpuPercent"`
	MemoryUsage   uint64  `json:"memoryUsage"`
	MemoryLimit   uint64  `json:"memoryLimit"`
	MemoryPercent float64 `json:"memoryPercent"`
	NetworkRx     uint64  `json:"networkRx"`
	NetworkTx     uint64  `json:"networkTx"`
}

func getContainerStats(c *gin.Context) {
	id := c.Param("id")

	cli, err := getDockerClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker not available"})
		return
	}
	defer cli.Close()

	stats, err := cli.ContainerStats(context.Background(), id, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer stats.Body.Close()

	var statsJSON types.StatsJSON
	if err := json.NewDecoder(stats.Body).Decode(&statsJSON); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse stats"})
		return
	}

	// Calculate CPU percentage
	cpuDelta := float64(statsJSON.CPUStats.CPUUsage.TotalUsage - statsJSON.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(statsJSON.CPUStats.SystemUsage - statsJSON.PreCPUStats.SystemUsage)
	cpuPercent := 0.0
	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(len(statsJSON.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}

	// Calculate memory percentage
	memPercent := 0.0
	if statsJSON.MemoryStats.Limit > 0 {
		memPercent = float64(statsJSON.MemoryStats.Usage) / float64(statsJSON.MemoryStats.Limit) * 100.0
	}

	// Calculate network I/O
	var networkRx, networkTx uint64
	for _, net := range statsJSON.Networks {
		networkRx += net.RxBytes
		networkTx += net.TxBytes
	}

	c.JSON(http.StatusOK, ContainerStats{
		CPUPercent:    cpuPercent,
		MemoryUsage:   statsJSON.MemoryStats.Usage,
		MemoryLimit:   statsJSON.MemoryStats.Limit,
		MemoryPercent: memPercent,
		NetworkRx:     networkRx,
		NetworkTx:     networkTx,
	})
}

func deleteContainer(c *gin.Context) {
	id := c.Param("id")

	cli, err := getDockerClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker not available"})
		return
	}
	defer cli.Close()

	// Force remove and remove volumes
	err = cli.ContainerRemove(context.Background(), id, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"containerId": id,
		"message":     "Container removed",
	})
}

func deleteImage(c *gin.Context) {
	id := c.Param("id")

	cli, err := getDockerClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Docker not available"})
		return
	}
	defer cli.Close()

	_, err = cli.ImageRemove(context.Background(), id, types.ImageRemoveOptions{
		Force:         true,
		PruneChildren: true,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"imageId": id,
		"message": "Image removed",
	})
}
