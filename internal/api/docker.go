package api

import (
	"context"
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
		docker.POST("/containers/:id/:action", containerAction)
		docker.GET("/images", listImages)
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
